# 模型配置输入优化：Base URL 与上下文长度

> 状态：已实现。对应分支 `feat/model-config-input`。

## 背景与目标

模型配置表单原先要求用户填写完整的请求端点（如
`https://api.deepseek.com/v1/chat/completions`），且上下文窗口只能输入纯数字，
需要用户自己在心里做单位换算。两处都容易填错。

本方案让用户：

1. 只填 base URL（如 `https://api.deepseek.com/v1`），请求端点由系统拼接；
2. 用 `128K`、`1M` 这类可读写法填写上下文窗口，系统换算成 token 数。

本方案只覆盖 chat（`chat/completions`）协议。`messages`、`responses` 协议未实现，
后续接入需新增各自的请求构造器与 SSE 解析器。

## Base URL 与端点派生

### 语义变更

`Provider.ChatURL` 的语义由「完整请求端点」改为「base URL」。请求时由
`DeriveChatEndpoint` 拼出实际端点：

```text
Providers 里存：  https://api.deepseek.com/v1
实际请求：        https://api.deepseek.com/v1/chat/completions
```

`internal/llm/baseurl.go` 提供两个函数：

| 函数 | 职责 |
| --- | --- |
| `NormalizeBaseURL` | 把用户输入规范成 base URL：补 `https://`、去尾斜杠、剥离 `/chat/completions`、裸域名补 `/v1` |
| `DeriveChatEndpoint` | 由 base URL 派生 `{base}/chat/completions`；已是完整端点则原样返回 |

`NormalizeBaseURL` 是幂等的，因此**旧配置无需数据迁移**：已保存的完整端点
经一次规范化即得到 base URL，其后再规范化结果不变。

### 调用点

端点派生后，三处请求路径保持一致：

| 位置 | 用途 |
| --- | --- |
| `internal/llm/stream.go` `ChatStream` | 流式对话 |
| `internal/llm/generate.go` `GenerateText` | 非流式生成 |
| `internal/llm/discover.go` `DiscoverModels` | `/models` 自动发现（复用 `NormalizeBaseURL` 推导 `{base}/models`） |

内置 provider 全部以 base URL 形式存储。`TestBuiltinProviders_UseBaseURL`
会在新增内置 provider 误写成完整端点时失败，防止回归。

### 连通性测试

`expandChatURLCandidates` 保留原六档候选与探测顺序，用户填错路径时仍能自愈。
候选 2 保持字面拼接而非调用 `DeriveChatEndpoint`，原因：裸域名的 `/v1` 补全由
候选 3 负责，若改用派生函数会丢掉「根路径 + `/chat/completions`」这一兜底
（少数服务的端点不带 `/v1`）。

## 上下文长度输入

### 进位规则

`K`/`M`/`G` 按 **1024** 二进制进位，与文件大小按 1000 进位的进位方式不同：

| 输入 | 结果 | 说明 |
| --- | --- | --- |
| `200000` | 200000 | 纯数字原样使用，不做换算 |
| `128K` | 131072 | 128 × 1024 |
| `1M` | 1048576 | 1 × 1024 × 1024 |
| `1.5M` | 1572864 | 支持小数 |
| `128Ki` | 131072 | `Ki`/`Mi`/`Gi` 等价于 `K`/`M`/`G` |

纯数字不做换算，因为用户写下的整数就是他想要的长度。大小写不敏感，
容忍数字与单位间的空格（`128 K`）。

### 解析方

**前端是唯一解析方**。`frontend/src/utils/contextWindow.ts` 解析字符串得到整数，
经表单写入 `llm.ModelInfo.context_window`；后端只接收解析好的数值，不解析字符串。

解析器用显式正则拆分数字与单位，而非 `TrimRight` + `ParseFloat`：
后者会把科学计数法（`1e5`）当作有效输入解析为 100000。

解析失败返回错误码（而非文案），文案在 i18n 的 `settings.contextWindowError.*`
下，避免在 `.ts` 源文件里硬编码中文（`npm run i18n:check` 会校验）。

### 表单行为

`ModelEditForm` 的上下文窗口改为文本输入：

- 编辑期间只更新本地字符串草稿，不回写 `model.context_window`，
  避免半截输入被 `parseInt` 破坏；
- 失焦或提交时解析；成功后回显规范化值（`1048576` → `1M`），
  失败时红字提示且不阻塞继续输入。

`formatContextWindow` 在不能被 1024 整除时保留纯数字（`200000` → `200000`），
避免取整丢精度。

## 涉及文件

| 文件 | 变更 |
| --- | --- |
| `internal/llm/baseurl.go` | 新增：`NormalizeBaseURL`、`DeriveChatEndpoint`、`hostOf` |
| `internal/llm/baseurl_test.go` | 新增：规范化、幂等、端点派生、内置 provider 回归 |
| `internal/llm/discover.go` | 改用 `NormalizeBaseURL`；删除重复的 `extractBaseURL` |
| `internal/llm/stream.go`、`generate.go` | 改用 `DeriveChatEndpoint` |
| `internal/llm/test_connection.go` | 候选生成改用 `hostOf` |
| `internal/llm/providers.go` | 9 个内置 provider 改为 base URL |
| `frontend/src/utils/contextWindow.ts` | 新增：解析与格式化 |
| `frontend/src/components/settings/ModelEditForm.tsx` | 上下文窗口改文本输入 |
| `frontend/src/components/settings/ProviderFormFields.tsx`、`BuiltinProviderPane.tsx` | Base URL 提示、统一用 `formatContextWindow` |
| `frontend/src/i18n/locales/{zh-CN,en}.json` | Base URL 文案、上下文长度占位与错误文案 |

## 已知限制

- 只支持 chat 协议；`messages`/`responses` 未实现。
- 上下文长度上限为 int32（2147483647），由前端校验。
- 前端解析器不做单位换算的国际化（`K`/`M` 后缀固定为拉丁字母）。
