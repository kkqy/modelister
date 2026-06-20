// 把后端的 RFC3339 时间字符串转成本地可读格式；空串 / 解析失败时返回占位符。
export function formatTime(value) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { hour12: false });
}

// 把 sync 接口返回的 results 数组归纳成一句人话。
export function summarizeSync(results) {
  if (!Array.isArray(results) || results.length === 0) {
    return { ok: true, message: "没有需要同步的启用 Key" };
  }
  const okCount = results.filter((r) => r.ok).length;
  const failed = results.filter((r) => !r.ok);
  const total = results.reduce((sum, r) => sum + (r.ok ? r.count || 0 : 0), 0);
  if (failed.length === 0) {
    return { ok: true, message: `同步成功，${okCount} 个 Key，共 ${total} 个模型` };
  }
  const firstErr = failed[0].error || "未知错误";
  return {
    ok: false,
    message: `部分失败：${okCount} 成功 / ${failed.length} 失败（${firstErr}）`,
  };
}
