import { SduiAdminShell } from "../sdui/admin-shell";
import BlockTypeEditorPage from "./block-type-editor";

export function SduiBlockTypeEditorPage() {
  return (
    <SduiAdminShell>
      <BlockTypeEditorPage />
    </SduiAdminShell>
  );
}
