import { useState } from "react";
import { api } from "../api.js";

export default function Login({ onLoggedIn }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async (e) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const res = await api.login(username.trim(), password);
      onLoggedIn(res && res.username ? res.username : username.trim());
    } catch (err) {
      setError(err.message || "登录失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-wrap">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">
          <span className="app-logo">◆</span>
          <h1>Modelister</h1>
        </div>
        <p className="login-sub">供应商模型管理控制台</p>

        <label className="field">
          <span>用户名</span>
          <input
            type="text"
            value={username}
            autoComplete="username"
            autoFocus
            onChange={(e) => setUsername(e.target.value)}
            required
          />
        </label>

        <label className="field">
          <span>密码</span>
          <input
            type="password"
            value={password}
            autoComplete="current-password"
            onChange={(e) => setPassword(e.target.value)}
            required
          />
        </label>

        {error && <div className="form-error">{error}</div>}

        <button className="btn btn-primary btn-block" type="submit" disabled={submitting}>
          {submitting ? "登录中…" : "登录"}
        </button>
      </form>
    </div>
  );
}
