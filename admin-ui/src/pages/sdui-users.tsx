import { SduiAdminShell } from "../sdui/admin-shell";
import UsersPage from "./users";

export function SduiUsersPage() {
  return (
    <SduiAdminShell>
      <UsersPage />
    </SduiAdminShell>
  );
}
