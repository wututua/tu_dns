import { useEffect, useState } from "react";
import { api } from "../lib/api";

type Notification = {
  id: number;
  title: string;
  content: string;
  read: boolean;
  link: string;
  created_at: string;
};

export default function NotificationsPage() {
  const [items, setItems] = useState<Notification[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  const load = async (p: number) => {
    const data = await api<{ items: Notification[]; total: number }>(`/api/notifications?page=${p}`);
    setItems(data.items || []);
    setTotal(data.total || 0);
  };

  useEffect(() => {
    load(page);
  }, [page]);

  const markRead = async (id: number) => {
    await api(`/api/notifications/${id}/read`, { method: "PUT" });
    await load(page);
  };

  const markAllRead = async () => {
    await api("/api/notifications/read-all", { method: "PUT" });
    await load(page);
  };

  const totalPages = Math.ceil(total / 20);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">通知</h1>
          <p className="muted mt-1">{total} 条通知</p>
        </div>
        <button className="btn-ghost" onClick={markAllRead}>
          全部已读
        </button>
      </div>

      <div className="space-y-2">
        {items.map((n) => (
          <div
            key={n.id}
            className={`card flex items-start gap-3 ${!n.read ? "border-indigo-400/30" : ""}`}
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                {!n.read && <span className="w-2 h-2 rounded-full bg-indigo-400 shrink-0" />}
                <span className="font-medium">{n.title}</span>
              </div>
              <p className="muted mt-1 text-sm">{n.content}</p>
              <p className="muted mt-1 text-xs">{new Date(n.created_at).toLocaleString("zh-CN")}</p>
            </div>
            {!n.read && (
              <button className="btn-ghost text-xs shrink-0" onClick={() => markRead(n.id)}>
                标为已读
              </button>
            )}
          </div>
        ))}
        {!items.length && <div className="muted p-10 text-center">暂无通知</div>}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button className="btn-ghost" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            上一页
          </button>
          <span className="muted text-sm">{page} / {totalPages}</span>
          <button className="btn-ghost" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
            下一页
          </button>
        </div>
      )}
    </div>
  );
}
