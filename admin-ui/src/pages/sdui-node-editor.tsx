import { useParams } from "react-router-dom";
import { SduiAdminShell } from "../sdui/admin-shell";
import NodeEditorPage from "./node-editor";
import { useAuth } from "@/hooks/use-auth";
import { getNodeAccess } from "@/api/client";
import { Navigate } from "react-router-dom";

interface Props {
  nodeTypeProp?: string;
}

export function SduiNodeEditorPage({ nodeTypeProp }: Props) {
  const { nodeType } = useParams<{ nodeType: string }>();
  const type = nodeTypeProp || nodeType || "page";
  const { user } = useAuth();
  const access = getNodeAccess(user, type);

  if (access.access !== "write" && access.access !== "all") {
    return <Navigate to="/admin/dashboard" replace />;
  }

  return (
    <SduiAdminShell>
      <NodeEditorPage nodeTypeProp={type} />
    </SduiAdminShell>
  );
}
