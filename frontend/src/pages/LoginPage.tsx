import { FormEvent, useState } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";

export default function LoginPage() {
  const { user, login } = useAuth();
  const nav = useNavigate();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  if (user) return <Navigate to="/" replace />;

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setErr("");
    try {
      await login(username, password);
      nav("/");
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "登录失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto flex min-h-screen max-w-md items-center p-6">
      <form className="card w-full space-y-4" onSubmit={onSubmit}>
        <div>
          <h1 className="text-2xl font-semibold">登录 TuDNS</h1>
          <p className="muted mt-1">使用账号密码进入控制台</p>
        </div>
        <div>
          <label className="label">用户名</label>
          <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div>
          <label className="label">密码</label>
          <input
            className="input"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
        {err && <div className="error">{err}</div>}
        <button className="btn w-full" disabled={loading}>
          {loading ? "登录中…" : "登录"}
        </button>
        <p className="muted text-center">
          没有账号？ <Link to="/register">注册</Link>
        </p>
      </form>
    </div>
  );
}
