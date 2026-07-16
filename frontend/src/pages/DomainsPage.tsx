import { FormEvent, useEffect, useState } from "react";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";

type Domain = {
  id: number;
  name: string;
  points_cost: number;
  record_types: string;
  description: string;
};

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

export default function DomainsPage() {
  const { refresh } = useAuth();

  const [domains, setDomains] = useState<Domain[]>([]);
  const [domainId, setDomainId] = useState<number>(0);
  const [sub, setSub] = useState("");
  const [bundleType, setBundleType] = useState("A");
  const [bundleValue, setBundleValue] = useState("");
  const [bundleTTL, setBundleTTL] = useState(600);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  const [subs, setSubs] = useState<Subdomain[]>([]);
  const [records, setRecords] = useState<RecordItem[]>([]);
  const [selectedSub, setSelectedSub] = useState(0);
  const [recType, setRecType] = useState("A");
  const [recValue, setRecValue] = useState("");
  const [recTTL, setRecTTL] = useState(600);
  const [editId, setEditId] = useState(0);
  const [editValue, setEditValue] = useState("");
  const [modalOpen, setModalOpen] = useState(false);

  const loadDomains = () => {
    api<Domain[]>("/api/public/domains").then((d) => {
      setDomains(d || []);
      if (d?.length) setDomainId(d[0].id);
    });
  };

  const loadMy = async () => {
    const [s, r] = await Promise.all([
      api<Subdomain[]>("/api/subdomains"),
      api<RecordItem[]>("/api/records"),
    ]);
    setSubs(s || []);
    setRecords(r || []);
    if (s?.length && !selectedSub) setSelectedSub(s[0].id);
  };

  useEffect(() => {
    loadDomains();
    loadMy().catch((e) => setErr(e.message));
  }, []);

  const selected = domains.find((d) => d.id === domainId);
  const types = (selected?.record_types || "A,AAAA,CNAME,TXT").split(",").map((s) => s.trim());

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setErr("");
    setMsg("");
    try {
      const res = await api<{ subdomain: { full_domain: string }; charged: number }>(
        "/api/subdomains/bundle",
        {
          method: "POST",
          body: JSON.stringify({
            domain_id: domainId,
            subdomain_name: sub,
            type: bundleType,
            value: bundleValue,
            ttl: bundleTTL,
          }),
        }
      );
      setMsg(`已创建 ${res.subdomain.full_domain}，扣费 ${res.charged}`);
      setSub("");
      setBundleValue("");
      await refresh();
      await loadMy();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "创建失败");
    } finally {
      setLoading(false);
    }
  };

  const addRecord = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const res = await api<{ charged: number }>("/api/records", {
        method: "POST",
        body: JSON.stringify({ subdomain_id: selectedSub, type: recType, value: recValue, ttl: recTTL }),
      });
      setMsg(`新增成功，扣费 ${res.charged}`);
      setRecValue("");
      await loadMy();
      await refresh();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "新增失败");
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
      await loadMy();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "修改失败");
    }
  };

  const delRecord = async (id: number) => {
    if (!confirm("确认删除该记录？不退积分")) return;
    await api(`/api/records/${id}`, { method: "DELETE" });
    await loadMy();
  };

  const delSub = async (id: number) => {
    if (!confirm("删除子域将同步删除其全部解析记录，确认？")) return;
    await api(`/api/subdomains/${id}`, { method: "DELETE" });
    setSelectedSub(0);
    await loadMy();
  };

  const filtered = records.filter((r) => !selectedSub || r.subdomain_id === selectedSub);

  return (
    <div className="space-y-10">
      {/* section 1: create new subdomain */}
      <section>
        <h1 className="text-2xl font-semibold">DNS 解析</h1>
        <p className="muted mt-1">子域 + 首条 DNS 记录一次提交，只扣一次费</p>

        {err && <div className="error mt-3">{err}</div>}
        {msg && <div className="success mt-3">{msg}</div>}

        <div className="mt-4 space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="font-medium">可用域名</h2>
            <button className="btn" disabled={!domains.length} onClick={() => setModalOpen(true)}>
              新建子域
            </button>
          </div>
          <div className="card overflow-x-auto">
            <table className="table">
              <thead>
                <tr>
                  <th>域名</th>
                  <th>单价</th>
                  <th>说明</th>
                  <th>允许类型</th>
                </tr>
              </thead>
              <tbody>
                {domains.map((d) => (
                  <tr key={d.id}>
                    <td className="text-indigo-300">{d.name}</td>
                    <td>{d.points_cost} 积分/次</td>
                    <td className="muted">{d.description || "-"}</td>
                    <td className="muted">{d.record_types}</td>
                  </tr>
                ))}
                {!domains.length && (
                  <tr>
                    <td colSpan={4} className="muted">暂无上架域名，请联系管理员</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* modal */}
        {modalOpen && (
          <div className="modal-backdrop" onClick={() => { setModalOpen(false); setErr(""); }}>
            <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold">新建子域</h2>
                <button className="btn-ghost text-sm" onClick={() => { setModalOpen(false); setErr(""); }}>&times;</button>
              </div>

              {err && <div className="error mb-3">{err}</div>}

              <form className="grid gap-3" onSubmit={onSubmit}>
                <div>
                  <label className="label">根域名</label>
                  <select className="input" value={domainId} onChange={(e) => setDomainId(Number(e.target.value))}>
                    {domains.map((d) => (
                      <option key={d.id} value={d.id}>{d.name}（{d.points_cost} 积分）</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="label">子域前缀</label>
                  <div className="flex items-center gap-2">
                    <input className="input" value={sub} onChange={(e) => setSub(e.target.value)} placeholder="blog" />
                    <span className="muted whitespace-nowrap">.{selected?.name || "example.com"}</span>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="label">记录类型</label>
                    <select className="input" value={bundleType} onChange={(e) => setBundleType(e.target.value)}>
                      {types.map((t) => (<option key={t}>{t}</option>))}
                    </select>
                  </div>
                  <div>
                    <label className="label">TTL</label>
                    <input className="input" type="number" value={bundleTTL} onChange={(e) => setBundleTTL(Number(e.target.value))} />
                  </div>
                </div>
                <div>
                  <label className="label">记录值</label>
                  <input className="input" value={bundleValue} onChange={(e) => setBundleValue(e.target.value)} placeholder="1.2.3.4 或 cname 目标" />
                </div>
                <div className="flex gap-3">
                  <button className="btn flex-1" type="submit" disabled={loading}>
                    {loading ? "提交中…" : "提交并扣费"}
                  </button>
                  <button className="btn-ghost" type="button" onClick={() => { setModalOpen(false); setErr(""); }}>取消</button>
                </div>
              </form>
            </div>
          </div>
        )}
      </section>

      {/* section 2: my subdomains + records */}
      <section>
        <h2 className="text-xl font-semibold">我的子域</h2>
        <p className="muted mt-1">管理已有子域的解析记录</p>

        <div className="mt-4 grid gap-4 lg:grid-cols-[260px_1fr]">
          <div className="card">
            <h3 className="mb-3 font-medium">子域列表</h3>
            <div className="space-y-2">
              {subs.map((s) => (
                <div
                  key={s.id}
                  className={`rounded-xl border px-3 py-2 ${selectedSub === s.id ? "border-indigo-400/50 bg-indigo-500/10" : "border-white/10"}`}
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
                <select className="input" value={recType} onChange={(e) => setRecType(e.target.value)}>
                  {["A", "AAAA", "CNAME", "TXT", "MX"].map((t) => (<option key={t}>{t}</option>))}
                </select>
              </div>
              <div className="md:col-span-2">
                <label className="label">值（新增将扣费）</label>
                <input className="input" value={recValue} onChange={(e) => setRecValue(e.target.value)} />
              </div>
              <div>
                <label className="label">TTL</label>
                <input className="input" type="number" value={recTTL} onChange={(e) => setRecTTL(Number(e.target.value))} />
              </div>
              <div className="md:col-span-4">
                <button className="btn" disabled={!selectedSub}>新增记录</button>
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
                          <input className="input" value={editValue} onChange={(e) => setEditValue(e.target.value)} />
                        ) : (r.value)}
                      </td>
                      <td>{r.ttl}</td>
                      <td className="space-x-2">
                        {editId === r.id ? (
                          <button className="btn-ghost" onClick={() => saveEdit(r.id)}>保存</button>
                        ) : (
                          <button className="btn-ghost" onClick={() => { setEditId(r.id); setEditValue(r.value); }}>
                            修改
                          </button>
                        )}
                        <button className="btn-ghost" onClick={() => delRecord(r.id)}>删除</button>
                      </td>
                    </tr>
                  ))}
                  {!filtered.length && (
                    <tr><td colSpan={5} className="muted">暂无记录</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
