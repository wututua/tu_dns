import { FormEvent, useEffect, useState } from "react";
import { api } from "../../lib/api";

export default function AdminSettings() {
  const [siteName, setSiteName] = useState("TuDNS");
  const [enabled, setEnabled] = useState(false);
  const [appId, setAppId] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [publicKey, setPublicKey] = useState("");
  const [notifyURL, setNotifyURL] = useState("");
  const [returnURL, setReturnURL] = useState("");
  const [rate, setRate] = useState(10);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  useEffect(() => {
    api<Record<string, string>>("/api/admin/settings").then((s) => {
      if (s.site_name) setSiteName(s.site_name);
    });
    api<{
      enabled: boolean;
      app_id: string;
      private_key: string;
      alipay_public_key: string;
      notify_url: string;
      return_url: string;
      points_per_yuan: number;
    }>("/api/admin/pay/alipay/config").then((c) => {
      setEnabled(!!c.enabled);
      setAppId(c.app_id || "");
      setPrivateKey(c.private_key || "");
      setPublicKey(c.alipay_public_key || "");
      setNotifyURL(c.notify_url || "");
      setReturnURL(c.return_url || "");
      setRate(c.points_per_yuan || 10);
    });
  }, []);

  const save = async (e: FormEvent) => {
    e.preventDefault();
    setErr("");
    setMsg("");
    try {
      await api("/api/admin/settings", {
        method: "PUT",
        body: JSON.stringify({ site_name: siteName }),
      });
      await api("/api/admin/pay/alipay/config", {
        method: "PUT",
        body: JSON.stringify({
          enabled,
          app_id: appId,
          private_key: privateKey,
          alipay_public_key: publicKey,
          notify_url: notifyURL,
          return_url: returnURL,
          points_per_yuan: rate,
        }),
      });
      setMsg("已保存");
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "保存失败");
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">系统设置</h1>
      {err && <div className="error">{err}</div>}
      {msg && <div className="success">{msg}</div>}
      <form className="card space-y-3 max-w-2xl" onSubmit={save}>
        <div>
          <label className="label">站点名称</label>
          <input className="input" value={siteName} onChange={(e) => setSiteName(e.target.value)} />
        </div>
        <div className="pt-2 font-medium">支付宝（官方）</div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
          启用支付宝充值
        </label>
        <div>
          <label className="label">AppID</label>
          <input className="input" value={appId} onChange={(e) => setAppId(e.target.value)} />
        </div>
        <div>
          <label className="label">应用私钥（留空/*** 表示不修改）</label>
          <textarea className="input min-h-24" value={privateKey} onChange={(e) => setPrivateKey(e.target.value)} />
        </div>
        <div>
          <label className="label">支付宝公钥</label>
          <textarea className="input min-h-24" value={publicKey} onChange={(e) => setPublicKey(e.target.value)} />
        </div>
        <div>
          <label className="label">异步通知 URL</label>
          <input className="input" value={notifyURL} onChange={(e) => setNotifyURL(e.target.value)} />
        </div>
        <div>
          <label className="label">同步跳转 URL</label>
          <input className="input" value={returnURL} onChange={(e) => setReturnURL(e.target.value)} />
        </div>
        <div>
          <label className="label">1 元 = 多少积分</label>
          <input className="input" type="number" value={rate} onChange={(e) => setRate(Number(e.target.value))} />
        </div>
        <button className="btn">保存</button>
      </form>
    </div>
  );
}
