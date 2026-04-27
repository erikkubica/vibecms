import { SduiAdminShell } from "../sdui/admin-shell";
import UserEditorPage from "./user-editor";

export function SduiUserEditorPage() {
  return (
    <SduiAdminShell>
      <UserEditorPage />
    </SduiAdminShell>
  );
}
