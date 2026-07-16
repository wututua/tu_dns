import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../lib/auth";

export default function Layout() {
  const { user, logout, refresh } = useAuth();

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
            </>
          )}
        </nav>
      </aside>
      <div className="flex flex-col overflow-hidden">
        <header className="flex items-center justify-between border-b border-white/10 px-6 py-4 shrink-0">
          <div className="muted">欢迎，{user?.username}</div>
          <div className="flex items-center gap-3">
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
