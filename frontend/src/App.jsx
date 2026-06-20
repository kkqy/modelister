import { useCallback, useEffect, useState } from "react";
import { api, ApiError } from "./api.js";
import { Toast, useToast, Spinner } from "./components/ui.jsx";
import Login from "./components/Login.jsx";
import Providers from "./components/Providers.jsx";
import Models from "./components/Models.jsx";

export default function App() {
  const [authState, setAuthState] = useState("loading"); // loading | out | in
  const [username, setUsername] = useState("");
  const [tab, setTab] = useState("providers");
  const toast = useToast();

  const checkAuth = useCallback(async () => {
    try {
      const me = await api.me();
      if (me && me.authenticated) {
        setUsername(me.username || "");
        setAuthState("in");
      } else {
        setAuthState("out");
      }
    } catch {
      setAuthState("out");
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  const handleLoggedIn = (name) => {
    setUsername(name);
    setAuthState("in");
  };

  const handleLogout = async () => {
    try {
      await api.logout();
    } catch (err) {
      // 即使请求失败也回到登录态，避免卡死
    }
    setUsername("");
    setAuthState("out");
    toast.success("已退出登录");
  };

  if (authState === "loading") {
    return (
      <div className="centered">
        <Spinner label="加载中…" />
      </div>
    );
  }

  if (authState === "out") {
    return (
      <>
        <Login onLoggedIn={handleLoggedIn} />
        <Toast toast={toast.toast} onClose={toast.clear} />
      </>
    );
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="app-brand">
          <span className="app-logo">◆</span>
          <h1>Modelister</h1>
        </div>
        <nav className="app-tabs">
          <button
            className={tab === "providers" ? "tab tab-active" : "tab"}
            onClick={() => setTab("providers")}
          >
            供应商
          </button>
          <button
            className={tab === "models" ? "tab tab-active" : "tab"}
            onClick={() => setTab("models")}
          >
            模型
          </button>
        </nav>
        <div className="app-user">
          <span className="app-username">{username}</span>
          <button className="btn btn-ghost" onClick={handleLogout}>
            退出
          </button>
        </div>
      </header>

      <main className="app-main">
        {tab === "providers" ? (
          <Providers toast={toast} onUnauthorized={() => setAuthState("out")} />
        ) : (
          <Models toast={toast} onUnauthorized={() => setAuthState("out")} />
        )}
      </main>

      <Toast toast={toast.toast} onClose={toast.clear} />
    </div>
  );
}

// 供子组件统一处理 401：踢回登录页。
export function isUnauthorized(err) {
  return err instanceof ApiError && err.status === 401;
}
