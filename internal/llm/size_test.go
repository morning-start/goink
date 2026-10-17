package llm

import (
	"strconv"
	"testing"
)

// TestParseContextWindow 验证上下文长度字符串解析。
//
// 核心语义（与按 1000 进位的普通数字不同）：
//   - 带 K/M/G 单位时按 1024 二进制进位
//   - 纯数字不做任何换算，即最终 token 数
func TestParseContextWindow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		// 纯数字：原样使用，不换算
		{"plain integer", "200000", 200000},
		{"plain small integer", "8192", 8192},
		{"plain integer with spaces", "  200000  ", 200000},

		// K = 1024
		{"1K", "1K", 1024},
		{"128K", "128K", 131072},
		{"lowercase k", "128k", 131072},
		{"K with space", "128 K", 131072},
		{"200K", "200K", 204800},

		// M = 1024^2
		{"1M", "1M", 1048576},
		{"lowercase m", "1m", 1048576},
		{"2M", "2M", 2097152},

		// 小数
		{"decimal 1.5M", "1.5M", 1572864},
		{"decimal 1.5K", "1.5K", 1536},
		{"decimal 0.5M", "0.5M", 524288},

		// G = 1024^3
		{"1G", "1G", 1073741824},

		// Ki / Mi / Gi 二进制后缀
		{"128Ki", "128Ki", 131072},
		{"1Mi", "1Mi", 1048576},
		{"128KiB", "128KiB", 131072},
		{"1MiB", "1MiB", 1048576},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContextWindow(tt.input)
			if err != nil {
				t.Fatalf("ParseContextWindow(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseContextWindow(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseContextWindow_Errors 验证非法输入被拒绝。
func TestParseContextWindow_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"only unit", "K"},
		{"only unit M", "M"},
		{"non numeric", "abc"},
		{"unknown unit", "128T"},
		{"zero", "0"},
		{"zero with unit", "0K"},
		{"negative", "-1K"},
		{"too large", "99999G"},
		// 科学计数法：ParseFloat 能解析成 100000，但语义意外，必须拒绝
		{"scientific notation", "1e5"},
		{"scientific notation with unit", "1e5K"},
		// 非法单位后缀：B 不是 token 单位
		{"byte suffix", "128B"},
		{"kb suffix", "128KB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseContextWindow(tt.input)
			if err == nil {
				t.Errorf("ParseContextWindow(%q) expected error, got %d", tt.input, got)
			}
		})
	}
}

// TestFormatContextWindow 验证格式化回显。
func TestFormatContextWindow(t *testing.T) {
	tests := []struct {
		tokens int
		want   string
	}{
		{0, ""},
		{-1, ""},
		{1048576, "1M"},
		{2097152, "2M"},
		{131072, "128K"},
		{1024, "1K"},
		{1073741824, "1G"},
		// 不能被 1024 整除时保留纯数字，避免丢失精度
		{200000, "200000"},
		{1500, "1500"},
	}
	for _, tt := range tests {
		if got := FormatContextWindow(tt.tokens); got != tt.want {
			t.Errorf("FormatContextWindow(%d) = %q, want %q", tt.tokens, got, tt.want)
		}
	}
}

// TestParseFormatContextWindow_RoundTrip 验证格式化后能被解析回原值。
func TestParseFormatContextWindow_RoundTrip(t *testing.T) {
	values := []int{1024, 131072, 1048576, 2097152, 1572864}
	for _, v := range values {
		s := FormatContextWindow(v)
		if s == "" {
			// 不能被 1024 整除的值格式化为纯数字，也应能解析回来
			s = strconv.Itoa(v)
		}
		got, err := ParseContextWindow(s)
		if err != nil {
			t.Errorf("round trip %d → %q failed to parse: %v", v, s, err)
			continue
		}
		if got != v {
			t.Errorf("round trip %d → %q → %d mismatch", v, s, got)
		}
	}
}
