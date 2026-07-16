import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "./lib/auth";
import InstallPage from "./pages/InstallPage";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import Dashboard from "./pages/Dashboard";
import DomainsPage from "./pages/DomainsPage";
import PointsPage from "./pages/PointsPage";
import AdminUsers from "./pages/admin/AdminUsers";
import AdminDomains from "./pages/admin/AdminDomains";
import AdminRedeem from "./pages/admin/AdminRedeem";
import AdminOrders from "./pages/admin/AdminOrders";
import AdminSettings from "./pages/admin/AdminSettings";
import Layout from "./components/Layout";
import { useEffect, useState } from "react";
import { api } from "./lib/api";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  if (loading) return <div className="p-10 muted">加载中…</div>;
  if (!user) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function RequireAdmin({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  if (!user || user.role !== "admin") return <Navigate to="/" replace />;
  return <>{children}</>;
}

export default function App() {
  const [installed, setInstalled] = useState<boolean | null>(null);

  useEffect(() => {
    api<{ installed: boolean }>("/api/install/status")
      .then((d) => setInstalled(d.installed))
      .catch(() => setInstalled(false));
  }, []);

  if (installed === null) {
    return <div className="p-10 muted">检查安装状态…</div>;
  }

  if (!installed) {
    return (
      <Routes>
        <Route path="/install" element={<InstallPage onDone={() => setInstalled(true)} />} />
        <Route path="*" element={<Navigate to="/install" replace />} />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <Layout />
          </RequireAuth>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="domains" element={<DomainsPage />} />
        <Route path="points" element={<PointsPage />} />
        <Route
          path="admin/users"
          element={
            <RequireAdmin>
              <AdminUsers />
            </RequireAdmin>
          }
        />
        <Route
          path="admin/domains"
          element={
            <RequireAdmin>
              <AdminDomains />
            </RequireAdmin>
          }
        />
        <Route
          path="admin/redeem"
          element={
            <RequireAdmin>
              <AdminRedeem />
            </RequireAdmin>
          }
        />
        <Route
          path="admin/orders"
          element={
            <RequireAdmin>
              <AdminOrders />
            </RequireAdmin>
          }
        />
        <Route
          path="admin/settings"
          element={
            <RequireAdmin>
              <AdminSettings />
            </RequireAdmin>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
