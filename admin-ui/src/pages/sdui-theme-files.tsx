import { useParams } from "react-router-dom";
import { useEffect, useState } from "react";
import { getTheme, type Theme } from "@/api/client";
import FileBrowser from "@/components/file-browser";
import { SduiAdminShell } from "../sdui/admin-shell";
import { Loader2 } from "lucide-react";

export function SduiThemeFilesPage() {
  const { id } = useParams<{ id: string }>();
  const [theme, setTheme] = useState<Theme | null>(null);

  useEffect(() => {
    if (id) getTheme(Number(id)).then(setTheme).catch(() => {});
  }, [id]);

  return (
    <SduiAdminShell mainClassName="flex-1 overflow-hidden">
      {!theme ? (
        <div className="flex h-full items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
        </div>
      ) : (
        <FileBrowser
          apiBase={`/admin/api/themes/${theme.id}/files`}
          title={theme.name}
          backUrl="/admin/themes"
          backLabel="Themes"
          defaultFile="theme.json"
        />
      )}
    </SduiAdminShell>
  );
}
