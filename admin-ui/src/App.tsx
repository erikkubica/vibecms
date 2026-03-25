import { Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider, useAuth } from "@/hooks/use-auth";
import AdminLayout from "@/components/layout/admin-layout";
import LoginPage from "@/pages/login";
import DashboardPage from "@/pages/dashboard";
import NodesListPage from "@/pages/nodes-list";
import NodeEditorPage from "@/pages/node-editor";
import { Loader2 } from "lucide-react";

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { loading, isAuthenticated } = useAuth();

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center bg-slate-50">
        <Loader2 className="h-10 w-10 animate-spin text-primary-500" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/admin/login" replace />;
  }

  return <>{children}</>;
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/admin/login" element={<LoginPage />} />
      <Route
        path="/admin"
        element={
          <ProtectedRoute>
            <AdminLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/admin/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route
          path="pages"
          element={<NodesListPage nodeType="page" />}
        />
        <Route
          path="pages/new"
          element={<NodeEditorPage nodeType="page" />}
        />
        <Route
          path="pages/:id/edit"
          element={<NodeEditorPage nodeType="page" />}
        />
        <Route
          path="posts"
          element={<NodesListPage nodeType="post" />}
        />
        <Route
          path="posts/new"
          element={<NodeEditorPage nodeType="post" />}
        />
        <Route
          path="posts/:id/edit"
          element={<NodeEditorPage nodeType="post" />}
        />
      </Route>
      <Route path="*" element={<Navigate to="/admin/dashboard" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <AppRoutes />
    </AuthProvider>
  );
}
