import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";

type Provider = {
  key: string;
  label: string;
  config_fields: { name: string; label: string; required: boolean; secret: boolean }[];
};

type Domain = {
  id: number;
  name: string;
  provider_key: string;
  remote_zone_id: string;
  points_cost: number;
  record_types: string;
  description: string;
  subdomain_ttl_days: number;
  status: number;
};

type Zone = { id: string; domain: string };

export default function AdminDomains() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [providerKey, setProviderKey] = useState("cloudflare");
  const [config, setConfig] = useState<Record<string, string>>({});
  const [zones, setZones] = useState<Zone[]>([]);
  const [name, setName] = useState("");
  const [zoneId, setZoneId] = useState("");
  const [cost, setCost] = useState(10);
  const [types, setTypes] = useState("A,AAAA,CNAME,TXT");
  const [desc, setDesc] = useState("");
  const [subdomainTTL, setSubdomainTTL] = useState(0);
  const [status, setStatus] = useState(1);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  const [editId, setEditId] = useState(0);
  const [modalOpen, setModalOpen] = useState(false);

  const load = async () => {
    const [p, d] = await Promise.all([
      api<Provider[]>("/api/dns/providers"),
      api<Domain[]>("/api/admin/domains"),
    ]);
    setProviders(p || []);
    setDomains(d || []);
    if (p?.length && !providerKey) setProviderKey(p[0].key);
  };

  useEffect(() => {
    load().catch((e) => setErr(e.message));
  }, []);

  const fields = providers.find((p) => p.key === providerKey)?.config_fields || [];

  const resetForm = () => {
    setEditId(0);
    setName("");
    setZoneId("");
    setConfig({});
    setZones([]);
    setCost(10);
    setTypes("A,AAAA,CNAME,TXT");
    setDesc("");
    setSubdomainTTL(0);
    setStatus(1);
    setMsg("");
    setErr("");
    setModalOpen(false);
  };

  const openAdd = () => {
    resetForm();
    setProviderKey(providers[0]?.key || "cloudflare");
    setModalOpen(true);
  };

  const openEdit = (d: Domain) => {
    setEditId(d.id);
    setProviderKey(d.provider_key);
    setName(d.name);
    setZoneId(d.remote_zone_id);
    setConfig({});
    setZones([]);
    setCost(d.points_cost);
    setTypes(d.record_types);
    setDesc(d.description);
    setSubdomainTTL(d.subdomain_ttl_days);
    setStatus(d.status);
    setMsg("");
    setErr("");
    setModalOpen(true);
  };

  const check = async () => {
    setErr("");
    setMsg("");
    try {
      await api("/api/admin/dns/check", {
        method: "POST",
        body: JSON.stringify({ provider_key: providerKey, config }),
      });
      const z = await api<Zone[]>("/api/admin/dns/zones", {
        method: "POST",
        body: JSON.stringify({ provider_key: providerKey, config }),
      });
      setZones(z || []);
      setMsg(`连通成功，拉取到 ${(z || []).length} 个 zone`);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "检测失败");
    }
  };

  const save = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const url = editId ? `/api/admin/domains/${editId}` : "/api/admin/domains";
      const method = editId ? "PUT" : "POST";
      const body: Record<string, unknown> = {
        name: name || zones.find((z) => z.id === zoneId)?.domain,
        provider_key: providerKey,
        remote_zone_id: zoneId,
        record_types: types,
        points_cost: cost,
        description: desc,
        subdomain_ttl_days: subdomainTTL,
        status,
      };
      if (!editId || Object.keys(config).length > 0) {
        body.config = config;
      }
      await api(url, { method, body: JSON.stringify(body) });
      resetForm();
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "保存失败");
    }
  };

  const remove = async (id: number) => {
    if (!confirm("确认删除？")) return;
    await api(`/api/admin/domains/${id}`, { method: "DELETE" });
    if (editId === id) resetForm();
    await load();
  };

  const title = editId ? "编辑域名" : "添加域名";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">域名管理</h1>
          <p className="muted mt-1">配置 DNS 平台密钥并管理可解析根域名</p>
        </div>
        <button className="btn" onClick={openAdd}>
          添加域名
        </button>
      </div>
      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      {/* modal */}
      {modalOpen && (
        <div className="modal-backdrop" onClick={resetForm}>
          <div className="modal-panel" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">{title}</h2>
              <button type="button" className="btn-ghost text-sm" onClick={resetForm}>
                &times;
              </button>
            </div>

            {err && <div className="error mb-3">{err}</div>}
            {msg && <div className="success mb-3">{msg}</div>}

            <form className="grid gap-3 md:grid-cols-2" onSubmit={save}>
              {!editId && (
                <>
                  <div className="md:col-span-2">
                    <label className="label">DNS 平台</label>
                    <select
                      className="input"
                      value={providerKey}
                      onChange={(e) => {
                        setProviderKey(e.target.value);
                        setConfig({});
                        setZones([]);
                      }}
                    >
                      {providers.map((p) => (
                        <option key={p.key} value={p.key}>{p.label}</option>
                      ))}
                    </select>
                  </div>
                  {fields.map((f) => (
                    <div key={f.name} className="md:col-span-2 sm:col-span-1">
                      <label className="label">{f.label}</label>
                      <input
                        className="input"
                        type={f.secret ? "password" : "text"}
                        value={config[f.name] || ""}
                        onChange={(e) => setConfig((c) => ({ ...c, [f.name]: e.target.value }))}
                      />
                    </div>
                  ))}
                  <div className="md:col-span-2">
                    <button type="button" className="btn-ghost" onClick={check}>
                      检测并拉取 Zone
                    </button>
                  </div>
                  <div className="md:col-span-2">
                    <label className="label">Zone</label>
                    <select
                      className="input"
                      value={zoneId}
                      onChange={(e) => {
                        setZoneId(e.target.value);
                        const z = zones.find((x) => x.id === e.target.value);
                        if (z) setName(z.domain);
                      }}
                    >
                      <option value="">选择 zone</option>
                      {zones.map((z) => (
                        <option key={z.id} value={z.id}>{z.domain}</option>
                      ))}
                    </select>
                  </div>
                </>
              )}

              <div>
                <label className="label">域名</label>
                <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div>
                <label className="label">每次新增记录积分</label>
                <input className="input" type="number" value={cost} onChange={(e) => setCost(Number(e.target.value))} />
              </div>
              <div>
                <label className="label">允许记录类型</label>
                <input className="input" value={types} onChange={(e) => setTypes(e.target.value)} />
              </div>
              <div>
                <label className="label">状态</label>
                <select className="input" value={status} onChange={(e) => setStatus(Number(e.target.value))}>
                  <option value={1}>启用</option>
                  <option value={0}>禁用</option>
                </select>
              </div>
              <div>
                <label className="label">子域有效期（天，0=永不过期）</label>
                <input className="input" type="number" min={0} value={subdomainTTL} onChange={(e) => setSubdomainTTL(Number(e.target.value))} />
              </div>
              <div className="md:col-span-2">
                <label className="label">说明</label>
                <input className="input" value={desc} onChange={(e) => setDesc(e.target.value)} />
              </div>

              {editId > 0 && (
                <div className="md:col-span-2">
                  <div className="muted text-xs mb-2">如需更换 DNS 平台密钥，填写下方字段后保存</div>
                  <div className="grid gap-3 md:grid-cols-2">
                    {fields.map((f) => (
                      <div key={f.name}>
                        <label className="label">{f.label}</label>
                        <input
                          className="input"
                          type={f.secret ? "password" : "text"}
                          value={config[f.name] || ""}
                          placeholder="留空则不更新"
                          onChange={(e) => setConfig((c) => ({ ...c, [f.name]: e.target.value }))}
                        />
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <div className="md:col-span-2 flex gap-3">
                <button className="btn flex-1" type="submit">{editId ? "保存修改" : "添加"}</button>
                <button className="btn-ghost" type="button" onClick={resetForm}>取消</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* table */}
      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>域名</th>
              <th>平台</th>
              <th>单价</th>
              <th>类型</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {domains.map((d) => (
              <tr key={d.id}>
                <td>{d.name}</td>
                <td>{d.provider_key}</td>
                <td>{d.points_cost}</td>
                <td>{d.record_types}</td>
                <td>{d.status === 1 ? "启用" : "禁用"}</td>
                <td className="space-x-2">
                  <button className="btn-ghost" onClick={() => openEdit(d)}>修改</button>
                  <button className="btn-ghost" onClick={() => remove(d.id)}>删除</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
