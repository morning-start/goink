package llm

import "testing"

// TestNormalizeBaseURL 验证把用户输入规范化为 base URL 的逻辑。
// NormalizeBaseURL 是 DiscoverModels 推导 /models 端点和 ChatStream 派生
// /chat/completions 的共同基础，必须幂等（旧配置的完整端点再规范化一次不变）。
func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"full endpoint", "https://x.com/v1/chat/completions", "https://x.com/v1"},
		{"base url with /v1", "https://x.com/v1", "https://x.com/v1"},
		{"bare domain with scheme", "https://x.com", "https://x.com/v1"},
		{"bare domain no scheme", "1024token.club", "https://1024token.club/v1"},
		{"trailing slash on /v1", "https://x.com/v1/", "https://x.com/v1"},
		{"trailing slash on full endpoint", "https://x.com/v1/chat/completions/", "https://x.com/v1"},
		{"custom path with /chat/completions", "https://x.com/api/openai/chat/completions", "https://x.com/api/openai"},
		{"custom path without /chat/completions", "https://x.com/api/openai", "https://x.com/api/openai"},
		{"with surrounding space", "  https://x.com/v1  ", "https://x.com/v1"},
		{"http scheme kept", "http://localhost:8080/v1", "http://localhost:8080/v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeBaseURL(tt.raw); got != tt.want {
				t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestNormalizeBaseURL_Idempotent 验证规范化幂等：
// 旧配置保存的完整端点经一次规范化后再规范化结果不变，因此无需数据迁移。
func TestNormalizeBaseURL_Idempotent(t *testing.T) {
	inputs := []string{
		"https://api.deepseek.com/v1/chat/completions",
		"https://opencode.ai/zen/v1/chat/completions",
		"https://x.com/api/openai/chat/completions",
		"1024token.club",
		"https://x.com",
	}
	for _, in := range inputs {
		once := NormalizeBaseURL(in)
		twice := NormalizeBaseURL(once)
		if once != twice {
			t.Errorf("not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}

// TestDeriveChatEndpoint 验证由 base URL 派生 chat 补全端点。
func TestDeriveChatEndpoint(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"base url", "https://api.deepseek.com/v1", "https://api.deepseek.com/v1/chat/completions"},
		{"bare domain", "https://x.com", "https://x.com/v1/chat/completions"},
		{"bare domain no scheme", "1024token.club", "https://1024token.club/v1/chat/completions"},
		{"already full endpoint", "https://x.com/v1/chat/completions", "https://x.com/v1/chat/completions"},
		{"custom path", "https://x.com/api/openai", "https://x.com/api/openai/chat/completions"},
		{"http scheme", "http://localhost:8080/v1", "http://localhost:8080/v1/chat/completions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveChatEndpoint(tt.raw); got != tt.want {
				t.Errorf("DeriveChatEndpoint(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestBuiltinProviders_UseBaseURL 验证内置 provider 全部以 base URL 形式存储，
// 不再硬编码 /chat/completions 后缀（端点统一由 DeriveChatEndpoint 派生）。
func TestBuiltinProviders_UseBaseURL(t *testing.T) {
	for key, p := range Builtin {
		if p.ChatURL == "" {
			t.Errorf("builtin provider %q has empty ChatURL", key)
			continue
		}
		if got := NormalizeBaseURL(p.ChatURL); got != p.ChatURL {
			t.Errorf("builtin provider %q ChatURL %q is not a base URL (normalized: %q)", key, p.ChatURL, got)
		}
		if derived := DeriveChatEndpoint(p.ChatURL); derived == p.ChatURL {
			t.Errorf("builtin provider %q derives no endpoint from %q", key, p.ChatURL)
		}
	}
}
