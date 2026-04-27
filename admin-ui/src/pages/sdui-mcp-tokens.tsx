import { SduiAdminShell } from "../sdui/admin-shell";
import McpTokensPage from "./mcp-tokens";

export function SduiMcpTokensPage() {
  return (
    <SduiAdminShell>
      <McpTokensPage />
    </SduiAdminShell>
  );
}
