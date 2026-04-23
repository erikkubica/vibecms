import { SduiAdminShell } from "../sdui/admin-shell";
import TaxonomyEditorPage from "./taxonomy-editor";

export function SduiTaxonomyEditorPage() {
  return (
    <SduiAdminShell>
      <TaxonomyEditorPage />
    </SduiAdminShell>
  );
}
