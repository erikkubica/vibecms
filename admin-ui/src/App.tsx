import { Routes, Route, Navigate, useParams } from "react-router-dom";
import { AuthProvider, useAuth } from "@/hooks/use-auth";
import AdminLayout from "@/components/layout/admin-layout";
import LoginPage from "@/pages/login";
import DashboardPage from "@/pages/dashboard";
import NodesListPage from "@/pages/nodes-list";
import NodeEditorPage from "@/pages/node-editor";
import NodeTypesListPage from "@/pages/node-types-list";
import NodeTypeEditorPage from "@/pages/node-type-editor";
import BlockTypesListPage from "@/pages/block-types-list";
import BlockTypeEditorPage from "@/pages/block-type-editor";
import TemplatesListPage from "@/pages/templates-list";
import TemplateEditorPage from "@/pages/template-editor";
import LanguagesPage from "@/pages/languages";
import { AdminLanguageProvider } from "@/hooks/use-admin-language";
import { Loader2 } from "lucide-react";

function DynamicNodeList() {
  const { nodeType } = useParams<{ nodeType: string }>();
  return <NodesListPage nodeType={nodeType || "page"} />;
}

function DynamicNodeEditor() {
  const { nodeType } = useParams<{ nodeType: string }>();
  return <NodeEditorPage nodeType={nodeType || "page"} />;
}

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
            <AdminLanguageProvider>
              <AdminLayout />
            </AdminLanguageProvider>
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
        <Route path="content-types" element={<NodeTypesListPage />} />
        <Route path="content-types/new" element={<NodeTypeEditorPage />} />
        <Route path="content-types/:id/edit" element={<NodeTypeEditorPage />} />
        <Route path="block-types" element={<BlockTypesListPage />} />
        <Route path="block-types/new" element={<BlockTypeEditorPage />} />
        <Route path="block-types/:id/edit" element={<BlockTypeEditorPage />} />
        <Route path="templates" element={<TemplatesListPage />} />
        <Route path="templates/new" element={<TemplateEditorPage />} />
        <Route path="templates/:id/edit" element={<TemplateEditorPage />} />
        <Route path="languages" element={<LanguagesPage />} />
        <Route
          path="content/:nodeType"
          element={<DynamicNodeList />}
        />
        <Route
          path="content/:nodeType/new"
          element={<DynamicNodeEditor />}
        />
        <Route
          path="content/:nodeType/:id/edit"
          element={<DynamicNodeEditor />}
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
