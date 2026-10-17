package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DiscoverModels 调用 /models 端点自动发现可用模型列表。
// 从 chatURL 推导 modelsURL（NormalizeBaseURL 规范化后拼接 /models），
// 解析标准 OpenAI 格式及 Kimi 等扩展字段。返回的 ModelInfo 中未获取到的字段留零值。
func DiscoverModels(ctx context.Context, builtin map[string]Provider, providerName, chatURL, apiKey string) ([]ModelInfo, error) {
	baseURL := NormalizeBaseURL(chatURL)
	modelsURL := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	// 按 provider 名匹配 builtin provider，应用其 BuildHeaders
	// （如 OpenCode 的 x-opencode-session + UA），保证 /models 探测与 chat 请求一致。
	// 未命中（自定义 provider 或内置无 hook）不附加额外头，仅保留鉴权头。
	if p, ok := builtin[providerName]; ok && p.BuildHeaders != nil {
		for k, v := range p.BuildHeaders(nil, map[string]string{}) {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("自动发现模型失败: %s", summarizeErrorBody(resp.StatusCode, errBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if isHTMLPage(body) {
		return nil, fmt.Errorf("该端点不支持自动发现（服务端返回了网页而非 JSON）")
	}

	var result struct {
		Data []struct {
			ID                string `json:"id"`
			ContextLength     int    `json:"context_length"`
			SupportsImageIn   *bool  `json:"supports_image_in"`
			SupportsVideoIn   *bool  `json:"supports_video_in"`
			SupportsReasoning *bool  `json:"supports_reasoning"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析模型列表失败（该端点可能不支持 /models）: %w", err)
	}

	models := make([]ModelInfo, 0, len(result.Data))
	for _, item := range result.Data {
		if item.ID == "" {
			continue
		}
		m := ModelInfo{
			ID:   item.ID,
			Name: modelIDToName(item.ID),
		}
		if item.ContextLength > 0 {
			m.ContextWindow = item.ContextLength
		}
		if item.SupportsReasoning != nil {
			m.SupportsThinking = *item.SupportsReasoning
		}
		if item.SupportsImageIn != nil {
			m.SupportsVision = *item.SupportsImageIn
		} else if item.SupportsVideoIn != nil && *item.SupportsVideoIn {
			m.SupportsVision = true
		}
		models = append(models, m)
	}

	return models, nil
}

// modelIDToName 将模型 ID 转为显示名称：首字母大写，- 替换为空格。
func modelIDToName(id string) string {
	s := strings.ReplaceAll(id, "-", " ")
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
