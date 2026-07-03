import { useCallback, useEffect, useState } from "react";
import { api } from "../api.js";
import { formatTime } from "../format.js";
import ProviderUrlLink from "./ProviderUrlLink.jsx";
import { Spinner } from "./ui.jsx";

const PAGE_SIZE = 20;
const MODEL_PREVIEW_LIMIT = 24;

function ChangeModelList({ title, models, count, tone }) {
  if (!count) return null;

  const items = Array.isArray(models) ? models.slice(0, MODEL_PREVIEW_LIMIT) : [];
  const hidden = Math.max(0, count - items.length);

  return (
    <div className="change-model-group">
      <div className={`change-model-title change-model-title-${tone}`}>{title}</div>
      <ul className="change-model-list">
        {items.map((id) => (
          <li className={`change-model change-model-${tone}`} key={id}>
            <code>{id}</code>
          </li>
        ))}
        {hidden > 0 && <li className="change-model change-model-more">另有 {hidden} 个</li>}
      </ul>
    </div>
  );
}

function ChangeEventItem({ event }) {
  return (
    <article className="timeline-item">
      <div className="timeline-dot" aria-hidden="true" />
      <div className="timeline-content">
        <div className="timeline-head">
          <div className="timeline-title">
            <h3>{event.provider_name}</h3>
            <span className="timeline-key">{event.key_name}</span>
          </div>
          <time className="timeline-time" dateTime={event.created_at}>
            {formatTime(event.created_at)}
          </time>
        </div>
        <div className="timeline-meta">
          <ProviderUrlLink url={event.base_url} />
        </div>
        <div className="change-summary">
          <span className="change-pill change-pill-added">+{event.added_count} 新增</span>
          <span className="change-pill change-pill-removed">-{event.removed_count} 移除</span>
        </div>
        <div className="change-models">
          <ChangeModelList
            title="新增模型"
            models={event.added_models}
            count={event.added_count}
            tone="added"
          />
          <ChangeModelList
            title="移除模型"
            models={event.removed_models}
            count={event.removed_count}
            tone="removed"
          />
        </div>
      </div>
    </article>
  );
}

export default function ModelChanges({ toast, onUnauthorized }) {
  const [events, setEvents] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const [nextBeforeId, setNextBeforeId] = useState(null);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const handleError = useCallback(
    (err) => {
      if (err && err.status === 401) return onUnauthorized();
      toast.error(err.message || "读取模型变动记录失败");
    },
    [onUnauthorized, toast]
  );

  const loadPage = useCallback(
    async (beforeId = null) => {
      const firstPage = !beforeId;
      if (firstPage) {
        setLoading(true);
      } else {
        setLoadingMore(true);
      }
      try {
        const res = await api.listModelChanges({ limit: PAGE_SIZE, beforeId });
        const pageEvents = res.events || [];
        setEvents((current) => (firstPage ? pageEvents : [...current, ...pageEvents]));
        setHasMore(Boolean(res.has_more));
        setNextBeforeId(res.next_before_id || null);
      } catch (err) {
        handleError(err);
      } finally {
        if (firstPage) {
          setLoading(false);
        } else {
          setLoadingMore(false);
        }
      }
    },
    [handleError]
  );

  useEffect(() => {
    loadPage();
  }, [loadPage]);

  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>变动记录</h2>
          <p className="panel-sub">按时间线查看每次 Key 刷新带来的模型新增与移除</p>
        </div>
        <div className="panel-actions">
          <button className="btn" onClick={() => loadPage()} disabled={loading || loadingMore}>
            {loading ? "刷新中…" : "刷新记录"}
          </button>
        </div>
      </div>

      {loading && events.length === 0 ? (
        <Spinner label="加载变动记录…" />
      ) : events.length === 0 ? (
        <div className="empty">暂无模型变动记录。有 Key 刷新出新增或移除模型后会出现在这里。</div>
      ) : (
        <>
          <div className="timeline">
            {events.map((event) => (
              <ChangeEventItem event={event} key={event.id} />
            ))}
          </div>
          {hasMore && (
            <div className="timeline-more">
              <button
                className="btn"
                onClick={() => loadPage(nextBeforeId)}
                disabled={loadingMore || !nextBeforeId}
              >
                {loadingMore ? "加载中…" : "加载更早记录"}
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}
