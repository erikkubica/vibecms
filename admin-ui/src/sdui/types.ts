// Boot Manifest
export interface BootManifest {
  version: string;
  user: BootUser;
  extensions: BootExtension[];
  navigation: NavItem[];
  node_types: BootNodeType[];
}

export interface BootUser {
  id: number;
  email: string;
  full_name?: string;
  role: string;
  capabilities: Record<string, unknown>;
}

export interface BootExtension {
  slug: string;
  name: string;
  entry?: string;
  components?: string[];
}

export interface NavItem {
  id: string;
  label: string;
  icon?: string;
  is_section?: boolean; // true = non-clickable section header
  section?: string; // grouping key for extension items
  path?: string;
  children?: NavItem[];
}

export interface BootNodeType {
  slug: string;
  label: string;
  label_plural: string;
  icon: string;
  supports_blocks: boolean;
}

// Layout Tree
export interface LayoutNode {
  type: string;
  props?: Record<string, unknown>;
  children?: LayoutNode[];
  actions?: Record<string, ActionDef>;
}

export interface ActionDef {
  type:
    | "CORE_API"
    | "NAVIGATE"
    | "TOAST"
    | "INVALIDATE"
    | "CONFIRM"
    | "SEQUENCE"
    | "SET_STORE";
  method?: string;
  params?: Record<string, unknown>;
  message?: string;
  title?: string;
  variant?: string;
  to?: string;
  keys?: string[];
  key?: string;
  value?: unknown;
  steps?: ActionDef[];
  bind?: string;
  then?: ActionDef;
  // CORE_API mutation feedback
  success_message?: string;
  error_message?: string;
  silent?: boolean;
}

// SSE Events — mirrors internal/sdui/types.go SSEEvent.
// See that file for the full taxonomy.
export interface SSEEventData {
  type:
    | "CONNECTED"
    | "NAV_STALE"
    | "ENTITY_CHANGED"
    | "SETTING_CHANGED"
    | "NOTIFY"
    | "UI_STALE";
  entity?: string;
  id?: string | number;
  op?: string;
  key?: string;
  data?: unknown;
}
