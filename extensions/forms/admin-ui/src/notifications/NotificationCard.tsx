import React, { useState } from "react";
import {
  Trash2,
  Mail,
  User,
  Users,
  Send,
  ChevronDown,
} from "@vibecms/icons";
import NotificationDisplayWhen from "./NotificationDisplayWhen";

const {
  Button,
  Card,
  CardContent,
  SectionHeader,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
  Textarea,
  Popover,
  PopoverContent,
  PopoverTrigger,
  AccordionRow,
} = (window as any).__VIBECMS_SHARED__.ui;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface NotificationCardProps {
  notif: any;
  index: number;
  formId?: number | string;
  emailFields: any[];
  formFields?: any[];
  onUpdate: (index: number, key: string, value: any) => void;
  onRemove: (index: number) => void;
}

export default function NotificationCard({
  notif,
  index,
  formId,
  emailFields,
  formFields = [],
  onUpdate,
  onRemove,
}: NotificationCardProps) {
  const [expandedCC, setExpandedCC] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [testRecipient, setTestRecipient] = useState("");
  const [testSending, setTestSending] = useState(false);
  const [testPopoverOpen, setTestPopoverOpen] = useState(false);

  const handleSendTest = async () => {
    if (!formId) {
      toast.error("Save the form first before sending a test email.");
      return;
    }
    setTestSending(true);
    try {
      const res = await fetch(
        `/admin/api/ext/forms/${formId}/notifications/${index}/test`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ override_recipient: testRecipient.trim() }),
          credentials: "include",
        },
      );
      const data = await res.json();
      if (res.ok) {
        toast.success(data.message || "Test email sent");
        setTestPopoverOpen(false);
      } else {
        toast.error(data.error || "Failed to send test email");
      }
    } catch {
      toast.error("An error occurred while sending the test email");
    } finally {
      setTestSending(false);
    }
  };

  const notifType = notif.type || "admin";
  const isAutoResponder = notifType === "auto-responder";

  const headerActions = (
    <div className="flex items-center gap-1">
      <Switch
        checked={notif.enabled !== false}
        onCheckedChange={(checked: boolean) => onUpdate(index, "enabled", checked)}
        className="scale-75"
      />
      {Popover ? (
        <Popover open={testPopoverOpen} onOpenChange={setTestPopoverOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-slate-400 hover:text-indigo-500"
              title="Send test email"
            >
              <Send className="h-3.5 w-3.5" />
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-72 p-3 space-y-3" align="end">
            <p className="text-sm font-medium text-slate-700">Send Test Email</p>
            <div className="space-y-1">
              <Label className="text-xs text-slate-500">
                Recipient (leave blank for your account email)
              </Label>
              <Input
                value={testRecipient}
                onChange={(e: any) => setTestRecipient(e.target.value)}
                placeholder="admin@example.com"
                className="h-8 text-sm"
              />
            </div>
            <Button className="w-full" size="sm" onClick={handleSendTest} disabled={testSending}>
              <Send className="mr-2 h-3.5 w-3.5" />
              {testSending ? "Sending…" : "Send Test"}
            </Button>
          </PopoverContent>
        </Popover>
      ) : (
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-slate-400 hover:text-indigo-500"
          title="Send test email"
          onClick={handleSendTest}
        >
          <Send className="h-3.5 w-3.5" />
        </Button>
      )}
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onRemove(index)}
        className="h-7 w-7 text-slate-400 hover:text-red-500"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );

  const notifLabel = (
    <div className="flex items-center gap-2 flex-1 min-w-0">
      {isAutoResponder ? (
        <User className="h-4 w-4 text-blue-400 shrink-0" />
      ) : (
        <Mail className="h-4 w-4 text-slate-400 shrink-0" />
      )}
      <span className="font-medium text-[13px] text-slate-700 truncate">
        {notif.name}
      </span>
      <span
        className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full shrink-0 ${
          isAutoResponder ? "bg-blue-100 text-blue-600" : "bg-slate-200 text-slate-500"
        }`}
      >
        {isAutoResponder ? "Auto-Responder" : "Admin"}
      </span>
      {notif.enabled === false && (
        <span className="text-[10px] font-medium px-1.5 py-0.5 rounded-full bg-slate-100 text-slate-400 shrink-0">
          Disabled
        </span>
      )}
    </div>
  );

  return (
    <Card
      className={`rounded-xl border border-slate-200 shadow-sm overflow-hidden ${
        !notif.enabled ? "opacity-60" : ""
      }`}
    >
      {/* Custom header to match SectionHeader look but with label + actions */}
      <div
        className="flex items-center justify-between px-4 py-3 cursor-pointer select-none"
        style={{
          background: "var(--sub-bg)",
          borderBottom: collapsed ? "none" : "1px solid var(--border)",
        }}
        onClick={() => setCollapsed(!collapsed)}
      >
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <ChevronDown
            className={`h-3.5 w-3.5 text-slate-400 transition-transform shrink-0 ${
              collapsed ? "-rotate-90" : ""
            }`}
          />
          {notifLabel}
        </div>
        <div onClick={(e) => e.stopPropagation()}>{headerActions}</div>
      </div>

      {!collapsed && (
      <CardContent className="p-4 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">Notification Name</Label>
            <Input
              value={notif.name}
              onChange={(e: any) => onUpdate(index, "name", e.target.value)}
              placeholder="e.g. Admin Email"
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">Notification Type</Label>
            <Select value={notifType} onValueChange={(val: string) => onUpdate(index, "type", val)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">
                  <span className="flex items-center gap-2">
                    <Users className="h-3.5 w-3.5" />
                    Admin Notification
                  </span>
                </SelectItem>
                <SelectItem value="auto-responder">
                  <span className="flex items-center gap-2">
                    <User className="h-3.5 w-3.5" />
                    Auto-Responder
                  </span>
                </SelectItem>
              </SelectContent>
            </Select>
            <p className="text-[10px] text-slate-400">
              {isAutoResponder
                ? "Sends to the submitter's email address."
                : "Sends to the configured recipient addresses."}
            </p>
          </div>
        </div>

        {isAutoResponder ? (
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">Recipient Field</Label>
            {emailFields.length > 0 ? (
              <Select
                value={notif.recipient_field || ""}
                onValueChange={(val: string) => onUpdate(index, "recipient_field", val)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select email field" />
                </SelectTrigger>
                <SelectContent>
                  {emailFields.map((field: any) => (
                    <SelectItem key={field.id} value={field.id}>
                      {field.label || field.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Input
                value={notif.recipient_field || ""}
                onChange={(e: any) => onUpdate(index, "recipient_field", e.target.value)}
                placeholder="Enter email field ID"
              />
            )}
            <p className="text-[10px] text-slate-400">
              Select which field contains the submitter's email address.
            </p>
          </div>
        ) : (
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-500">Recipient(s)</Label>
            <Input
              value={notif.recipients}
              onChange={(e: any) => onUpdate(index, "recipients", e.target.value)}
              placeholder="admin@example.com"
            />
            <p className="text-[10px] text-slate-400">
              Comma separated. Supports templates like <code>{"{{.SiteEmail}}"}</code>
            </p>
          </div>
        )}

        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Subject</Label>
          <Input
            value={notif.subject}
            onChange={(e: any) => onUpdate(index, "subject", e.target.value)}
            placeholder="New Message"
          />
        </div>

        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Email Body (Plain Text or HTML)</Label>
          <Textarea
            value={notif.body}
            onChange={(e: any) => onUpdate(index, "body", e.target.value)}
            className="min-h-[150px] font-mono text-sm"
            placeholder="Hello,\n\nYou have a new submission..."
          />
        </div>

        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-500">Reply-To Address</Label>
          {emailFields.length > 0 ? (
            <Select
              value={notif.reply_to || ""}
              onValueChange={(val: string) => onUpdate(index, "reply_to", val)}
            >
              <SelectTrigger>
                <SelectValue placeholder="None" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">None</SelectItem>
                {emailFields.map((field: any) => (
                  <SelectItem key={field.id} value={field.id}>
                    {field.label || field.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              value={notif.reply_to || ""}
              onChange={(e: any) => onUpdate(index, "reply_to", e.target.value)}
              placeholder="[email_field_id]"
            />
          )}
        </div>

        {/* CC / BCC */}
        <div className="border-t border-slate-100 pt-3">
          <AccordionRow
            open={expandedCC}
            onToggle={() => setExpandedCC(!expandedCC)}
            headerLeft={
              <span className="text-xs font-medium" style={{ color: "var(--fg)" }}>CC / BCC</span>
            }
          >
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">CC</Label>
                <Input
                  value={notif.cc || ""}
                  onChange={(e: any) => onUpdate(index, "cc", e.target.value)}
                  placeholder="cc@example.com"
                />
                <p className="text-[10px] text-slate-400">Comma separated email addresses.</p>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-500">BCC</Label>
                <Input
                  value={notif.bcc || ""}
                  onChange={(e: any) => onUpdate(index, "bcc", e.target.value)}
                  placeholder="bcc@example.com"
                />
                <p className="text-[10px] text-slate-400">Comma separated email addresses.</p>
              </div>
            </div>
          </AccordionRow>
        </div>

        <NotificationDisplayWhen
          routeWhen={notif.route_when || {}}
          onChange={(next) => onUpdate(index, "route_when", next)}
          formFields={formFields}
        />
      </CardContent>
      )}
    </Card>
  );
}
