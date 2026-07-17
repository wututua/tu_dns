import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { useEffect, useState } from "react";
import { api } from "../lib/api";

export default function Layout() {
  const { user, logout, refresh } = useAuth();
  const navigate = useNavigate();
  const [unread, setUnread] = useState(0);

  const loadUnread = async () => {
    try {
      const data = await api<{ count: number }>("/api/notifications/unread");
      setUnread(data.count || 0);
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    if (!user) return;
    loadUnread();
    const timer = setInterval(loadUnread, 30000);
    return () => clearInterval(timer);
  }, [user]);

  return (
    <div className="h-screen overflow-hidden grid md:grid-cols-[240px_1fr]">
      <aside className="glass border-r border-white/10 p-5 overflow-y-auto">
        <div className="mb-8">
          <div className="text-xl font-semibold tracking-tight">TuDNS</div>
          <div className="muted mt-1">二级域名分发</div>
        </div>
        <nav className="space-y-1">
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/" end>
            仪表盘
          </NavLink>
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/domains">
            创建子域
          </NavLink>
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/mydns">
            我的解析
          </NavLink>
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/api-keys">
            API 密钥
          </NavLink>
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/notifications">
            通知
            {unread > 0 && <span className="ml-2 text-xs bg-indigo-500 text-white px-1.5 py-0.5 rounded-full">{unread}</span>}
          </NavLink>
          <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/points">
            积分中心
          </NavLink>
          {user?.role === "admin" && (
            <>
              <div className="pt-4 pb-1 text-xs uppercase tracking-wider text-slate-500">管理</div>
              <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/users">
                用户管理
              </NavLink>
              <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/domains">
                域名管理
              </NavLink>
              <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/redeem">
                兑换码
              </NavLink>
              <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/orders">
                支付订单
              </NavLink>
              <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/settings">
                 系统设置
               </NavLink>
               <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/logs">
                 操作日志
               </NavLink>
               <NavLink className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`} to="/admin/webhooks">
                 Webhook
               </NavLink>
            </>
          )}
        </nav>
      </aside>
      <div className="flex flex-col overflow-hidden">
        <header className="flex items-center justify-between border-b border-white/10 px-6 py-4 shrink-0">
          <div className="muted">欢迎，{user?.username}</div>
          <div className="flex items-center gap-3">
            <button className="btn-ghost relative" onClick={() => navigate("/notifications")}>
              通知
              {unread > 0 && (
                <span className="absolute -top-1 -right-1 text-xs bg-indigo-500 text-white w-4 h-4 rounded-full flex items-center justify-center">
                  {unread > 9 ? "9+" : unread}
                </span>
              )}
            </button>
            <span className="badge">积分 {user?.points ?? 0}</span>
            <button className="btn-ghost" onClick={() => refresh()}>
              刷新
            </button>
            <button className="btn-ghost" onClick={logout}>
              退出
            </button>
          </div>
        </header>
        <main className="flex-1 p-6 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
