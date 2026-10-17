package llm

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// contextWindowPattern 拆分数字与单位：数字为十进制小数（不接受科学计数法），
// 单位为可选字母串（空表示纯数字，由 sizeMultiplier 判合法）。
var contextWindowPattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)([a-zA-Z]*)$`)

// 上下文长度单位。以 1024 为进位的二进制单位。
//
// 进位规则（与文件大小等按 1000 进位的「普通数字」不同）：
//   - K = 1024
//   - M = 1024 * 1024 = 1,048,576
//   - G = 1024 * 1024 * 1024 = 1,073,741,824
//
// 用户只填纯数字时不套用任何单位换算，该数字即最终的上下文长度。
const (
	sizeUnitK = 1024
	sizeUnitM = 1024 * 1024
	sizeUnitG = 1024 * 1024 * 1024
)

// ParseContextWindow 把用户输入的上下文长度字符串解析为 token 数。
//
// 支持的写法：
//   - "200000"   → 200000（纯数字按原值使用，不做单位换算）
//   - "128K"     → 131072（128 * 1024）
//   - "1M"       → 1048576（1 * 1024 * 1024）
//   - "1.5M"     → 1572864（支持小数）
//   - "128Ki"    → 131072（Ki / Mi / Gi 等价于 K / M / G）
//   - "1 M"      → 1048576（容忍数字与单位之间的空格）
//
// 大小写不敏感。结果四舍五入到整数 token。
// 空字符串、非法格式、非正数、超出 int32 范围均返回错误。
//
// 用显式正则拆分数字与单位，而不是 TrimRight：
// 科学计数法（"1e5"）等写法会被正则拒绝，避免 ParseFloat 顺带解析出意外值。
func ParseContextWindow(input string) (int, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, fmt.Errorf("上下文长度不能为空")
	}

	// 去掉数字与单位之间可能出现的空格："1 M" → "1M"
	s = strings.Join(strings.Fields(s), "")

	// 拆出数字部分和单位部分（单位可能为空）。
	// 数字部分只接受十进制小数，不接受科学计数法与千分位逗号。
	match := contextWindowPattern.FindStringSubmatch(s)
	if match == nil {
		return 0, fmt.Errorf("无法识别上下文长度 %q，请输入如 128K、1M 或纯数字", input)
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, fmt.Errorf("无法识别上下文长度 %q，请输入如 128K、1M 或纯数字", input)
	}
	if value <= 0 {
		return 0, fmt.Errorf("上下文长度必须大于 0")
	}

	// 单位统一大写后匹配（"k"/"ki"/"kib" 都按二进制处理）
	multiplier, err := sizeMultiplier(strings.ToUpper(match[2]))
	if err != nil {
		return 0, fmt.Errorf("无法识别单位 %q，可用 K、M、G 或纯数字", match[2])
	}

	total := value * float64(multiplier)
	if total > math.MaxInt32 {
		return 0, fmt.Errorf("上下文长度 %q 过大", input)
	}
	return int(math.Round(total)), nil
}

// sizeMultiplier 返回单位对应的进位倍数。空单位（纯数字）为 1。
// "IB" 后缀（如 MiB 的 IB 部分）与 "I" 一样按二进制处理。
func sizeMultiplier(unit string) (int, error) {
	switch unit {
	case "":
		return 1, nil
	case "K", "KI", "KIB":
		return sizeUnitK, nil
	case "M", "MI", "MIB":
		return sizeUnitM, nil
	case "G", "GI", "GIB":
		return sizeUnitG, nil
	default:
		return 0, fmt.Errorf("未知单位: %s", unit)
	}
}

// FormatContextWindow 把 token 数格式化为紧凑的二进制单位表示。
// 用于前端回显规范化后的值，与 ParseContextWindow 互逆。
//
// 示例：
//   - 1048576 → "1M"
//   - 131072  → "128K"
//   - 200000  → "200000"（不能被 1024 整除时保留纯数字，避免丢失精度）
func FormatContextWindow(tokens int) string {
	if tokens <= 0 {
		return ""
	}
	for _, u := range []struct {
		unit       string
		multiplier int
	}{
		{"G", sizeUnitG},
		{"M", sizeUnitM},
		{"K", sizeUnitK},
	} {
		if tokens%u.multiplier == 0 {
			return strconv.Itoa(tokens/u.multiplier) + u.unit
		}
	}
	return strconv.Itoa(tokens)
}
