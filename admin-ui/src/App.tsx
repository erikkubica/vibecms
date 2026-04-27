import { Routes, Route, Navigate, useParams } from "react-router-dom";
import { SduiPage } from "@/components/sdui-page";
import { SduiDashboardPage } from "@/pages/sdui-dashboard";
import { SduiContentTypesPage } from "@/pages/sdui-content-types";
import { SduiTaxonomiesPage } from "@/pages/sdui-taxonomies";
import { SduiNodeListPage } from "@/pages/sdui-node-list";
import { SduiTaxonomyTermsPage } from "@/pages/sdui-taxonomy-terms";
import { SduiTemplatesPage } from "@/pages/sdui-templates";
import { SduiLayoutsPage } from "@/pages/sdui-layouts";
import { SduiBlockTypesPage } from "@/pages/sdui-block-types";
import { SduiLayoutBlocksPage } from "@/pages/sdui-layout-blocks";
import { SduiMenusPage } from "@/pages/sdui-menus";
import { AuthProvider, useAuth } from "@/hooks/use-auth";
import { getNodeAccess } from "@/api/client";
import { SduiThemesPage } from "@/pages/sdui-themes";
import { SduiNodeEditorPage } from "@/pages/sdui-node-editor";
import { SduiExtensionsPage } from "@/pages/sdui-extensions";
import { SduiThemeFilesPage } from "@/pages/sdui-theme-files";
import { SduiExtensionFilesPage } from "@/pages/sdui-extension-files";
import { SduiNodeTypeEditorPage } from "@/pages/sdui-node-type-editor";
import { SduiBlockTypeEditorPage } from "@/pages/sdui-block-type-editor";
import { SduiTemplateEditorPage } from "@/pages/sdui-template-editor";
import { SduiLayoutEditorPage } from "@/pages/sdui-layout-editor";
import { SduiLayoutBlockEditorPage } from "@/pages/sdui-layout-block-editor";
import { SduiMenuEditorPage } from "@/pages/sdui-menu-editor";
import { SduiUserEditorPage } from "@/pages/sdui-user-editor";
import { SduiRoleEditorPage } from "@/pages/sdui-role-editor";
import { SduiTermEditorPage } from "@/pages/sdui-term-editor";
import { SduiTaxonomyEditorPage } from "@/pages/sdui-taxonomy-editor";
import { SduiSiteSettingsPage } from "@/pages/sdui-site-settings";
import { SduiLanguagesPage } from "@/pages/sdui-languages";
import { SduiUsersPage } from "@/pages/sdui-users";
import { SduiRolesPage } from "@/pages/sdui-roles";
import { SduiMcpTokensPage } from "@/pages/sdui-mcp-tokens";
import LoginPage from "@/pages/login";
import { SduiAdminShell } from "@/sdui/admin-shell";
import { AdminLanguageProvider } from "@/hooks/use-admin-language";
import { ExtensionsProvider } from "@/hooks/use-extensions";
import { ExtensionPageLoader } from "@/components/extension-page-loader";
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

function NodeAccessGuard({
  nodeType,
  children,
}: {
  nodeType: string;
  children: React.ReactNode;
}) {
  const { user } = useAuth();
  const access = getNodeAccess(user, nodeType);
  if (access.access === "none") {
    return <Navigate to="/admin/dashboard" replace />;
  }
  return <>{children}</>;
}

function DynamicNodeList() {
  const { nodeType } = useParams<{ nodeType: string }>();
  const type = nodeType || "page";
  return (
    <NodeAccessGuard nodeType={type}>
      <SduiNodeListPage />
    </NodeAccessGuard>
  );
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/admin/login" element={<LoginPage />} />

      {/* Dashboard */}
      <Route path="/admin/dashboard" element={<ProtectedRoute><SduiDashboardPage /></ProtectedRoute>} />

      {/* Content lists */}
      <Route path="/admin/content-types" element={<ProtectedRoute><SduiContentTypesPage /></ProtectedRoute>} />
      <Route path="/admin/taxonomies" element={<ProtectedRoute><SduiTaxonomiesPage /></ProtectedRoute>} />
      <Route path="/admin/content/:nodeType" element={<ProtectedRoute><DynamicNodeList /></ProtectedRoute>} />
      <Route path="/admin/content/:nodeType/taxonomies/:taxonomy" element={<ProtectedRoute><SduiTaxonomyTermsPage /></ProtectedRoute>} />
      <Route path="/admin/pages" element={<ProtectedRoute><SduiNodeListPage nodeTypeOverride="page" /></ProtectedRoute>} />
      <Route path="/admin/posts" element={<ProtectedRoute><SduiNodeListPage nodeTypeOverride="post" /></ProtectedRoute>} />
      <Route path="/admin/templates" element={<ProtectedRoute><SduiTemplatesPage /></ProtectedRoute>} />
      <Route path="/admin/layouts" element={<ProtectedRoute><SduiLayoutsPage /></ProtectedRoute>} />
      <Route path="/admin/block-types" element={<ProtectedRoute><SduiBlockTypesPage /></ProtectedRoute>} />
      <Route path="/admin/layout-blocks" element={<ProtectedRoute><SduiLayoutBlocksPage /></ProtectedRoute>} />
      <Route path="/admin/menus" element={<ProtectedRoute><SduiMenusPage /></ProtectedRoute>} />
      <Route path="/admin/themes" element={<ProtectedRoute><SduiThemesPage /></ProtectedRoute>} />
      <Route path="/admin/extensions" element={<ProtectedRoute><SduiExtensionsPage /></ProtectedRoute>} />

      {/* Node editor */}
      <Route path="/admin/pages/new" element={<ProtectedRoute><SduiNodeEditorPage nodeTypeProp="page" /></ProtectedRoute>} />
      <Route path="/admin/pages/:id/edit" element={<ProtectedRoute><SduiNodeEditorPage nodeTypeProp="page" /></ProtectedRoute>} />
      <Route path="/admin/posts/new" element={<ProtectedRoute><SduiNodeEditorPage nodeTypeProp="post" /></ProtectedRoute>} />
      <Route path="/admin/posts/:id/edit" element={<ProtectedRoute><SduiNodeEditorPage nodeTypeProp="post" /></ProtectedRoute>} />
      <Route path="/admin/content/:nodeType/new" element={<ProtectedRoute><SduiNodeEditorPage /></ProtectedRoute>} />
      <Route path="/admin/content/:nodeType/:id/edit" element={<ProtectedRoute><SduiNodeEditorPage /></ProtectedRoute>} />

      {/* Content type editors */}
      <Route path="/admin/content-types/new" element={<ProtectedRoute><SduiNodeTypeEditorPage /></ProtectedRoute>} />
      <Route path="/admin/content-types/:id/edit" element={<ProtectedRoute><SduiNodeTypeEditorPage /></ProtectedRoute>} />

      {/* Block type editors */}
      <Route path="/admin/block-types/new" element={<ProtectedRoute><SduiBlockTypeEditorPage /></ProtectedRoute>} />
      <Route path="/admin/block-types/:id/edit" element={<ProtectedRoute><SduiBlockTypeEditorPage /></ProtectedRoute>} />

      {/* Template editors */}
      <Route path="/admin/templates/new" element={<ProtectedRoute><SduiTemplateEditorPage /></ProtectedRoute>} />
      <Route path="/admin/templates/:id/edit" element={<ProtectedRoute><SduiTemplateEditorPage /></ProtectedRoute>} />

      {/* Layout editors */}
      <Route path="/admin/layouts/new" element={<ProtectedRoute><SduiLayoutEditorPage /></ProtectedRoute>} />
      <Route path="/admin/layouts/:id" element={<ProtectedRoute><SduiLayoutEditorPage /></ProtectedRoute>} />

      {/* Layout block editors */}
      <Route path="/admin/layout-blocks/new" element={<ProtectedRoute><SduiLayoutBlockEditorPage /></ProtectedRoute>} />
      <Route path="/admin/layout-blocks/:id" element={<ProtectedRoute><SduiLayoutBlockEditorPage /></ProtectedRoute>} />
      <Route path="/admin/layout-blocks/:id/edit" element={<ProtectedRoute><SduiLayoutBlockEditorPage /></ProtectedRoute>} />

      {/* Menu editors */}
      <Route path="/admin/menus/new" element={<ProtectedRoute><SduiMenuEditorPage /></ProtectedRoute>} />
      <Route path="/admin/menus/:id" element={<ProtectedRoute><SduiMenuEditorPage /></ProtectedRoute>} />
      <Route path="/admin/menus/:id/edit" element={<ProtectedRoute><SduiMenuEditorPage /></ProtectedRoute>} />

      {/* Taxonomy editors */}
      <Route path="/admin/taxonomies/new" element={<ProtectedRoute><SduiTaxonomyEditorPage /></ProtectedRoute>} />
      <Route path="/admin/taxonomies/:slug/edit" element={<ProtectedRoute><SduiTaxonomyEditorPage /></ProtectedRoute>} />

      {/* Term editors */}
      <Route path="/admin/content/:nodeType/taxonomies/:taxonomy/new" element={<ProtectedRoute><SduiTermEditorPage /></ProtectedRoute>} />
      <Route path="/admin/content/:nodeType/taxonomies/:taxonomy/:id/edit" element={<ProtectedRoute><SduiTermEditorPage /></ProtectedRoute>} />

      {/* Settings & admin pages */}
      <Route path="/admin/settings/site" element={<ProtectedRoute><SduiSiteSettingsPage /></ProtectedRoute>} />
      <Route path="/admin/languages" element={<ProtectedRoute><SduiLanguagesPage /></ProtectedRoute>} />
      <Route path="/admin/users" element={<ProtectedRoute><SduiUsersPage /></ProtectedRoute>} />
      <Route path="/admin/users/new" element={<ProtectedRoute><SduiUserEditorPage /></ProtectedRoute>} />
      <Route path="/admin/users/:id/edit" element={<ProtectedRoute><SduiUserEditorPage /></ProtectedRoute>} />
      <Route path="/admin/roles" element={<ProtectedRoute><SduiRolesPage /></ProtectedRoute>} />
      <Route path="/admin/roles/new" element={<ProtectedRoute><SduiRoleEditorPage /></ProtectedRoute>} />
      <Route path="/admin/roles/:id/edit" element={<ProtectedRoute><SduiRoleEditorPage /></ProtectedRoute>} />
      <Route path="/admin/mcp-tokens" element={<ProtectedRoute><SduiMcpTokensPage /></ProtectedRoute>} />

      {/* File viewers */}
      <Route path="/admin/themes/:id/files" element={<ProtectedRoute><SduiThemeFilesPage /></ProtectedRoute>} />
      <Route path="/admin/extensions/:slug/files" element={<ProtectedRoute><SduiExtensionFilesPage /></ProtectedRoute>} />

      {/* Extension pages */}
      <Route
        path="/admin/ext/:slug/*"
        element={
          <ProtectedRoute>
            <AdminLanguageProvider>
              <SduiAdminShell>
                <ExtensionPageLoader />
              </SduiAdminShell>
            </AdminLanguageProvider>
          </ProtectedRoute>
        }
      />

      {/* SDUI test route */}
      <Route path="/admin/sdui/:page" element={<ProtectedRoute><SduiAdminShell><SduiPage /></SduiAdminShell></ProtectedRoute>} />

      <Route path="/admin" element={<Navigate to="/admin/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/admin/dashboard" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <ExtensionsProvider>
        <AppRoutes />
      </ExtensionsProvider>
    </AuthProvider>
  );
}
