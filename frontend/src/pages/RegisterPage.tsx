import { FormEvent, useState } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";

export default function RegisterPage() {
  const { user, register } = useAuth();
  const nav = useNavigate();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [email, setEmail] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  if (user) return <Navigate to="/" replace />;

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setErr("");
    try {
      await register(username, password, email);
      nav("/");
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "注册失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto flex min-h-screen max-w-md items-center p-6">
      <form className="card w-full space-y-4" onSubmit={onSubmit}>
        <div>
          <h1 className="text-2xl font-semibold">注册</h1>
          <p className="muted mt-1">创建账号后即可申请二级域名解析</p>
        </div>
        <div>
          <label className="label">用户名</label>
          <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} />
        </div>
        <div>
          <label className="label">邮箱</label>
          <input className="input" value={email} onChange={(e) => setEmail(e.target.value)} />
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
          {loading ? "提交中…" : "注册"}
        </button>
        <p className="muted text-center">
          已有账号？ <Link to="/login">登录</Link>
        </p>
      </form>
    </div>
  );
}
