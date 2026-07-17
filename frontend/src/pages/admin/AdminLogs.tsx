import { useEffect, useState } from "react";
import { api } from "../../lib/api";

type Log = {
  id: number;
  user_id: number;
  admin_id: number;
  action: string;
  target_type: string;
  target_id: string;
  ip: string;
  message: string;
  created_at: string;
};

export default function AdminLogs() {
  const [items, setItems] = useState<Log[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [err, setErr] = useState("");

  const load = async (p: number) => {
    const data = await api<{ items: Log[]; total: number }>(`/api/admin/logs?page=${p}`);
    setItems(data.items || []);
    setTotal(data.total || 0);
  };

  useEffect(() => {
    load(page).catch((e) => setErr(e.message));
  }, [page]);

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">操作日志</h1>
      {err && <div className="error">{err}</div>}

      <div className="card overflow-x-auto">
        <table className="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>操作</th>
              <th>对象</th>
              <th>内容</th>
              <th>IP</th>
              <th>时间</th>
            </tr>
          </thead>
          <tbody>
            {items.map((log) => (
              <tr key={log.id}>
                <td className="muted">{log.id}</td>
                <td>{log.action}</td>
                <td className="muted">{log.target_type} #{log.target_id}</td>
                <td className="max-w-xs truncate">{log.message}</td>
                <td className="muted text-xs">{log.ip}</td>
                <td className="muted text-xs">{new Date(log.created_at).toLocaleString("zh-CN")}</td>
              </tr>
            ))}
            {!items.length && (
              <tr>
                <td colSpan={6} className="muted">暂无日志</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button className="btn-ghost" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            上一页
          </button>
          <span className="muted text-sm">
            {page} / {totalPages}
          </span>
          <button className="btn-ghost" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
            下一页
          </button>
        </div>
      )}
    </div>
  );
}
