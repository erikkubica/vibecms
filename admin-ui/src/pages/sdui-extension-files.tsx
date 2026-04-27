import { useParams } from "react-router-dom";
import FileBrowser from "@/components/file-browser";
import { SduiAdminShell } from "../sdui/admin-shell";

export function SduiExtensionFilesPage() {
  const { slug } = useParams<{ slug: string }>();

  return (
    <SduiAdminShell mainClassName="flex-1 overflow-hidden">
      <FileBrowser
        apiBase={`/admin/api/extensions/${slug}/files`}
        title={slug || "Extension"}
        backUrl="/admin/extensions"
        backLabel="Extensions"
        defaultFile="extension.json"
      />
    </SduiAdminShell>
  );
}
