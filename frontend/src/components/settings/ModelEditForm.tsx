import { useState } from "react";
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { llm } from "@/lib/wailsjs/go/models";
import {
  formatContextWindow,
  parseContextWindow,
} from "@/utils/contextWindow";

interface Props {
  model: llm.ModelInfo;
  onChange: (patch: Partial<llm.ModelInfo>) => void;
  onSave: () => void;
  onCancel: () => void;
  title?: string;
}

export default function ModelEditForm({
  model,
  onChange,
  onSave,
  onCancel,
  title,
}: Props) {
  const { t } = useTranslation();
  const [error, setError] = useState("");
  // 上下文长度以字符串编辑（支持 "128K"/"1M"），
  // 输入过程中不回写 model.context_window，避免半截输入被 parseInt 破坏；
  // 解析成功才同步回 model，保证 "1M" 这类输入可见且可编辑。
  const [contextDraft, setContextDraft] = useState(() =>
    formatContextWindow(model.context_window),
  );
  const [contextError, setContextError] = useState("");

  // 解析草稿并回写 model.context_window。返回是否合法。
  const commitContextDraft = (): boolean => {
    const parsed = parseContextWindow(contextDraft);
    if (!parsed.ok) {
      setContextError(parsed.error);
      return false;
    }
    setContextError("");
    onChange({ context_window: parsed.tokens });
    // 回显规范化值（1048576 → "1M"）
    setContextDraft(formatContextWindow(parsed.tokens));
    return true;
  };

  const handleSave = () => {
    if (!model.id.trim()) {
      setError(t("settings.modelIdRequired"));
      return;
    }
    if (!model.name.trim()) {
      setError(t("settings.modelNameRequired"));
      return;
    }
    // 提交前解析上下文长度，非法则拦截（与后端 ParseContextWindow 同规则）
    const parsed = parseContextWindow(contextDraft);
    if (!parsed.ok) {
      setContextError(parsed.error);
      return;
    }
    if (!model.max_output_tokens || model.max_output_tokens <= 0) {
      setError(t("settings.maxOutputPositive"));
      return;
    }
    setError("");
    setContextError("");
    onChange({ context_window: parsed.tokens });
    onSave();
  };

  const handleChange = (patch: Partial<llm.ModelInfo>) => {
    setError("");
    onChange(patch);
  };

  return (
    <div className="border rounded-md p-3 space-y-2">
      {title && (
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">{title}</span>
          <button
            onClick={onCancel}
            className="text-muted-foreground hover:text-destructive"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      )}
      <div>
        <label className="text-xs text-muted-foreground mb-0.5 block">
          {t("settings.modelId")}
        </label>
        <input
          value={model.id}
          onChange={(e) => handleChange({ id: e.target.value })}
          className="w-full h-8 rounded-md border bg-background px-2.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        />
      </div>
      <div>
        <label className="text-xs text-muted-foreground mb-0.5 block">
          {t("common.name")}
        </label>
        <input
          value={model.name}
          onChange={(e) => handleChange({ name: e.target.value })}
          className="w-full h-8 rounded-md border bg-background px-2.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div>
          <label className="text-xs text-muted-foreground mb-0.5 block">
            {t("settings.contextWindow")}
          </label>
          <input
            value={contextDraft}
            onChange={(e) => {
              setContextDraft(e.target.value);
              setContextError("");
            }}
            onBlur={commitContextDraft}
            placeholder={t("settings.contextWindowPlaceholder")}
            className="w-full h-8 rounded-md border bg-background px-2.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          />
          {contextError && (
            <div className="text-xs text-red-500 mt-0.5">{contextError}</div>
          )}
        </div>
        <div>
          <label className="text-xs text-muted-foreground mb-0.5 block">
            {t("settings.maxOutput")}
          </label>
          <input
            type="number"
            value={model.max_output_tokens || ""}
            onChange={(e) =>
              handleChange({
                max_output_tokens: parseInt(e.target.value, 10) || 0,
              })
            }
            className="w-full h-8 rounded-md border bg-background px-2.5 text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          />
        </div>
      </div>
      <div className="text-xs text-muted-foreground">
        {t("settings.modelDefaultValues")}
      </div>
      <div className="flex items-center gap-3">
        <label className="flex items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={model.supports_thinking}
            onChange={(e) =>
              handleChange({ supports_thinking: e.target.checked })
            }
            className="rounded"
          />
          {t("settings.supportDeepThinking")}
        </label>
        <label className="flex items-center gap-1 text-xs">
          <input
            type="checkbox"
            checked={model.supports_vision}
            onChange={(e) =>
              handleChange({ supports_vision: e.target.checked })
            }
            className="rounded"
          />
          {t("settings.vision")}
        </label>
      </div>
      {error && <div className="text-xs text-red-500">{error}</div>}
      <div className="flex justify-end gap-2 pt-1">
        <button
          onClick={onCancel}
          className="h-8 px-3 rounded-md border text-xs text-muted-foreground"
        >
          {t("common.cancel")}
        </button>
        <button
          onClick={handleSave}
          className="h-8 px-3 rounded-md bg-primary text-primary-foreground text-xs"
        >
          {t("common.confirm")}
        </button>
      </div>
    </div>
  );
}
