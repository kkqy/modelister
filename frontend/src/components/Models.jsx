import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "../api.js";
import { Spinner } from "./ui.jsx";
import ModelTree from "./ModelTree.jsx";

export default function Models({ toast, onUnauthorized }) {
  const [mode, setMode] = useState("by_key");
  const [query, setQuery] = useState("");
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(false);
  const debounceRef = useRef(null);

  const handleError = useCallback(
    (err) => {
      if (err && err.status === 401) return onUnauthorized();
      toast.error(err.message || "读取模型失败");
    },
    [onUnauthorized, toast]
  );

  // 统一的拉取：有关键词走 search，否则走 list。
  const fetchModels = useCallback(
    async ({ refresh = false } = {}) => {
      setLoading(true);
      try {
        const q = query.trim();
        const res = q
          ? await api.searchModels({ q, mode, refresh })
          : await api.listModels({ mode, refresh });
        setData(res);
        if (refresh) toast.success("已刷新并重新同步");
      } catch (err) {
        handleError(err);
      } finally {
        setLoading(false);
      }
    },
    [query, mode, handleError, toast]
  );

  // mode 切换或关键词变化时自动拉取（关键词加防抖）。
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchModels();
    }, query.trim() ? 350 : 0);
    return () => clearTimeout(debounceRef.current);
  }, [fetchModels, query]);

  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>模型</h2>
          <p className="panel-sub">查看与搜索各供应商 Key 下缓存的模型</p>
        </div>
        <div className="panel-actions">
          <button className="btn" onClick={() => fetchModels({ refresh: true })} disabled={loading}>
            {loading ? "处理中…" : "刷新并同步"}
          </button>
        </div>
      </div>

      <div className="toolbar">
        <input
          className="search"
          type="search"
          value={query}
          placeholder="搜索模型 ID，如 gpt"
          onChange={(e) => setQuery(e.target.value)}
        />
        <div className="segmented">
          <button
            className={mode === "by_key" ? "seg seg-active" : "seg"}
            onClick={() => setMode("by_key")}
          >
            按 Key 分组
          </button>
          <button
            className={mode === "merged" ? "seg seg-active" : "seg"}
            onClick={() => setMode("merged")}
          >
            汇总去重
          </button>
        </div>
      </div>

      {data && data.query && (
        <div className="result-hint">
          搜索 “{data.query}” 的结果
        </div>
      )}

      {loading && !data ? (
        <Spinner label="加载模型…" />
      ) : (
        <ModelTree mode={data?.mode || mode} providers={data?.providers || []} />
      )}
    </section>
  );
}
