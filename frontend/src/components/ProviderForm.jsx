import { useState } from "react";

// 供应商创建 / 编辑表单。initial 为空表示创建。
export default function ProviderForm({ initial, onSubmit, onCancel }) {
  const [name, setName] = useState(initial?.name ?? "");
  const [baseURL, setBaseURL] = useState(initial?.base_url ?? "");
  const [note, setNote] = useState(initial?.note ?? "");
  const [enabled, setEnabled] = useState(initial?.enabled ?? true);
  const [busy, setBusy] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      await onSubmit({
        name: name.trim(),
        base_url: baseURL.trim(),
        note: note.trim(),
        enabled,
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <form className="inline-form" onSubmit={submit}>
      <div className="form-grid">
        <label className="field">
          <span>名称</span>
          <input value={name} onChange={(e) => setName(e.target.value)} required placeholder="OpenAI" />
        </label>
        <label className="field">
          <span>Base URL</span>
          <input
            value={baseURL}
            onChange={(e) => setBaseURL(e.target.value)}
            required
            placeholder="https://api.openai.com"
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
