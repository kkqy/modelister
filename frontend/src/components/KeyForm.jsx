import { useState } from "react";

// Key 创建 / 编辑表单。编辑时 api_key 留空表示保持原 Key 不变。
export default function KeyForm({ initial, onSubmit, onCancel }) {
  const isEdit = Boolean(initial);
  const [name, setName] = useState(initial?.name ?? "");
  const [apiKey, setApiKey] = useState("");
  const [note, setNote] = useState(initial?.note ?? "");
  const [enabled, setEnabled] = useState(initial?.enabled ?? true);
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      await onSubmit({
        name: name.trim(),
        api_key: apiKey,
        note: note.trim(),
        enabled,
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="inline-form inline-form-nested" onSubmit={submit}>
      <div className="form-grid">
        <label className="field">
          <span>名称</span>
          <input value={name} onChange={(e) => setName(e.target.value)} required placeholder="生产 Key" />
        </label>
        <label className="field">
          <span>
            API Key
            {isEdit && <em className="hint"> （留空保持不变）</em>}
          </span>
          <input
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            required={!isEdit}
            placeholder={isEdit ? "••••••••" : "sk-..."}
            autoComplete="off"
          />
        </label>
      </div>
      <label className="field">
        <span>备注</span>
        <input value={note} onChange={(e) => setNote(e.target.value)} placeholder="可选" />
      </label>
      <label className="checkbox">
        <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
        <span>启用</span>
      </label>
      <div className="form-actions">
        <button className="btn btn-primary" type="submit" disabled={busy}>
          {busy ? "保存中…" : "保存"}
        </button>
        <button className="btn btn-ghost" type="button" onClick={onCancel}>
          取消
        </button>
      </div>
    </form>
  );
}
