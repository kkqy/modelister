import { useEffect, useState } from "react";

// 轻量提示条：成功 / 错误，几秒后自动消失。
export function Toast({ toast, onClose }) {
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(onClose, 4000);
    return () => clearTimeout(timer);
  }, [toast, onClose]);

  if (!toast) return null;
  return (
    <div className={`toast toast-${toast.kind}`} role="status">
      <span>{toast.message}</span>
      <button className="toast-close" onClick={onClose} aria-label="关闭">
        ×
      </button>
    </div>
  );
}

// 统一的 toast 状态管理 hook。
export function useToast() {
  const [toast, setToast] = useState(null);
  return {
    toast,
    clear: () => setToast(null),
    success: (message) => setToast({ kind: "success", message }),
    error: (message) => setToast({ kind: "error", message }),
  };
}

export function Spinner({ label }) {
  return (
    <div className="spinner">
      <span className="spinner-dot" />
      {label && <span>{label}</span>}
    </div>
  );
}

export function ConfirmButton({ onConfirm, children, className = "", confirmText }) {
  const [armed, setArmed] = useState(false);
  useEffect(() => {
    if (!armed) return;
    const timer = setTimeout(() => setArmed(false), 3000);
    return () => clearTimeout(timer);
  }, [armed]);

  return (
    <button
      className={`btn btn-danger ${className}`}
      onClick={() => {
        if (armed) {
          setArmed(false);
          onConfirm();
        } else {
          setArmed(true);
        }
      }}
    >
      {armed ? confirmText || "确认删除？" : children}
    </button>
  );
}
