import { useEffect, useState } from "react";
import { api } from "../../lib/api";

type Order = {
  id: number;
  user_id: number;
  out_trade_no: string;
  amount_cent: number;
  points: number;
  status: string;
  created_at: string;
};

export default function AdminOrders() {
  const [items, setItems] = useState<Order[]>([]);

  useEffect(() => {
    api<Order[]>("/api/admin/pay/orders").then((d) => setItems(d || []));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">支付订单</h1>
      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>订单号</th>
              <th>用户</th>
              <th>金额</th>
              <th>积分</th>
              <th>状态</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {items.map((o) => (
              <tr key={o.id}>
                <td className="font-mono text-xs">{o.out_trade_no}</td>
                <td>{o.user_id}</td>
                <td>¥{(o.amount_cent / 100).toFixed(2)}</td>
                <td>{o.points}</td>
                <td>{o.status}</td>
                <td className="muted">{new Date(o.created_at).toLocaleString()}</td>
              </tr>
            ))}
            {!items.length && (
              <tr>
                <td colSpan={6} className="muted">
                  暂无订单
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
