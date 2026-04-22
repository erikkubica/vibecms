import React, { useEffect, useState } from "react";
import {
  Save,
  ArrowLeft,
  Layout,
  Settings,
  Mail,
  ListPlus,
  Eye,
} from "@vibecms/icons";

const {
  Button,
  Card,
  CardContent,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  Input,
  Label,
} = (window as any).__VIBECMS_SHARED__.ui;
const { useParams, useNavigate } = (window as any).__VIBECMS_SHARED__
  .ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

import BuilderTab from "./tabs/BuilderTab";
import LayoutTab from "./tabs/LayoutTab";
import PreviewTab from "./tabs/PreviewTab";
import NotificationsTab from "./tabs/NotificationsTab";
import SettingsTab from "./tabs/SettingsTab";

export default function FormEditor() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(id ? true : false);
  const [form, setForm] = useState({
    name: "",
    slug: "",
    fields: [],
    layout:
      '{{/* Template variables available:\n     .fields_list    — ordered array of all field objects (use {{range .fields_list}})\n     .fields         — map keyed by field ID:   .fields.email.label\n     .fields_by_id   — same map, explicit access: .fields_by_id.email.label\n     (shorthand)     — each field at top level by ID: .email.label, .email.value\n     .id             — form ID\n     .name           — form name\n\n     Each field has: .id, .label, .type, .placeholder, .required, .options (for selects)\n     Select options are objects: .label and .value\n*/}}\n<div class="max-w-2xl mx-auto">\n  <form class="space-y-5">\n    {{range .fields_list}}\n      <div class="space-y-1.5">\n        <label class="block text-sm font-semibold text-gray-700">{{.label}}{{if .required}} <span class="text-red-500">*</span>{{end}}</label>\n        {{if eq .type "textarea"}}\n          <textarea name="{{.id}}" rows="4" placeholder="{{.placeholder}}" class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-gray-900 shadow-sm placeholder:text-gray-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none transition-all duration-200" {{if .required}}required{{end}}></textarea>\n        {{else if eq .type "select"}}\n          <select name="{{.id}}" class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none transition-all duration-200" {{if .required}}required{{end}}>\n            <option value="" disabled selected>{{.placeholder}}</option>\n            {{range .options}}<option value="{{.value}}">{{.label}}</option>{{end}}\n          </select>\n        {{else}}\n          <input type="{{.type}}" name="{{.id}}" placeholder="{{.placeholder}}" class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-gray-900 shadow-sm placeholder:text-gray-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none transition-all duration-200" {{if .required}}required{{end}} />\n        {{end}}\n      </div>\n    {{end}}\n    {{/* Example: place a specific field by ID (shorthand or via .fields map)\n    <div class="grid grid-cols-2 gap-4">\n      <div>\n        <label class="block text-sm font-semibold text-gray-700">{{.first_name.label}}</label>\n        <input type="{{.first_name.type}}" name="{{.first_name.id}}" placeholder="{{.first_name.placeholder}}" class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none" />\n      </div>\n      <div>\n        <label class="block text-sm font-semibold text-gray-700">{{.fields.last_name.label}}</label>\n        <input type="{{.fields.last_name.type}}" name="{{.fields.last_name.id}}" class="w-full rounded-lg border border-gray-300 bg-white px-4 py-2.5 text-gray-900 shadow-sm focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none" />\n      </div>\n    </div>\n    */}}\n    <div class="pt-2">\n      <button type="submit" class="w-full sm:w-auto rounded-lg bg-indigo-600 px-8 py-3 text-sm font-semibold text-white shadow-md hover:bg-indigo-700 focus:ring-2 focus:ring-indigo-500/20 focus:outline-none active:scale-[0.98] transition-all duration-150 cursor-pointer">Send Message</button>\n    </div>\n  </form>\n</div>',
    notifications: [
      {
        name: "Admin Notification",
        enabled: true,
        recipients: "{{.SiteEmail}}",
        subject: "New submission: {{.FormName}}",
        body: "You have a new submission.\n\n{{range .Data}}\n{{.Label}}: {{.Value}}\n{{end}}",
        reply_to: "",
      },
    ],
    settings: {
      success_message: "Thank you! Your message has been sent.",
      error_message: "Oops! Something went wrong.",
      submit_button_text: "Send Message",
      redirect_url: "",
    },
  });

  useEffect(() => {
    if (id && id !== "new") {
      fetch(`/admin/api/ext/forms/${id}`, { credentials: "include" })
        .then((res) => res.json())
        .then((data) => {
          setForm({
            ...data,
            fields: data.fields || [],
            notifications: data.notifications || [],
            settings: data.settings || {},
          });
          setLoading(false);
        })
        .catch(() => {
          toast.error("Failed to load form");
          navigate("/admin/ext/forms");
        });
    }
  }, [id]);

  const handleSave = async () => {
    const method = id && id !== "new" ? "PUT" : "POST";
    const url =
      id && id !== "new"
        ? `/admin/api/ext/forms/${id}`
        : "/admin/api/ext/forms/";

    try {
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
        credentials: "include",
      });
      if (res.ok) {
        toast.success("Form saved successfully");
        if (method === "POST") {
          const data = await res.json();
          navigate(`/admin/ext/forms/edit/${data.id}`);
        }
      } else {
        const err = await res.json();
        toast.error(err.error || "Failed to save form");
      }
    } catch (err) {
      toast.error("An error occurred while saving");
    }
  };

  if (loading)
    return (
      <div className="p-8 text-center text-slate-500">
        Loading form editor...
      </div>
    );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/admin/ext/forms")}
            className="rounded-full"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-slate-900">
              {id && id !== "new" ? "Edit Form" : "Create New Form"}
            </h1>
            <p className="text-slate-500 text-sm">
              {form.name || "Untitled Form"}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            onClick={() => navigate("/admin/ext/forms")}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            className="bg-indigo-600 hover:bg-indigo-700"
          >
            <Save className="mr-2 h-4 w-4" /> Save Form
          </Button>
        </div>
      </div>

      <Tabs defaultValue="builder" className="w-full">
        <TabsList className="bg-slate-100 p-1 mb-6">
          <TabsTrigger
            value="builder"
            className="data-[state=active]:bg-white data-[state=active]:shadow-sm"
          >
            <ListPlus className="mr-2 h-4 w-4" /> Builder
          </TabsTrigger>
          <TabsTrigger
            value="layout"
            className="data-[state=active]:bg-white data-[state=active]:shadow-sm"
          >
            <Layout className="mr-2 h-4 w-4" /> Layout
          </TabsTrigger>
          <TabsTrigger
            value="preview"
            className="data-[state=active]:bg-white data-[state=active]:shadow-sm"
          >
            <Eye className="mr-2 h-4 w-4" /> Preview
          </TabsTrigger>
          <TabsTrigger
            value="notifications"
            className="data-[state=active]:bg-white data-[state=active]:shadow-sm"
          >
            <Mail className="mr-2 h-4 w-4" /> Notifications
          </TabsTrigger>
          <TabsTrigger
            value="settings"
            className="data-[state=active]:bg-white data-[state=active]:shadow-sm"
          >
            <Settings className="mr-2 h-4 w-4" /> Settings
          </TabsTrigger>
        </TabsList>

        <TabsContent value="builder">
          <BuilderTab form={form} setForm={setForm} />
        </TabsContent>
        <TabsContent value="layout">
          <LayoutTab form={form} setForm={setForm} />
        </TabsContent>
        <TabsContent value="preview">
          <PreviewTab form={form} />
        </TabsContent>
        <TabsContent value="notifications">
          <NotificationsTab form={form} setForm={setForm} />
        </TabsContent>
        <TabsContent value="settings">
          <SettingsTab form={form} setForm={setForm} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
