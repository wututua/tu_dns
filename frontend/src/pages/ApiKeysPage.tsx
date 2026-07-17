import { FormEvent, useEffect, useState } from "react";
import { api } from "../lib/api";

type ApiKey = {
  id: number;
  name: string;
  key_prefix: string;
  last_used_at: string | null;
  enabled: boolean;
  created_at: string;
};

export default function ApiKeysPage() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [name, setName] = useState("");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [createdKey, setCreatedKey] = useState("");

  const load = async () => {
    const d = await api<ApiKey[]>("/api/api-keys");
    setKeys(d || []);
  };

  useEffect(() => {
    load().catch((e) => setErr(e.message));
  }, []);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    setCreatedKey("");
    if (!name.trim()) { setErr("名称必填"); return; }
    try {
      const res = await api<{ raw_key: string }>("/api/api-keys", {
        method: "POST",
        body: JSON.stringify({ name: name.trim() }),
      });
      setCreatedKey(res.raw_key);
      setName("");
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "创建失败");
    }
  };

  const revoke = async (id: number) => {
    if (!confirm("确认吊销该密钥？")) return;
    await api(`/api/api-keys/${id}`, { method: "DELETE" });
    await load();
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">API 密钥</h1>
        <p className="muted mt-1">用于程序化调用 API，请妥善保管密钥原文</p>
      </div>

      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      {createdKey && (
        <div className="card border-amber-400/30 bg-amber-500/10">
          <div className="font-medium text-amber-300">新密钥已创建</div>
          <p className="mt-1 text-amber-200/80 text-sm">请立即复制，关闭后不再显示：</p>
          <div className="mt-2 flex gap-2">
            <input className="input flex-1" readOnly value={createdKey} onClick={(e) => e.currentTarget.select()} />
            <button className="btn" onClick={() => { navigator.clipboard.writeText(createdKey); setMsg("已复制"); }}>复制</button>
          </div>
          <button className="btn-ghost mt-2 text-sm" onClick={() => setCreatedKey("")}>关闭</button>
        </div>
      )}

      <form className="card flex gap-3" onSubmit={create}>
        <input className="input flex-1" value={name} onChange={(e) => setName(e.target.value)} placeholder="密钥名称，如 CI/CD" />
        <button className="btn">创建密钥</button>
      </form>

      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>名称</th>
              <th>前缀</th>
              <th>状态</th>
              <th>最后使用</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k) => (
              <tr key={k.id}>
                <td>{k.name}</td>
                <td><code className="text-xs bg-white/5 px-1.5 py-0.5 rounded">tud_{k.key_prefix}...</code></td>
                <td>{k.enabled ? <span className="text-emerald-400">启用</span> : <span className="muted">已吊销</span>}</td>
                <td className="muted text-xs">{k.last_used_at ? new Date(k.last_used_at).toLocaleString("zh-CN") : "-"}</td>
                <td className="muted text-xs">{new Date(k.created_at).toLocaleString("zh-CN")}</td>
                <td>
                  {k.enabled && <button className="btn-ghost" onClick={() => revoke(k.id)}>吊销</button>}
                </td>
              </tr>
            ))}
            {!keys.length && <tr><td colSpan={6} className="muted">暂无密钥</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
