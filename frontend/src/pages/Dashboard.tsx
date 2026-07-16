import { useAuth } from "../lib/auth";
import { Link } from "react-router-dom";

export default function Dashboard() {
  const { user } = useAuth();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">仪表盘</h1>
        <p className="muted mt-1">一步创建子域名并添加首条解析，只扣一次积分</p>
      </div>
      <div className="grid gap-4 md:grid-cols-3">
        <div className="card">
          <div className="muted">当前积分</div>
          <div className="mt-2 text-3xl font-semibold">{user?.points ?? 0}</div>
        </div>
        <div className="card">
          <div className="muted">快捷操作</div>
          <div className="mt-3 flex flex-wrap gap-2">
            <Link className="btn" to="/domains">
              创建子域
            </Link>
            <Link className="btn-ghost" to="/points">
              充值积分
            </Link>
          </div>
        </div>
        <div className="card">
          <div className="muted">计费说明</div>
          <ul className="mt-2 space-y-1 text-sm text-slate-300">
            <li>· 申请子域 + 首条解析：扣 1 次</li>
            <li>· 同一子域再新增记录：再扣费</li>
            <li>· 修改记录：免费</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
