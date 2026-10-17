// 上下文长度的解析与回显。
//
// 进位规则（与文件大小等按 1000 进位的「普通数字」不同）：
//   - K = 1024
//   - M = 1024 * 1024 = 1,048,576
//   - G = 1024 * 1024 * 1024
//
// 用户只填纯数字时不套用任何单位换算，该数字即最终的上下文长度。
//
// 本模块是上下文长度字符串的唯一解析方：解析出的整数由表单写入
// llm.ModelInfo.context_window，后端只接收解析好的数值，不再解析字符串。
//
// 错误以错误码返回（见 ContextWindowErrorCode），文案由调用方用 t() 翻译，
// 避免在 .ts 源文件里硬编码中文（i18n:check 会校验）。

const UNIT_K = 1024;
const UNIT_M = 1024 * 1024;
const UNIT_G = 1024 * 1024 * 1024;

const MAX_TOKENS = 2147483647; // int32 上限

/** 上下文长度解析失败的原因码，对应 settings.contextWindowError.* 的 i18n key。 */
export type ContextWindowErrorCode =
  | "empty"
  | "invalid"
  | "notPositive"
  | "unknownUnit"
  | "tooLarge";

export type ParseContextWindowResult =
  | { ok: true; tokens: number }
  | { ok: false; code: ContextWindowErrorCode; /** 插入文案的变量：非法输入原文或单位 */ detail?: string };

/**
 * 把用户输入的上下文长度字符串解析为 token 数。
 *
 * 支持：`200000`（纯数字原样）、`128K`、`1M`、`1.5M`、`128Ki`、`1MiB`、
 * `128 K`（容忍空格），大小写不敏感。
 *
 * 解析失败返回错误码，不抛异常，便于表单直接展示。数字与单位用显式
 * 正则拆分，避免把科学计数法（`1e5`）当作有效输入。
 */
export function parseContextWindow(input: string): ParseContextWindowResult {
  const raw = (input ?? "").trim();
  if (!raw) {
    return { ok: false, code: "empty" };
  }

  // 去掉数字与单位之间的空白，并统一大写
  const s = raw.replace(/\s+/g, "").toUpperCase();

  // 拆出数字部分与单位部分（数字只接受十进制小数）
  const match = /^([0-9]*\.?[0-9]+)([A-Z]*)$/.exec(s);
  if (!match) {
    return { ok: false, code: "invalid", detail: input };
  }

  const value = Number(match[1]);
  if (!Number.isFinite(value)) {
    return { ok: false, code: "invalid", detail: input };
  }
  if (value <= 0) {
    return { ok: false, code: "notPositive" };
  }

  const multiplier = unitMultiplier(match[2]);
  if (multiplier === null) {
    return { ok: false, code: "unknownUnit", detail: match[2] };
  }

  const tokens = Math.round(value * multiplier);
  if (tokens > MAX_TOKENS) {
    return { ok: false, code: "tooLarge", detail: input };
  }

  return { ok: true, tokens };
}

/** 单位对应的进位倍数；空单位（纯数字）为 1；未知单位返回 null。 */
function unitMultiplier(unit: string): number | null {
  switch (unit) {
    case "":
      return 1;
    case "K":
    case "KI":
    case "KIB":
      return UNIT_K;
    case "M":
    case "MI":
    case "MIB":
      return UNIT_M;
    case "G":
    case "GI":
    case "GIB":
      return UNIT_G;
    default:
      return null;
  }
}

/**
 * 把 token 数格式化为紧凑的二进制单位表示，与 parseContextWindow 互逆。
 *
 * 例：1048576 → "1M"，131072 → "128K"，200000 → "200000"
 * （不能被 1024 整除时保留纯数字，避免丢失精度）
 */
export function formatContextWindow(tokens: number): string {
  if (!tokens || tokens <= 0) {
    return "";
  }
  const units: Array<{ unit: string; multiplier: number }> = [
    { unit: "G", multiplier: UNIT_G },
    { unit: "M", multiplier: UNIT_M },
    { unit: "K", multiplier: UNIT_K },
  ];
  for (const { unit, multiplier } of units) {
    if (tokens % multiplier === 0) {
      return `${tokens / multiplier}${unit}`;
    }
  }
  return String(tokens);
}
