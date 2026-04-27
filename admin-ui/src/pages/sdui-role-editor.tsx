import { SduiAdminShell } from "../sdui/admin-shell";
import RoleEditorPage from "./role-editor";

export function SduiRoleEditorPage() {
  return (
    <SduiAdminShell>
      <RoleEditorPage />
    </SduiAdminShell>
  );
}
