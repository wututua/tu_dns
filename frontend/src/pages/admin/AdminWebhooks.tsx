import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";

type Webhook = {
  id: number;
  name: string;
  url: string;
  events: string;
  enabled: boolean;
  created_at: string;
};

export default function AdminWebhooks() {
  const [items, setItems] = useState<Webhook[]>([]);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const [editId, setEditId] = useState(0);
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [events, setEvents] = useState("*");
  const [secret, setSecret] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [showSecret, setShowSecret] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);

  const load = async () => {
    const d = await api<Webhook[]>("/api/admin/webhooks");
    setItems(d || []);
  };

  useEffect(() => {
    load().catch((e) => setErr(e.message));
  }, []);

  const reset = () => {
    setEditId(0);
    setName("");
    setURL("");
    setEvents("*");
    setSecret("");
    setEnabled(true);
    setShowSecret(false);
    setErr("");
    setMsg("");
  };

  const openEdit = (w: Webhook) => {
    setEditId(w.id);
    setName(w.name);
    setURL(w.url);
    setEvents(w.events);
    setEnabled(w.enabled);
    setSecret("");
    setShowSecret(false);
    setErr("");
    setMsg("");
    setModalOpen(true);
  };

  const save = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const url2 = editId ? `/api/admin/webhooks/${editId}` : "/api/admin/webhooks";
      const method = editId ? "PUT" : "POST";
      const body: Record<string, unknown> = { name, url, events, enabled };
      if (editId && !showSecret) body.secret = "";
      if (!editId || secret) body.secret = secret;
      await api(url2, { method, body: JSON.stringify(body) });
      setMsg("已保存");
      setModalOpen(false);
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "保存失败");
    }
  };

  const remove = async (id: number) => {
    if (!confirm("确认删除？")) return;
    await api(`/api/admin/webhooks/${id}`, { method: "DELETE" });
    await load();
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Webhook</h1>
          <p className="muted mt-1">配置事件回调，当 DNS 变更/支付完成时通知外部系统</p>
        </div>
        <button className="btn" onClick={() => { reset(); setModalOpen(true); }}>添加</button>
      </div>

      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      {modalOpen && (
        <div className="modal-backdrop" onClick={() => setModalOpen(false)}>
          <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">{editId ? "编辑" : "添加"} Webhook</h2>
              <button className="btn-ghost text-sm" onClick={() => setModalOpen(false)}>&times;</button>
            </div>
            <form className="grid gap-3" onSubmit={save}>
              <div>
                <label className="label">名称</label>
                <input className="input" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
              <div>
                <label className="label">回调 URL</label>
                <input className="input" value={url} onChange={(e) => setURL(e.target.value)} required placeholder="https://example.com/webhook" />
              </div>
              <div>
                <label className="label">事件（逗号分隔，* 为全部）</label>
                <input className="input" value={events} onChange={(e) => setEvents(e.target.value)} placeholder="subdomain.created,record.created,payment.completed" />
                <div className="muted text-xs mt-1">可用事件: subdomain.created, record.created, record.updated, record.deleted, payment.completed</div>
              </div>
              <div>
                <label className="label">
                  <label className="flex items-center gap-2">
                    <input type="checkbox" checked={showSecret} onChange={(e) => setShowSecret(e.target.checked)} />
                    设置密钥
                  </label>
                </label>
                {showSecret && (
                  <input className="input" value={secret} onChange={(e) => setSecret(e.target.value)} placeholder="HMAC SHA256 签名密钥" />
                )}
              </div>
              <div>
                <label className="label">状态</label>
                <select className="input" value={enabled ? 1 : 0} onChange={(e) => setEnabled(e.target.value === "1")}>
                  <option value={1}>启用</option>
                  <option value={0}>禁用</option>
                </select>
              </div>
              <div className="flex gap-3">
                <button className="btn flex-1" type="submit">保存</button>
                <button className="btn-ghost" type="button" onClick={() => setModalOpen(false)}>取消</button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>名称</th>
              <th>URL</th>
              <th>事件</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((w) => (
              <tr key={w.id}>
                <td>{w.name}</td>
                <td className="max-w-xs truncate">{w.url}</td>
                <td className="muted">{w.events}</td>
                <td>{w.enabled ? <span className="text-emerald-400">启用</span> : <span className="muted">禁用</span>}</td>
                <td className="space-x-2">
                  <button className="btn-ghost" onClick={() => openEdit(w)}>修改</button>
                  <button className="btn-ghost" onClick={() => remove(w.id)}>删除</button>
                </td>
              </tr>
            ))}
            {!items.length && <tr><td colSpan={5} className="muted">暂无 Webhook</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
