import { SduiAdminShell } from "../sdui/admin-shell";
import MenuEditorPage from "./menu-editor";

export function SduiMenuEditorPage() {
  return (
    <SduiAdminShell>
      <MenuEditorPage />
    </SduiAdminShell>
  );
}
