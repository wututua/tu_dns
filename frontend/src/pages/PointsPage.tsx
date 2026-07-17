import { FormEvent, useEffect, useState } from "react";
import { api } from "../lib/api";
import { useAuth } from "../lib/auth";

type Ledger = {
  id: number;
  delta: number;
  balance: number;
  type: string;
  remark: string;
  created_at: string;
};

type Order = {
  id: number;
  out_trade_no: string;
  amount_cent: number;
  points: number;
  status: string;
  pay_url: string;
};

export default function PointsPage() {
  const { user, refresh } = useAuth();
  const [items, setItems] = useState<Ledger[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [code, setCode] = useState("");
  const [amount, setAmount] = useState(10);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  const load = async (p: number) => {
    const data = await api<{ items: Ledger[]; total: number }>(`/api/points?page=${p}`);
    setItems(data.items || []);
    setTotal(data.total || 0);
  };

  useEffect(() => {
    load(page).catch((e) => setErr(e.message));
  }, [page]);

  const redeem = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const res = await api<{ gained: number }>("/api/redeem", {
        method: "POST",
        body: JSON.stringify({ code }),
      });
      setMsg(`兑换成功 +${res.gained}`);
      setCode("");
      await refresh();
      await load(page);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "兑换失败");
    }
  };

  const createPay = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      const order = await api<Order>("/api/pay/alipay/create", {
        method: "POST",
        body: JSON.stringify({ amount }),
      });
      if (order.pay_url?.startsWith("/pay/mock")) {
        await api("/api/pay/alipay/mock", {
          method: "POST",
          body: JSON.stringify({ out_trade_no: order.out_trade_no }),
        });
        setMsg(`模拟支付成功，到账 ${order.points} 积分`);
        await refresh();
        await load(page);
      } else if (order.pay_url) {
        window.open(order.pay_url, "_blank");
        setMsg("已打开支付页面，完成支付后刷新积分");
      }
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "创建订单失败");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">积分中心</h1>
        <p className="muted mt-1">当前余额 {user?.points ?? 0}</p>
      </div>

      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}

      <div className="grid gap-4 md:grid-cols-2">
        <form className="card space-y-3" onSubmit={redeem}>
          <h2 className="font-medium">兑换码</h2>
          <input className="input" value={code} onChange={(e) => setCode(e.target.value)} placeholder="输入兑换码" />
          <button className="btn">兑换</button>
        </form>
        <form className="card space-y-3" onSubmit={createPay}>
          <h2 className="font-medium">支付宝充值</h2>
          <p className="muted">未配置密钥时走模拟支付，便于本地调试</p>
          <input
            className="input"
            type="number"
            min={1}
            value={amount}
            onChange={(e) => setAmount(Number(e.target.value))}
          />
          <button className="btn">创建订单</button>
        </form>
      </div>

      <div className="card overflow-x-auto">
        <h2 className="mb-3 font-medium">积分流水</h2>
        <table className="table">
          <thead>
            <tr>
              <th>变动</th>
              <th>余额</th>
              <th>类型</th>
              <th>备注</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id}>
                <td className={it.delta >= 0 ? "text-emerald-400" : "text-rose-400"}>
                  {it.delta >= 0 ? `+${it.delta}` : it.delta}
                </td>
                <td>{it.balance}</td>
                <td>{it.type}</td>
                <td>{it.remark}</td>
                <td className="muted">{new Date(it.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {!items.length && (
              <tr>
                <td colSpan={5} className="muted">
                  暂无流水
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

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
    </div>
  );
}
