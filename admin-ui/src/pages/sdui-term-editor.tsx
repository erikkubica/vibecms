import { SduiAdminShell } from "../sdui/admin-shell";
import TermEditorPage from "./term-editor";

export function SduiTermEditorPage() {
  return (
    <SduiAdminShell>
      <TermEditorPage />
    </SduiAdminShell>
  );
}
