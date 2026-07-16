import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";

type Code = {
  id: number;
  code: string;
  points: number;
  max_uses: number;
  used_count: number;
  enabled: boolean;
};

export default function AdminRedeem() {
  const [items, setItems] = useState<Code[]>([]);
  const [points, setPoints] = useState(100);
  const [maxUses, setMaxUses] = useState(1);
  const [msg, setMsg] = useState("");

  const load = async () => {
    setItems((await api<Code[]>("/api/admin/redeem")) || []);
  };

  useEffect(() => {
    load();
  }, []);

  const create = async (e: FormEvent) => {
    e.preventDefault();
    const item = await api<Code>("/api/admin/redeem", {
      method: "POST",
      body: JSON.stringify({ points, max_uses: maxUses }),
    });
    setMsg(`已生成：${item.code}`);
    await load();
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">兑换码</h1>
      {msg && <div className="success">{msg}</div>}
      <form className="card grid gap-3 md:grid-cols-3" onSubmit={create}>
        <div>
          <label className="label">积分</label>
          <input className="input" type="number" value={points} onChange={(e) => setPoints(Number(e.target.value))} />
        </div>
        <div>
          <label className="label">可用次数</label>
          <input className="input" type="number" value={maxUses} onChange={(e) => setMaxUses(Number(e.target.value))} />
        </div>
        <div className="flex items-end">
          <button className="btn">生成</button>
        </div>
      </form>
      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>码</th>
              <th>积分</th>
              <th>使用</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id}>
                <td className="font-mono">{it.code}</td>
                <td>{it.points}</td>
                <td>
                  {it.used_count}/{it.max_uses}
                </td>
                <td>{it.enabled ? "启用" : "停用"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
