import { useState } from "react";
import KeyForm from "./KeyForm.jsx";
import { ConfirmButton } from "./ui.jsx";
import { formatTime } from "../format.js";

// 单个 Key 行：展示掩码、同步状态，支持编辑 / 删除 / 单独同步。
export default function KeyRow({ providerId, item, onUpdate, onDelete, onSync }) {
  const [editing, setEditing] = useState(false);
  const [syncing, setSyncing] = useState(false);

  if (editing) {
    return (
      <li className="key-row key-row-editing">
        <KeyForm
          initial={item}
          onSubmit={async (payload) => {
            await onUpdate(payload);
            setEditing(false);
          }}
          onCancel={() => setEditing(false)}
        />
      </li>
    );
  }

  const sync = async () => {
    setSyncing(true);
    try {
      await onSync();
    } finally {
      setSyncing(false);
    }
  };

  return (
    <li className="key-row">
      <div className="key-main">
        <div className="key-title">
          <span className="key-name">{item.name}</span>
          <code className="key-mask">{item.api_key_masked}</code>
          {!item.enabled && <span className="badge badge-muted">已禁用</span>}
        </div>
        {item.note && <div className="key-note">{item.note}</div>}
        <div className="key-meta">
          <span>上次同步：{formatTime(item.last_sync_at)}</span>
          {item.last_sync_error && (
            <span className="key-error" title={item.last_sync_error}>
              错误：{item.last_sync_error}
            </span>
          )}
        </div>
      </div>
      <div className="key-actions">
        <button className="btn btn-sm" onClick={sync} disabled={syncing}>
          {syncing ? "同步中…" : "同步"}
        </button>
        <button className="btn btn-sm btn-ghost" onClick={() => setEditing(true)}>
          编辑
        </button>
        <ConfirmButton className="btn-sm" onConfirm={onDelete}>
          删除
        </ConfirmButton>
      </div>
    </li>
  );
}
