import { useState } from "react";
import { api } from "../api.js";
import ProviderForm from "./ProviderForm.jsx";
import KeyForm from "./KeyForm.jsx";
import KeyRow from "./KeyRow.jsx";
import ProviderUrlLink from "./ProviderUrlLink.jsx";
import { ConfirmButton } from "./ui.jsx";
import { summarizeSync } from "../format.js";

export default function ProviderCard({ provider, toast, onUnauthorized, onChanged }) {
  const [expanded, setExpanded] = useState(false);
  const [editing, setEditing] = useState(false);
  const [keys, setKeys] = useState(null); // null = 未加载
  const [loadingKeys, setLoadingKeys] = useState(false);
  const [addingKey, setAddingKey] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const handleError = (err) => {
    if (err && err.status === 401) return onUnauthorized();
    toast.error(err.message || "操作失败");
  };

  const loadKeys = async () => {
    setLoadingKeys(true);
    try {
      const res = await api.listKeys(provider.id);
      setKeys(res.keys || []);
    } catch (err) {
      handleError(err);
    } finally {
      setLoadingKeys(false);
    }
  };

  const toggle = () => {
    const next = !expanded;
    setExpanded(next);
    if (next && keys === null) loadKeys();
  };

  const saveProvider = async (payload) => {
    try {
      await api.updateProvider(provider.id, payload);
      toast.success("供应商已更新");
      setEditing(false);
      onChanged();
    } catch (err) {
      handleError(err);
    }
  };

  const removeProvider = async () => {
    try {
      await api.deleteProvider(provider.id);
      toast.success("供应商已删除");
      onChanged();
    } catch (err) {
      handleError(err);
    }
  };

  const createKey = async (payload) => {
    try {
      await api.createKey(provider.id, payload);
      toast.success("Key 已添加");
      setAddingKey(false);
      loadKeys();
    } catch (err) {
      handleError(err);
    }
  };

  const updateKey = (keyId) => async (payload) => {
    try {
      await api.updateKey(provider.id, keyId, payload);
      toast.success("Key 已更新");
      loadKeys();
    } catch (err) {
      handleError(err);
    }
  };

  const deleteKey = (keyId) => async () => {
    try {
      await api.deleteKey(provider.id, keyId);
      toast.success("Key 已删除");
      loadKeys();
    } catch (err) {
      handleError(err);
    }
  };

  const syncKey = (keyId) => async () => {
    try {
      const res = await api.syncKey(provider.id, keyId);
      const sum = summarizeSync(res.results);
      sum.ok ? toast.success(sum.message) : toast.error(sum.message);
      loadKeys();
    } catch (err) {
      handleError(err);
    }
  };

  const syncProvider = async () => {
    setSyncing(true);
    try {
      const res = await api.syncProvider(provider.id);
      const sum = summarizeSync(res.results);
      sum.ok ? toast.success(sum.message) : toast.error(sum.message);
      if (keys !== null) loadKeys();
    } catch (err) {
      handleError(err);
    } finally {
      setSyncing(false);
    }
  };

  if (editing) {
    return (
      <div className="card">
        <ProviderForm initial={provider} onSubmit={saveProvider} onCancel={() => setEditing(false)} />
      </div>
    );
  }

  return (
    <div className="card">
      <div className="card-head">
        <button className="card-toggle" onClick={toggle} aria-expanded={expanded}>
          <span className={expanded ? "caret caret-open" : "caret"}>▶</span>
          <span className="provider-name">{provider.name}</span>
          {!provider.enabled && <span className="badge badge-muted">已禁用</span>}
        </button>
        <div className="card-actions">
          <button className="btn btn-sm" onClick={syncProvider} disabled={syncing}>
            {syncing ? "同步中…" : "同步全部 Key"}
          </button>
          <button className="btn btn-sm btn-ghost" onClick={() => setEditing(true)}>
            编辑
          </button>
          <ConfirmButton className="btn-sm" onConfirm={removeProvider} confirmText="确认删除供应商？">
            删除
          </ConfirmButton>
        </div>
      </div>
      <div className="provider-sub">
        <ProviderUrlLink url={provider.base_url} />
        {provider.note && <span className="provider-note">{provider.note}</span>}
      </div>

      {expanded && (
        <div className="card-body">
          {loadingKeys && <div className="muted">加载 Key…</div>}
          {!loadingKeys && keys && keys.length === 0 && !addingKey && (
            <div className="muted">还没有 Key。</div>
          )}
          {!loadingKeys && keys && keys.length > 0 && (
            <ul className="key-list">
              {keys.map((k) => (
                <KeyRow
                  key={k.id}
                  providerId={provider.id}
                  item={k}
                  onUpdate={updateKey(k.id)}
                  onDelete={deleteKey(k.id)}
                  onSync={syncKey(k.id)}
                />
              ))}
            </ul>
          )}
          {addingKey ? (
            <KeyForm onSubmit={createKey} onCancel={() => setAddingKey(false)} />
          ) : (
            <button className="btn btn-sm btn-dashed" onClick={() => setAddingKey(true)}>
              + 添加 Key
            </button>
          )}
        </div>
      )}
    </div>
  );
}
