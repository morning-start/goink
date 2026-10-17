package llm

import (
	"net/url"
	"strings"
)

// chatCompletionsPath 是 OpenAI 兼容 chat 补全端点的固定路径后缀。
const chatCompletionsPath = "/chat/completions"

// NormalizeBaseURL 把用户填的地址规范化为 base URL。
// 用户只需填 base URL（如 https://api.deepseek.com/v1），
// 实际请求端点由 DeriveChatEndpoint 派生，不再要求填完整请求 URL。
//
// 处理步骤：补 https:// 协议头 → TrimRight 末尾斜杠 → 去掉末尾 /chat/completions
// → 裸域名补 /v1。
//
// 幂等：对已是 base URL 的值再调用一次结果不变，因此旧配置里保存的
// 完整端点（如 https://x.com/v1/chat/completions）无需数据迁移即可工作。
//
// 示例：
//   - https://x.com/v1/chat/completions → https://x.com/v1
//   - https://x.com/v1                  → https://x.com/v1
//   - https://x.com                      → https://x.com/v1
//   - 1024token.club                     → https://1024token.club/v1
//   - https://x.com/api/openai          → https://x.com/api/openai
func NormalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	raw = strings.TrimRight(raw, "/")
	if base, ok := strings.CutSuffix(raw, chatCompletionsPath); ok {
		return base
	}
	// 裸域名（host 后无路径）补 /v1
	if _, hostPart, ok := strings.Cut(raw, "://"); ok {
		if !strings.Contains(hostPart, "/") {
			return raw + "/v1"
		}
	}
	return raw
}

// DeriveChatEndpoint 由 base URL 派生 chat 补全请求端点。
// 入参可以已是完整端点（旧配置），此时原样返回，保证向后兼容。
//
// 示例：
//   - https://api.deepseek.com/v1 → https://api.deepseek.com/v1/chat/completions
//   - https://x.com              → https://x.com/v1/chat/completions
//   - 已是完整端点                → 原样返回
func DeriveChatEndpoint(raw string) string {
	base := NormalizeBaseURL(raw)
	if base == "" {
		return base
	}
	if strings.HasSuffix(base, chatCompletionsPath) {
		return base
	}
	return base + chatCompletionsPath
}

// hostOf 提取 URL 的 scheme://host 部分，解析失败时返回空串。
// 用于连通性测试的兜底候选（忽略用户填的路径，直接探测标准端点）。
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
