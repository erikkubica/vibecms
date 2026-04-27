import { SduiAdminShell } from "../sdui/admin-shell";
import NodeTypeEditorPage from "./node-type-editor";

export function SduiNodeTypeEditorPage() {
  return (
    <SduiAdminShell>
      <NodeTypeEditorPage />
    </SduiAdminShell>
  );
}
