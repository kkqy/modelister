import { useCallback, useEffect, useState } from "react";
import { api } from "../api.js";
import { Spinner } from "./ui.jsx";
import { summarizeSync } from "../format.js";
import ProviderForm from "./ProviderForm.jsx";
import ProviderCard from "./ProviderCard.jsx";

export default function Providers({ toast, onUnauthorized }) {
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [syncingAll, setSyncingAll] = useState(false);

  const handleError = useCallback(
    (err) => {
      if (err && err.status === 401) return onUnauthorized();
      toast.error(err.message || "操作失败");
    },
    [onUnauthorized, toast]
  );

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.listProviders();
      setProviders(res.providers || []);
    } catch (err) {
      handleError(err);
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  useEffect(() => {
    load();
  }, [load]);

  const create = async (payload) => {
    try {
      await api.createProvider(payload);
      toast.success("供应商已创建");
      setCreating(false);
      load();
    } catch (err) {
      handleError(err);
    }
  };

  const syncAll = async () => {
    setSyncingAll(true);
    try {
      const res = await api.syncAll();
      const sum = summarizeSync(res.results);
      sum.ok ? toast.success(sum.message) : toast.error(sum.message);
    } catch (err) {
      handleError(err);
    } finally {
      setSyncingAll(false);
    }
  };

  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>供应商</h2>
          <p className="panel-sub">管理 OpenAI 兼容供应商及其 API Key</p>
        </div>
        <div className="panel-actions">
          <button className="btn" onClick={syncAll} disabled={syncingAll}>
            {syncingAll ? "同步中…" : "同步全部"}
          </button>
          <button className="btn btn-primary" onClick={() => setCreating((v) => !v)}>
            {creating ? "取消" : "+ 新建供应商"}
          </button>
        </div>
      </div>

      {creating && (
        <div className="card">
          <ProviderForm onSubmit={create} onCancel={() => setCreating(false)} />
        </div>
      )}

      {loading ? (
        <Spinner label="加载供应商…" />
      ) : providers.length === 0 ? (
        <div className="empty">还没有供应商，点击「新建供应商」开始。</div>
      ) : (
        <div className="card-list">
          {providers.map((p) => (
            <ProviderCard
              key={p.id}
              provider={p}
              toast={toast}
              onUnauthorized={onUnauthorized}
              onChanged={load}
            />
          ))}
        </div>
      )}
    </section>
  );
}
