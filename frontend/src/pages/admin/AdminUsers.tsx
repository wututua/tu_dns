import { useEffect, useState } from "react";
import { api } from "../../lib/api";

type User = {
  id: number;
  username: string;
  email: string;
  role: string;
  status: number;
  points: number;
  created_at: string;
};

const roleLabel = (r: string) =>
  ({ admin: "管理员", premium: "高级会员", user: "正式用户" }[r] || r);

const statusLabel = (s: number) => (s === 1 ? "正常" : "封禁");

export default function AdminUsers() {
  const [items, setItems] = useState<User[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");

  // dropdown
  const [openMenu, setOpenMenu] = useState(0);

  // edit modal
  const [edit, setEdit] = useState<User | null>(null);
  const [eUsername, setEUsername] = useState("");
  const [eEmail, setEEmail] = useState("");
  const [eRole, setERole] = useState("user");
  const [eStatus, setEStatus] = useState(1);
  const [eDelta, setEDelta] = useState(0);

  const load = async (p: number) => {
    const data = await api<{ items: User[]; total: number }>(`/api/admin/users?page=${p}`);
    setItems(data.items || []);
    setTotal(data.total || 0);
  };

  useEffect(() => {
    load(page).catch((e) => setErr(e.message));
    const closer = () => setOpenMenu(0);
    document.addEventListener("click", closer);
    return () => document.removeEventListener("click", closer);
  }, []);

  const toggleMenu = (id: number) => {
    setOpenMenu((prev) => (prev === id ? 0 : id));
  };

  const openEdit = (u: User) => {
    setEdit(u);
    setEUsername(u.username);
    setEEmail(u.email || "");
    setERole(u.role);
    setEStatus(u.status);
    setEDelta(0);
    setOpenMenu(0);
    setErr("");
    setMsg("");
  };

  const saveEdit = async () => {
    if (!edit) return;
    setErr("");
    try {
      await api(`/api/admin/users/${edit.id}`, {
        method: "PUT",
        body: JSON.stringify({
          username: eUsername,
          email: eEmail,
          role: eRole,
          status: eStatus,
        }),
      });
      if (eDelta !== 0) {
        await api(`/api/admin/users/${edit.id}/points`, {
          method: "POST",
          body: JSON.stringify({ delta: eDelta, remark: "管理员调整" }),
        });
      }
      setMsg("已保存");
      setEdit(null);
      await load(page);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "保存失败");
    }
  };

  const toggleStatus = async (u: User) => {
    setOpenMenu(0);
    const newStatus = u.status === 1 ? 0 : 1;
    try {
      await api(`/api/admin/users/${u.id}`, {
        method: "PUT",
        body: JSON.stringify({ status: newStatus }),
      });
      await load(page);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "操作失败");
    }
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">用户管理</h1>
      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      <div className="muted text-sm">共 {total} 条</div>

      <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>邮箱</th>
              <th>用户组</th>
              <th>注册时间</th>
              <th>状态</th>
              <th style={{ width: 60 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((u) => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>{u.username}</td>
                <td>{u.email || "-"}</td>
                <td>{roleLabel(u.role)}</td>
                <td className="muted">{new Date(u.created_at).toLocaleDateString("zh-CN")}</td>
                <td>
                  <span className={u.status === 1 ? "text-emerald-400" : "text-rose-400"}>
                    {statusLabel(u.status)}
                  </span>
                </td>
                <td className="relative">
                  <button
                    className="btn-ghost px-2 text-lg leading-none"
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleMenu(u.id);
                    }}
                  >
                    &#8942;
                  </button>
                  {openMenu === u.id && (
                    <div
                      className="dropdown-menu"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <button className="dropdown-item" onClick={() => openEdit(u)}>
                        编辑
                      </button>
                      <button className="dropdown-item" onClick={() => toggleStatus(u)}>
                        {u.status === 1 ? "封禁" : "解封"}
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

      {Math.ceil(total / 20) > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button className="btn-ghost" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            上一页
          </button>
          <span className="muted text-sm">{page} / {Math.ceil(total / 20)}</span>
          <button className="btn-ghost" disabled={page >= Math.ceil(total / 20)} onClick={() => setPage(page + 1)}>
            下一页
          </button>
        </div>
      )}

      {/* edit modal */}
      {edit && (
        <div className="modal-backdrop" onClick={() => setEdit(null)}>
          <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">编辑用户 #{edit.id}</h2>
              <button className="btn-ghost text-sm" onClick={() => setEdit(null)}>&times;</button>
            </div>

            {err && <div className="error mb-3">{err}</div>}

            <div className="grid gap-3">
              <div>
                <label className="label">用户名</label>
                <input className="input" value={eUsername} onChange={(e) => setEUsername(e.target.value)} />
              </div>
              <div>
                <label className="label">邮箱</label>
                <input className="input" value={eEmail} onChange={(e) => setEEmail(e.target.value)} />
              </div>
              <div>
                <label className="label">用户组</label>
                <select className="input" value={eRole} onChange={(e) => setERole(e.target.value)}>
                  <option value="admin">管理员</option>
                  <option value="user">正式用户</option>
                  <option value="premium">高级会员</option>
                </select>
              </div>
              <div>
                <label className="label">账户状态</label>
                <select className="input" value={eStatus} onChange={(e) => setEStatus(Number(e.target.value))}>
                  <option value={1}>正常</option>
                  <option value={0}>封禁</option>
                </select>
              </div>
              <div>
                <label className="label">积分调整（正数增加，负数扣减）</label>
                <input
                  className="input"
                  type="number"
                  value={eDelta}
                  onChange={(e) => setEDelta(Number(e.target.value))}
                />
              </div>
              <div className="flex gap-3 pt-1">
                <button className="btn flex-1" onClick={saveEdit}>保存</button>
                <button className="btn-ghost" onClick={() => setEdit(null)}>取消</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
