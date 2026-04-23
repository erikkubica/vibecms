import { SduiAdminShell } from "../sdui/admin-shell";
import SiteSettingsPage from "./site-settings";

export function SduiSiteSettingsPage() {
  return (
    <SduiAdminShell>
      <SiteSettingsPage />
    </SduiAdminShell>
  );
}
