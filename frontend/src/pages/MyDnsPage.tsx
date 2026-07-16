import { FormEvent, useEffect, useState } from "react";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";

type Subdomain = {
  id: number;
  full_domain: string;
  name: string;
  domain_id: number;
  status: number;
};

type RecordItem = {
  id: number;
  subdomain_id: number;
  name: string;
  type: string;
  value: string;
  ttl: number;
};

export default function MyDnsPage() {
  const { refresh } = useAuth();
  const [subs, setSubs] = useState<Subdomain[]>([]);
  const [records, setRecords] = useState<RecordItem[]>([]);
  const [selectedSub, setSelectedSub] = useState(0);
  const [type, setType] = useState("A");
  const [value, setValue] = useState("");
  const [ttl, setTtl] = useState(600);
  const [err, setErr] = useState("");
  const [msg, setMsg] = useState("");
  const [editId, setEditId] = useState(0);
  const [editValue, setEditValue] = useState("");

  const load = async () => {
    const [s, r] = await Promise.all([
      api<Subdomain[]>("/api/subdomains"),
      api<RecordItem[]>("/api/records"),
    ]);
    setSubs(s || []);
    setRecords(r || []);
    if (s?.length && !selectedSub) setSelectedSub(s[0].id);
  };

  useEffect(() => {
    load().catch((e) => setErr(e.message));
  }, []);

  const addRecord = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const res = await api<{ charged: number }>("/api/records", {
        method: "POST",
        body: JSON.stringify({ subdomain_id: selectedSub, type, value, ttl }),
      });
      setMsg(`新增成功，扣费 ${res.charged}`);
      setValue("");
      await load();
      await refresh();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "失败");
    }
  };

  const saveEdit = async (id: number) => {
    try {
      await api(`/api/records/${id}`, {
        method: "PUT",
        body: JSON.stringify({ value: editValue }),
      });
      setEditId(0);
      setMsg("修改成功（不扣费）");
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "修改失败");
    }
  };

  const delRecord = async (id: number) => {
    if (!confirm("确认删除该记录？不退积分")) return;
    await api(`/api/records/${id}`, { method: "DELETE" });
    await load();
  };

  const delSub = async (id: number) => {
    if (!confirm("删除子域将同步删除其全部解析记录，确认？")) return;
    await api(`/api/subdomains/${id}`, { method: "DELETE" });
    setSelectedSub(0);
    await load();
  };

  const filtered = records.filter((r) => !selectedSub || r.subdomain_id === selectedSub);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">我的域名</h1>
        <p className="muted mt-1">管理已申请的子域与解析记录</p>
      </div>

      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
        <div className="card">
          <h2 className="mb-3 font-medium">子域名</h2>
          <div className="space-y-2">
            {subs.map((s) => (
              <div
                key={s.id}
                className={`rounded-xl border px-3 py-2 ${
                  selectedSub === s.id ? "border-indigo-400/50 bg-indigo-500/10" : "border-white/10"
                }`}
              >
                <button className="w-full text-left" onClick={() => setSelectedSub(s.id)}>
                  {s.full_domain}
                </button>
                <button className="mt-1 text-xs text-rose-300" onClick={() => delSub(s.id)}>
                  删除
                </button>
              </div>
            ))}
            {!subs.length && <div className="muted">暂无子域</div>}
          </div>
        </div>

        <div className="space-y-4">
          <form className="card grid gap-3 md:grid-cols-4" onSubmit={addRecord}>
            <div>
              <label className="label">类型</label>
              <select className="input" value={type} onChange={(e) => setType(e.target.value)}>
                {["A", "AAAA", "CNAME", "TXT", "MX"].map((t) => (
                  <option key={t}>{t}</option>
                ))}
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="label">值（新增将扣费）</label>
              <input className="input" value={value} onChange={(e) => setValue(e.target.value)} />
            </div>
            <div>
              <label className="label">TTL</label>
              <input
                className="input"
                type="number"
                value={ttl}
                onChange={(e) => setTtl(Number(e.target.value))}
              />
            </div>
            <div className="md:col-span-4">
              <button className="btn" disabled={!selectedSub}>
                新增记录
              </button>
            </div>
          </form>

          <div className="card overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>主机</th>
                  <th>类型</th>
                  <th>值</th>
                  <th>TTL</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r) => (
                  <tr key={r.id}>
                    <td>{r.name}</td>
                    <td>{r.type}</td>
                    <td>
                      {editId === r.id ? (
                        <input
                          className="input"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                        />
                      ) : (
                        r.value
                      )}
                    </td>
                    <td>{r.ttl}</td>
                    <td className="space-x-2">
                      {editId === r.id ? (
                        <button className="btn-ghost" onClick={() => saveEdit(r.id)}>
                          保存
                        </button>
                      ) : (
                        <button
                          className="btn-ghost"
                          onClick={() => {
                            setEditId(r.id);
                            setEditValue(r.value);
                          }}
                        >
                          修改
                        </button>
                      )}
                      <button className="btn-ghost" onClick={() => delRecord(r.id)}>
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
                {!filtered.length && (
                  <tr>
                    <td colSpan={5} className="muted">
                      暂无记录
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
}
