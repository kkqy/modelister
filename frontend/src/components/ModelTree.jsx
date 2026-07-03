import { formatTime } from "../format.js";
import ProviderUrlLink from "./ProviderUrlLink.jsx";

// 单个模型条目。
function ModelChip({ model }) {
  return (
    <li className="model-chip">
      <code className="model-id">{model.id}</code>
      {model.owned_by && <span className="model-owner">{model.owned_by}</span>}
    </li>
  );
}

function ModelGrid({ models }) {
  if (!models || models.length === 0) {
    return <div className="muted small">无匹配模型</div>;
  }
  return (
    <ul className="model-grid">
      {models.map((m) => (
        <ModelChip key={m.id} model={m} />
      ))}
    </ul>
  );
}

// 渲染 by_key 或 merged 两种结构。
export default function ModelTree({ mode, providers }) {
  if (!providers || providers.length === 0) {
    return <div className="empty">没有模型数据。先在「供应商」里添加 Key 并同步。</div>;
  }

  return (
    <div className="model-tree">
      {providers.map((p) => (
        <div className="model-provider" key={p.id}>
          <div className="model-provider-head">
            <span className="provider-name">{p.name}</span>
            <ProviderUrlLink url={p.base_url} className="provider-url" />
            {p.note && <span className="provider-note">{p.note}</span>}
          </div>

          {mode === "merged" ? (
            <div className="model-provider-body">
              <ModelGrid models={p.models} />
            </div>
          ) : (
            <div className="model-provider-body">
              {(p.keys || []).map((k) => (
                <div className="model-key" key={k.id}>
                  <div className="model-key-head">
                    <span className="key-name">{k.name}</span>
                    <span className="model-key-meta">
                      {k.models ? k.models.length : 0} 个模型 · 同步于 {formatTime(k.last_sync_at)}
                    </span>
                    {k.last_sync_error && (
                      <span className="key-error" title={k.last_sync_error}>
                        错误：{k.last_sync_error}
                      </span>
                    )}
                  </div>
                  <ModelGrid models={k.models} />
                </div>
              ))}
              {(!p.keys || p.keys.length === 0) && (
                <div className="muted small">该供应商下没有可显示的 Key</div>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
