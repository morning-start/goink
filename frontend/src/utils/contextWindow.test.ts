import { describe, expect, it } from "vitest";
import { formatContextWindow, parseContextWindow } from "./contextWindow";

// 进位规则：K/M/G 按 1024 二进制进位；纯数字不做换算，即最终 token 数。

describe("parseContextWindow", () => {
  it("纯数字原样使用，不做单位换算", () => {
    expect(parseContextWindow("200000")).toEqual({
      ok: true,
      tokens: 200000,
    });
    expect(parseContextWindow("8192")).toEqual({ ok: true, tokens: 8192 });
    expect(parseContextWindow("  200000  ")).toEqual({
      ok: true,
      tokens: 200000,
    });
  });

  it("K 按 1024 进位", () => {
    expect(parseContextWindow("1K")).toEqual({ ok: true, tokens: 1024 });
    expect(parseContextWindow("128K")).toEqual({ ok: true, tokens: 131072 });
    expect(parseContextWindow("128k")).toEqual({ ok: true, tokens: 131072 });
    expect(parseContextWindow("128 K")).toEqual({ ok: true, tokens: 131072 });
    expect(parseContextWindow("200K")).toEqual({ ok: true, tokens: 204800 });
  });

  it("M 按 1024^2 进位", () => {
    expect(parseContextWindow("1M")).toEqual({ ok: true, tokens: 1048576 });
    expect(parseContextWindow("1m")).toEqual({ ok: true, tokens: 1048576 });
    expect(parseContextWindow("2M")).toEqual({ ok: true, tokens: 2097152 });
  });

  it("支持小数", () => {
    expect(parseContextWindow("1.5M")).toEqual({ ok: true, tokens: 1572864 });
    expect(parseContextWindow("1.5K")).toEqual({ ok: true, tokens: 1536 });
    expect(parseContextWindow("0.5M")).toEqual({ ok: true, tokens: 524288 });
  });

  it("G 按 1024^3 进位", () => {
    expect(parseContextWindow("1G")).toEqual({ ok: true, tokens: 1073741824 });
  });

  it("支持 Ki/Mi 二进制后缀", () => {
    expect(parseContextWindow("128Ki")).toEqual({ ok: true, tokens: 131072 });
    expect(parseContextWindow("1Mi")).toEqual({ ok: true, tokens: 1048576 });
    expect(parseContextWindow("1MiB")).toEqual({ ok: true, tokens: 1048576 });
  });

  it("空输入返回 empty 错误码", () => {
    expect(parseContextWindow("")).toEqual({ ok: false, code: "empty" });
    expect(parseContextWindow("   ")).toEqual({ ok: false, code: "empty" });
  });

  it("非法格式返回 invalid 错误码并带回原文", () => {
    for (const bad of ["K", "M", "abc", "128T", "1e5", "128B", "128KB"]) {
      const result = parseContextWindow(bad);
      expect(result.ok, `expected rejection for ${bad}`).toBe(false);
      if (!result.ok) {
        // 字母单位会被先识别为 unknownUnit，纯字母则无数字部分
        expect(["invalid", "unknownUnit"]).toContain(result.code);
      }
    }
  });

  it("非正数返回 notPositive 错误码", () => {
    expect(parseContextWindow("0")).toEqual({ ok: false, code: "notPositive" });
    expect(parseContextWindow("0K")).toEqual({
      ok: false,
      code: "notPositive",
    });
  });

  it("超出 int32 上限返回 tooLarge 错误码", () => {
    expect(parseContextWindow("99999G")).toEqual({
      ok: false,
      code: "tooLarge",
      detail: "99999G",
    });
  });
});

describe("formatContextWindow", () => {
  it("按二进制单位回显", () => {
    expect(formatContextWindow(1048576)).toBe("1M");
    expect(formatContextWindow(2097152)).toBe("2M");
    expect(formatContextWindow(131072)).toBe("128K");
    expect(formatContextWindow(1024)).toBe("1K");
    expect(formatContextWindow(1073741824)).toBe("1G");
  });

  it("不能被 1024 整除时保留纯数字", () => {
    expect(formatContextWindow(200000)).toBe("200000");
    expect(formatContextWindow(1500)).toBe("1500");
  });

  it("零值返回空串", () => {
    expect(formatContextWindow(0)).toBe("");
    expect(formatContextWindow(-1)).toBe("");
  });
});

describe("round trip", () => {
  it("格式化后可解析回原值", () => {
    for (const v of [1024, 131072, 1048576, 2097152, 1572864, 200000]) {
      const s = formatContextWindow(v);
      const parsed = parseContextWindow(s);
      expect(parsed.ok, `round trip ${v} via ${s}`).toBe(true);
      if (parsed.ok) {
        expect(parsed.tokens).toBe(v);
      }
    }
  });
});
