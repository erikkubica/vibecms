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
import LayoutsListPage from "@/pages/layouts-list";
import LayoutEditorPage from "@/pages/layout-editor";
import LayoutBlocksListPage from "@/pages/layout-blocks-list";
import LayoutBlockEditorPage from "@/pages/layout-block-editor";
import MenusListPage from "@/pages/menus-list";
import MenuEditorPage from "@/pages/menu-editor";
import UsersPage from "@/pages/users";
import UserEditorPage from "@/pages/user-editor";
import RolesPage from "@/pages/roles";
import RoleEditorPage from "@/pages/role-editor";
import EmailTemplatesPage from "@/pages/email-templates";
import EmailTemplateEditorPage from "@/pages/email-template-editor";
import EmailRulesPage from "@/pages/email-rules";
import EmailRuleEditorPage from "@/pages/email-rule-editor";
import EmailLogsPage from "@/pages/email-logs";
import EmailSettingsPage from "@/pages/email-settings";
import ThemesPage from "@/pages/themes";
import ExtensionsPage from "@/pages/extensions";
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
        <Route path="layouts" element={<LayoutsListPage />} />
        <Route path="layouts/new" element={<LayoutEditorPage />} />
        <Route path="layouts/:id" element={<LayoutEditorPage />} />
        <Route path="layout-blocks" element={<LayoutBlocksListPage />} />
        <Route path="layout-blocks/new" element={<LayoutBlockEditorPage />} />
        <Route path="layout-blocks/:id" element={<LayoutBlockEditorPage />} />
        <Route path="menus" element={<MenusListPage />} />
        <Route path="menus/new" element={<MenuEditorPage />} />
        <Route path="menus/:id" element={<MenuEditorPage />} />
        <Route path="languages" element={<LanguagesPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="users/new" element={<UserEditorPage />} />
        <Route path="users/:id/edit" element={<UserEditorPage />} />
        <Route path="roles" element={<RolesPage />} />
        <Route path="roles/new" element={<RoleEditorPage />} />
        <Route path="roles/:id/edit" element={<RoleEditorPage />} />
        <Route path="email-templates" element={<EmailTemplatesPage />} />
        <Route path="email-templates/new" element={<EmailTemplateEditorPage />} />
        <Route path="email-templates/:id/edit" element={<EmailTemplateEditorPage />} />
        <Route path="email-rules" element={<EmailRulesPage />} />
        <Route path="email-rules/new" element={<EmailRuleEditorPage />} />
        <Route path="email-rules/:id/edit" element={<EmailRuleEditorPage />} />
        <Route path="email-logs" element={<EmailLogsPage />} />
        <Route path="email-settings" element={<EmailSettingsPage />} />
        <Route path="themes" element={<ThemesPage />} />
        <Route path="extensions" element={<ExtensionsPage />} />
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
