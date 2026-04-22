import React, { useState } from "react";
import {
  Plus,
  Trash2,
  Mail,
  Send,
  ChevronDown,
  ChevronUp,
  Code,
  User,
  Users,
  Eye,
  EyeOff,
} from "@vibecms/icons";

const {
  Button,
  Card,
  CardContent,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
  Textarea,
} = (window as any).__VIBECMS_SHARED__.ui;

const TEMPLATE_VARS = [
  { syntax: "{{.FormName}}", desc: "Form display name" },
  { syntax: "{{.FormSlug}}", desc: "Form URL slug" },
  { syntax: "{{.FormID}}", desc: "Form database ID" },
  { syntax: "{{.SubmittedAt}}", desc: "Submission timestamp" },
  {
    syntax: "{{range .Data}}",
    desc: "Loop all submitted fields",
    children: [
      { syntax: "  {{.Label}}", desc: "Field label" },
      { syntax: "  {{.Value}}", desc: "Submitted value" },
      { syntax: "  {{.Key}}", desc: "Field key" },
      { syntax: "{{end}}", desc: "End loop" },
    ],
  },
  {
    syntax: "{{.Field.email}}",
    desc: "Direct access to specific field value",
  },
  {
    syntax: "{{.Field.name}}",
    desc: "Replace email/name with your field keys",
  },
];

function getEmailFields(fields: any[]): any[] {
  return (fields || []).filter((f: any) => f.type === "email");
}

export default function NotificationsTab({ form, setForm }: any) {
  const [varsExpanded, setVarsExpanded] = useState(false);
  const [expandedCC, setExpandedCC] = useState<Record<number, boolean>>({});

  const emailFields = getEmailFields(form.fields || []);

  const addNotification = () => {
    const newNotif = {
      name: "New Notification",
      type: "admin",
      enabled: true,
      recipients: "{{.SiteEmail}}",
      recipient_field: "",
      subject: "New submission: {{.FormName}}",
      body: "You have a new submission.\n\n{{range .Data}}\n{{.Label}}: {{.Value}}\n{{end}}",
      reply_to: "",
      cc: "",
      bcc: "",
    };
    setForm({
      ...form,
      notifications: [...(form.notifications || []), newNotif],
    });
  };

  const removeNotification = (index: number) => {
    const newNotifs = [...form.notifications];
    newNotifs.splice(index, 1);
    setForm({ ...form, notifications: newNotifs });
  };

  const updateNotification = (index: number, key: string, value: any) => {
    const newNotifs = [...form.notifications];
    newNotifs[index] = { ...newNotifs[index], [key]: value };

    // When switching to auto-responder, set recipients from recipient_field
    if (key === "type" && value === "auto-responder") {
      const field = emailFields.length > 0 ? emailFields[0].id : "";
      newNotifs[index].recipient_field = field;
      if (field) {
        newNotifs[index].recipients = `{{.Field.${field}}}`;
      }
    }

    // When switching back to admin, clear recipient_field
    if (key === "type" && value === "admin") {
      newNotifs[index].recipient_field = "";
      if (
        !newNotifs[index].recipients ||
        newNotifs[index].recipients.startsWith("{{.Field.")
      ) {
        newNotifs[index].recipients = "{{.SiteEmail}}";
      }
    }

    // Update recipients template when recipient_field changes
    if (key === "recipient_field") {
      if (value) {
        newNotifs[index].recipients = `{{.Field.${value}}}`;
      }
    }

    setForm({ ...form, notifications: newNotifs });
  };

  const getNotifType = (notif: any): string => {
    return notif.type || "admin";
  };

  const toggleCC = (index: number) => {
    setExpandedCC((prev) => ({ ...prev, [index]: !prev[index] }));
  };

  return (
    <div className="space-y-6">
      {/* Template Variables Reference */}
      <Card className="border-slate-200 shadow-none">
        <button
          type="button"
          className="w-full px-4 py-3 flex items-center justify-between hover:bg-slate-50 transition-colors"
          onClick={() => setVarsExpanded(!varsExpanded)}
        >
          <div className="flex items-center gap-2">
            <Code className="h-4 w-4 text-indigo-500" />
            <span className="text-sm font-semibold text-slate-700">
              Template Variables Reference
            </span>
          </div>
          {varsExpanded ? (
            <ChevronUp className="h-4 w-4 text-slate-400" />
          ) : (
            <ChevronDown className="h-4 w-4 text-slate-400" />
          )}
        </button>
        {varsExpanded && (
          <CardContent className="px-4 pb-4 pt-0">
            <div className="bg-slate-50 rounded-lg p-4 font-mono text-xs leading-relaxed text-slate-700 border border-slate-100">
              <p className="text-slate-500 text-[11px] mb-2 font-sans font-medium">
                Available in Subject and Body:
              </p>
              {TEMPLATE_VARS.map((v, i) => (
                <React.Fragment key={i}>
                  <div className="flex gap-3 py-0.5">
                    <span className="text-indigo-600 whitespace-nowrap min-w-[180px]">
                      {v.syntax}
                    </span>
                    <span className="text-slate-500 font-sans">— {v.desc}</span>
                  </div>
                  {v.children &&
                    v.children.map((child, j) => (
                      <div key={`${i}-${j}`} className="flex gap-3 py-0.5">
                        <span className="text-indigo-600 whitespace-nowrap min-w-[180px]">
                          {child.syntax}
                        </span>
                        <span className="text-slate-500 font-sans">
                          — {child.desc}
                        </span>
                      </div>
                    ))}
                </React.Fragment>
              ))}
            </div>
          </CardContent>
        )}
      </Card>

      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-900">
          Email Notifications
        </h3>
        <Button
          variant="outline"
          size="sm"
          onClick={addNotification}
          className="border-indigo-200 text-indigo-600 hover:bg-indigo-50"
        >
          <Plus className="mr-2 h-4 w-4" /> Add Notification
        </Button>
      </div>

      {/* Notifications List */}
      <div className="space-y-4">
        {(form.notifications || []).map((notif: any, index: number) => {
          const notifType = getNotifType(notif);
          const isAutoResponder = notifType === "auto-responder";

          return (
            <Card
              key={index}
              className={`border-slate-200 shadow-none overflow-hidden ${
                !notif.enabled ? "opacity-60" : ""
              }`}
            >
              {/* Card Header */}
              <div className="bg-slate-50 px-4 py-2 border-b border-slate-100 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <Switch
                    checked={notif.enabled !== false}
                    onCheckedChange={(checked: boolean) =>
                      updateNotification(index, "enabled", checked)
                    }
                    className="scale-75"
                  />
                  <div className="flex items-center gap-2">
                    {isAutoResponder ? (
                      <User className="h-4 w-4 text-blue-400" />
                    ) : (
                      <Mail className="h-4 w-4 text-slate-400" />
                    )}
                    <span className="font-medium text-slate-700">
                      {notif.name}
                    </span>
                    <span
                      className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${
                        isAutoResponder
                          ? "bg-blue-100 text-blue-600"
                          : "bg-slate-200 text-slate-500"
                      }`}
                    >
                      {isAutoResponder ? "Auto-Responder" : "Admin"}
                    </span>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => removeNotification(index)}
                  className="h-7 w-7 text-slate-400 hover:text-red-500"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>

              <CardContent className="p-4 space-y-4">
                {/* Row 1: Name + Type */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Notification Name</Label>
                    <Input
                      value={notif.name}
                      onChange={(e: any) =>
                        updateNotification(index, "name", e.target.value)
                      }
                      placeholder="e.g. Admin Email"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Notification Type</Label>
                    <Select
                      value={notifType}
                      onValueChange={(val: string) =>
                        updateNotification(index, "type", val)
                      }
                    >
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

                {/* Row 2: Recipients or Recipient Field */}
                {isAutoResponder ? (
                  <div className="space-y-2">
                    <Label>Recipient Field</Label>
                    {emailFields.length > 0 ? (
                      <Select
                        value={notif.recipient_field || ""}
                        onValueChange={(val: string) =>
                          updateNotification(index, "recipient_field", val)
                        }
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
                        onChange={(e: any) =>
                          updateNotification(
                            index,
                            "recipient_field",
                            e.target.value,
                          )
                        }
                        placeholder="Enter email field ID"
                      />
                    )}
                    <p className="text-[10px] text-slate-400">
                      Select which field contains the submitter's email address.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <Label>Recipient(s)</Label>
                    <Input
                      value={notif.recipients}
                      onChange={(e: any) =>
                        updateNotification(index, "recipients", e.target.value)
                      }
                      placeholder="admin@example.com"
                    />
                    <p className="text-[10px] text-slate-400">
                      Comma separated. Supports templates like{" "}
                      <code>{"{{.SiteEmail}}"}</code>
                    </p>
                  </div>
                )}

                {/* Row 3: Subject */}
                <div className="space-y-2">
                  <Label>Subject</Label>
                  <Input
                    value={notif.subject}
                    onChange={(e: any) =>
                      updateNotification(index, "subject", e.target.value)
                    }
                    placeholder="New Message"
                  />
                </div>

                {/* Row 4: Body */}
                <div className="space-y-2">
                  <Label>Email Body (Plain Text or HTML)</Label>
                  <Textarea
                    value={notif.body}
                    onChange={(e: any) =>
                      updateNotification(index, "body", e.target.value)
                    }
                    className="min-h-[150px] font-mono text-sm"
                    placeholder="Hello,\n\nYou have a new submission..."
                  />
                </div>

                {/* Row 5: Reply-To */}
                <div className="space-y-2">
                  <Label>Reply-To Address</Label>
                  {emailFields.length > 0 ? (
                    <Select
                      value={notif.reply_to || ""}
                      onValueChange={(val: string) =>
                        updateNotification(index, "reply_to", val)
                      }
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
                      onChange={(e: any) =>
                        updateNotification(index, "reply_to", e.target.value)
                      }
                      placeholder="[email_field_id]"
                    />
                  )}
                  <p className="text-[10px] text-slate-400">
                    {emailFields.length > 0
                      ? "Select an email field to use as the reply-to address."
                      : "Enter a field ID from the builder to use as reply-to address."}
                  </p>
                </div>

                {/* CC / BCC */}
                <div className="border-t border-slate-100 pt-3">
                  <button
                    type="button"
                    className="flex items-center gap-1.5 text-xs text-slate-500 hover:text-slate-700 transition-colors"
                    onClick={() => toggleCC(index)}
                  >
                    {expandedCC[index] ? (
                      <ChevronUp className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronDown className="h-3.5 w-3.5" />
                    )}
                    <span className="font-medium">CC / BCC</span>
                  </button>
                  {expandedCC[index] && (
                    <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <Label>CC</Label>
                        <Input
                          value={notif.cc || ""}
                          onChange={(e: any) =>
                            updateNotification(index, "cc", e.target.value)
                          }
                          placeholder="cc@example.com"
                        />
                        <p className="text-[10px] text-slate-400">
                          Comma separated email addresses.
                        </p>
                      </div>
                      <div className="space-y-2">
                        <Label>BCC</Label>
                        <Input
                          value={notif.bcc || ""}
                          onChange={(e: any) =>
                            updateNotification(index, "bcc", e.target.value)
                          }
                          placeholder="bcc@example.com"
                        />
                        <p className="text-[10px] text-slate-400">
                          Comma separated email addresses.
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          );
        })}

        {(form.notifications || []).length === 0 && (
          <div className="p-12 text-center border-2 border-dashed border-slate-200 rounded-xl text-slate-400">
            No notifications configured. You won't be alerted of new
            submissions.
          </div>
        )}
      </div>
    </div>
  );
}
