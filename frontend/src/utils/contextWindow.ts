// 上下文长度解析：与 Go 端 internal/llm/size.go 的 ParseContextWindow 保持同一套规则。
//
// 进位规则（与按 1000 进位的普通数字不同）：
//   - K = 1024
//   - M = 1024 * 1024 = 1,048,576
//   - G = 1024 * 1024 * 1024
//
// 用户只填纯数字时不套用任何单位换算，该数字即最终的上下文长度。
// 前端做即时校验与回显，后端在保存时仍会以同一规则解析，两者必须一致。

const UNIT_K = 1024;
const UNIT_M = 1024 * 1024;
const UNIT_G = 1024 * 1024 * 1024;

const MAX_TOKENS = 2147483647; // int32 上限，与 Go 端一致

/**
 * 把用户输入的上下文长度字符串解析为 token 数。
 *
 * 支持：`200000`（纯数字原样）、`128K`、`1M`、`1.5M`、`128Ki`、`1MiB`、
 * `128 K`（容忍空格），大小写不敏感。
 *
 * 解析失败返回 `{ ok: false, error }`，不抛异常，便于表单直接展示错误。
 */
export function parseContextWindow(
  input: string,
): { ok: true; tokens: number } | { ok: false; error: string } {
  const raw = (input ?? "").trim();
  if (!raw) {
    return { ok: false, error: "上下文长度不能为空" };
  }

  // 去掉数字与单位之间的空白，并统一大写
  const s = raw.replace(/\s+/g, "").toUpperCase();

  // 拆出数字部分与单位部分
  const match = /^([0-9]*\.?[0-9]+)([A-Z]*)$/.exec(s);
  if (!match) {
    return {
      ok: false,
      error: `无法识别上下文长度「${input}」，请输入如 128K、1M 或纯数字`,
    };
  }

  const value = Number(match[1]);
  if (!Number.isFinite(value)) {
    return {
      ok: false,
      error: `无法识别上下文长度「${input}」，请输入如 128K、1M 或纯数字`,
    };
  }
  if (value <= 0) {
    return { ok: false, error: "上下文长度必须大于 0" };
  }

  const multiplier = unitMultiplier(match[2]);
  if (multiplier === null) {
    return {
      ok: false,
      error: `无法识别单位「${match[2]}」，可用 K、M、G 或纯数字`,
    };
  }

  const tokens = Math.round(value * multiplier);
  if (tokens > MAX_TOKENS) {
    return { ok: false, error: `上下文长度「${input}」过大` };
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
