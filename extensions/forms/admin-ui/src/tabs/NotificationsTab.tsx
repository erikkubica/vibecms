import React from "react";
import { Plus, Trash2, Mail, Send, ChevronDown, ChevronUp } from "lucide-react";

const { Button, Card, CardContent, Input, Label, Textarea } = (window as any).__VIBECMS_SHARED__.ui;

export default function NotificationsTab({ form, setForm }: any) {
    const addNotification = () => {
        const newNotif = {
            name: "New Notification",
            enabled: true,
            recipients: "{{.SiteEmail}}",
            subject: "New submission: {{.FormName}}",
            body: "You have a new submission.\n\n{{range .Data}}\n{{.Label}}: {{.Value}}\n{{end}}",
            reply_to: ""
        };
        setForm({ ...form, notifications: [...form.notifications, newNotif] });
    };

    const removeNotification = (index: number) => {
        const newNotifs = [...form.notifications];
        newNotifs.splice(index, 1);
        setForm({ ...form, notifications: newNotifs });
    };

    const updateNotification = (index: number, key: string, value: any) => {
        const newNotifs = [...form.notifications];
        newNotifs[index] = { ...newNotifs[index], [key]: value };
        setForm({ ...form, notifications: newNotifs });
    };

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-slate-900">Email Notifications</h3>
                <Button variant="outline" size="sm" onClick={addNotification} className="border-indigo-200 text-indigo-600 hover:bg-indigo-50">
                    <Plus className="mr-2 h-4 w-4" /> Add Notification
                </Button>
            </div>

            <div className="space-y-4">
                {form.notifications.map((notif: any, index: number) => (
                    <Card key={index} className="border-slate-200 shadow-none overflow-hidden">
                        <div className="bg-slate-50 px-4 py-2 border-b border-slate-100 flex items-center justify-between">
                            <div className="flex items-center gap-2">
                                <Mail className="h-4 w-4 text-slate-400" />
                                <span className="font-medium text-slate-700">{notif.name}</span>
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
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <Label>Notification Name</Label>
                                    <Input 
                                        value={notif.name} 
                                        onChange={(e: any) => updateNotification(index, "name", e.target.value)}
                                        placeholder="e.g. Admin Email"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label>Recipient(s)</Label>
                                    <Input 
                                        value={notif.recipients} 
                                        onChange={(e: any) => updateNotification(index, "recipients", e.target.value)}
                                        placeholder="admin@example.com"
                                    />
                                    <p className="text-[10px] text-slate-400">Comma separated. Supports templates like <code>{"{{.SiteEmail}}"}</code></p>
                                </div>
                            </div>
                            <div className="space-y-2">
                                <Label>Subject</Label>
                                <Input 
                                    value={notif.subject} 
                                    onChange={(e: any) => updateNotification(index, "subject", e.target.value)}
                                    placeholder="New Message"
                                />
                            </div>
                            <div className="space-y-2">
                                <Label>Email Body (Plain Text or HTML)</Label>
                                <Textarea 
                                    value={notif.body} 
                                    onChange={(e: any) => updateNotification(index, "body", e.target.value)}
                                    className="min-h-[150px] font-mono text-sm"
                                    placeholder="Hello,\n\nYou have a new submission..."
                                />
                            </div>
                            <div className="space-y-2">
                                <Label>Reply-To Address</Label>
                                <Input 
                                    value={notif.reply_to} 
                                    onChange={(e: any) => updateNotification(index, "reply_to", e.target.value)}
                                    placeholder="[email_field_id]"
                                />
                                <p className="text-[10px] text-slate-400">Enter a field ID from the builder to use as reply-to address.</p>
                            </div>
                        </CardContent>
                    </Card>
                ))}

                {form.notifications.length === 0 && (
                    <div className="p-12 text-center border-2 border-dashed border-slate-200 rounded-xl text-slate-400">
                        No notifications configured. You won't be alerted of new submissions.
                    </div>
                )}
            </div>
        </div>
    );
}
