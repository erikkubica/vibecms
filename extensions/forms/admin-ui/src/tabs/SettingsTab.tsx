import React from "react";
import { CheckCircle2, XCircle, MousePointer2, Globe } from "lucide-react";

const { Card, CardContent, Input, Label } = (window as any).__VIBECMS_SHARED__.ui;

export default function SettingsTab({ form, setForm }: any) {
    const updateSettings = (key: string, value: any) => {
        setForm({
            ...form,
            settings: { ...form.settings, [key]: value }
        });
    };

    return (
        <div className="space-y-6 max-w-2xl">
            <Card className="border-slate-200 shadow-none">
                <CardContent className="p-4 space-y-6">
                    <div className="space-y-4">
                        <h3 className="text-lg font-semibold flex items-center gap-2">
                            <CheckCircle2 className="h-5 w-5 text-green-500" />
                            Success & Error Messages
                        </h3>
                        <div className="space-y-2">
                            <Label>Success Message</Label>
                            <Input 
                                value={form.settings.success_message} 
                                onChange={(e: any) => updateSettings("success_message", e.target.value)}
                                placeholder="Thank you for your submission!"
                            />
                        </div>
                        <div className="space-y-2">
                            <Label>Error Message</Label>
                            <Input 
                                value={form.settings.error_message} 
                                onChange={(e: any) => updateSettings("error_message", e.target.value)}
                                placeholder="There was an error processing your request."
                            />
                        </div>
                    </div>

                    <div className="space-y-4 pt-6 border-t border-slate-100">
                        <h3 className="text-lg font-semibold flex items-center gap-2">
                            <MousePointer2 className="h-5 w-5 text-indigo-500" />
                            Button Settings
                        </h3>
                        <div className="space-y-2">
                            <Label>Submit Button Text</Label>
                            <Input 
                                value={form.settings.submit_button_text} 
                                onChange={(e: any) => updateSettings("submit_button_text", e.target.value)}
                                placeholder="Submit"
                            />
                        </div>
                    </div>

                    <div className="space-y-4 pt-6 border-t border-slate-100">
                        <h3 className="text-lg font-semibold flex items-center gap-2">
                            <Globe className="h-5 w-5 text-blue-500" />
                            Redirects
                        </h3>
                        <div className="space-y-2">
                            <Label>Redirect URL (Optional)</Label>
                            <Input 
                                value={form.settings.redirect_url} 
                                onChange={(e: any) => updateSettings("redirect_url", e.target.value)}
                                placeholder="e.g. /thank-you"
                            />
                            <p className="text-xs text-slate-400 italic">Leave empty to stay on the same page and show success message.</p>
                        </div>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
