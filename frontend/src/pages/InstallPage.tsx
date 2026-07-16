import { useState } from "react";
import { api } from "../lib/api";

export default function InstallPage({ onDone }: { onDone: () => void }) {
  const [driver, setDriver] = useState("sqlite");
  const [dsn, setDsn] = useState("");
  const [sqlitePath, setSqlitePath] = useState("tudns.db");
  const [adminUser, setAdminUser] = useState("admin");
  const [adminPass, setAdminPass] = useState("");
  const [adminEmail, setAdminEmail] = useState("");
  const [siteName, setSiteName] = useState("TuDNS");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  const payload = () => ({
    driver,
    dsn,
    sqlite_path: sqlitePath,
    admin_user: adminUser,
    admin_pass: adminPass,
    admin_email: adminEmail,
    site_name: siteName,
  });

  const testDB = async () => {
    setErr("");
    setMsg("");
    try {
      await api("/api/install/test-db", { method: "POST", body: JSON.stringify(payload()) });
      setMsg("数据库连接成功");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "测试失败");
    }
  };

  const install = async () => {
    setLoading(true);
    setErr("");
    setMsg("");
    try {
      await api("/api/install", { method: "POST", body: JSON.stringify(payload()) });
      setMsg("安装完成，即将进入系统");
      onDone();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "安装失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto flex min-h-screen max-w-2xl items-center p-6">
      <div className="card w-full space-y-4">
        <div>
          <h1 className="text-2xl font-semibold">TuDNS 安装向导</h1>
          <p className="muted mt-1">首次运行：选择数据库并创建管理员</p>
        </div>

        <div>
          <label className="label">数据库类型</label>
          <select className="input" value={driver} onChange={(e) => setDriver(e.target.value)}>
            <option value="sqlite">SQLite</option>
            <option value="mysql">MySQL</option>
            <option value="postgres">PostgreSQL</option>
          </select>
        </div>

        {driver === "sqlite" ? (
          <div>
            <label className="label">SQLite 文件名（相对 data/）</label>
            <input className="input" value={sqlitePath} onChange={(e) => setSqlitePath(e.target.value)} />
          </div>
        ) : (
          <div>
            <label className="label">DSN 连接串</label>
            <input
              className="input"
              placeholder={
                driver === "mysql"
                  ? "user:pass@tcp(127.0.0.1:3306)/tudns?parseTime=true&charset=utf8mb4"
                  : "host=127.0.0.1 user=tudns password=xxx dbname=tudns port=5432 sslmode=disable"
              }
              value={dsn}
              onChange={(e) => setDsn(e.target.value)}
            />
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label className="label">管理员用户名</label>
            <input className="input" value={adminUser} onChange={(e) => setAdminUser(e.target.value)} />
          </div>
          <div>
            <label className="label">管理员密码</label>
            <input
              className="input"
              type="password"
              value={adminPass}
              onChange={(e) => setAdminPass(e.target.value)}
            />
          </div>
        </div>
        <div>
          <label className="label">邮箱（可选）</label>
          <input className="input" value={adminEmail} onChange={(e) => setAdminEmail(e.target.value)} />
        </div>
        <div>
          <label className="label">站点名称</label>
          <input className="input" value={siteName} onChange={(e) => setSiteName(e.target.value)} />
        </div>

        {err && <div className="error">{err}</div>}
        {msg && <div className="success">{msg}</div>}

        <div className="flex gap-3">
          <button className="btn-ghost" type="button" onClick={testDB}>
            测试连接
          </button>
          <button className="btn" type="button" disabled={loading} onClick={install}>
            {loading ? "安装中…" : "完成安装"}
          </button>
        </div>
      </div>
    </div>
  );
}
