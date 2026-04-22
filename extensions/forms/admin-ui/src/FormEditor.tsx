import React, { useEffect, useState } from "react";
import { Save, ArrowLeft, Layout, Settings, Mail, ListPlus, Eye } from "lucide-react";

const { Button, Card, CardContent, Tabs, TabsList, TabsTrigger, TabsContent, Input, Label } = (window as any).__VIBECMS_SHARED__.ui;
const { useParams, useNavigate } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

import BuilderTab from "./tabs/BuilderTab";
import LayoutTab from "./tabs/LayoutTab";
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
        layout: "<!-- Default layout -->\n<div class=\"vibe-form\">\n  {{range .Fields}}\n    <div class=\"vibe-form-field\">\n      <label>{{.Label}}</label>\n      <input type=\"{{.Type}}\" name=\"{{.ID}}\" placeholder=\"{{.Placeholder}}\" {{if .Required}}required{{end}} />\n    </div>\n  {{end}}\n  <button type=\"submit\">Submit</button>\n</div>",
        notifications: [
            {
                name: "Admin Notification",
                enabled: true,
                recipients: "{{.SiteEmail}}",
                subject: "New submission: {{.FormName}}",
                body: "You have a new submission.\n\n{{range .Data}}\n{{.Label}}: {{.Value}}\n{{end}}",
                reply_to: ""
            }
        ],
        settings: {
            success_message: "Thank you! Your message has been sent.",
            error_message: "Oops! Something went wrong.",
            submit_button_text: "Send Message",
            redirect_url: ""
        }
    });

    useEffect(() => {
        if (id && id !== "new") {
            fetch(`/admin/api/ext/forms/${id}`, { credentials: "include" })
                .then(res => res.json())
                .then(data => {
                    setForm({
                        ...data,
                        fields: data.fields || [],
                        notifications: data.notifications || [],
                        settings: data.settings || {}
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
        const url = id && id !== "new" ? `/admin/api/ext/forms/${id}` : "/admin/api/ext/forms/";

        try {
            const res = await fetch(url, {
                method,
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(form),
                credentials: "include"
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

    if (loading) return <div className="p-8 text-center text-slate-500">Loading form editor...</div>;

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" onClick={() => navigate("/admin/ext/forms")} className="rounded-full">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900">{id && id !== "new" ? "Edit Form" : "Create New Form"}</h1>
                        <p className="text-slate-500 text-sm">{form.name || "Untitled Form"}</p>
                    </div>
                </div>
                <div className="flex items-center gap-3">
                    <Button variant="outline" onClick={() => navigate("/admin/ext/forms")}>Cancel</Button>
                    <Button onClick={handleSave} className="bg-indigo-600 hover:bg-indigo-700">
                        <Save className="mr-2 h-4 w-4" /> Save Form
                    </Button>
                </div>
            </div>

            <Tabs defaultValue="builder" className="w-full">
                <TabsList className="bg-slate-100 p-1 mb-6">
                    <TabsTrigger value="builder" className="data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        <ListPlus className="mr-2 h-4 w-4" /> Builder
                    </TabsTrigger>
                    <TabsTrigger value="layout" className="data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        <Layout className="mr-2 h-4 w-4" /> Layout
                    </TabsTrigger>
                    <TabsTrigger value="preview" className="data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        <Eye className="mr-2 h-4 w-4" /> Preview
                    </TabsTrigger>
                    <TabsTrigger value="notifications" className="data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        <Mail className="mr-2 h-4 w-4" /> Notifications
                    </TabsTrigger>
                    <TabsTrigger value="settings" className="data-[state=active]:bg-white data-[state=active]:shadow-sm">
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
