import React from "react";
import { Plus, Trash2, GripVertical } from "lucide-react";

const { Button, Card, CardContent, Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } = (window as any).__VIBECMS_SHARED__.ui;

export default function BuilderTab({ form, setForm }: any) {
    const addField = () => {
        const newField = {
            id: `field_${Date.now()}`,
            type: "text",
            label: "New Field",
            placeholder: "",
            required: false
        };
        setForm({ ...form, fields: [...form.fields, newField] });
    };

    const removeField = (index: number) => {
        const newFields = [...form.fields];
        newFields.splice(index, 1);
        setForm({ ...form, fields: newFields });
    };

    const updateField = (index: number, key: string, value: any) => {
        const newFields = [...form.fields];
        newFields[index] = { ...newFields[index], [key]: value };
        setForm({ ...form, fields: newFields });
    };

    return (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="md:col-span-2 space-y-4">
                {form.fields.map((field: any, index: number) => (
                    <Card key={field.id} className="border-slate-200 shadow-none hover:border-indigo-200 transition-colors">
                        <CardContent className="p-4 flex items-start gap-4">
                            <div className="mt-2 cursor-grab text-slate-300">
                                <GripVertical className="h-5 w-5" />
                            </div>
                            <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <Label className="text-xs text-slate-500 uppercase tracking-wider">Label</Label>
                                    <Input 
                                        value={field.label} 
                                        onChange={(e: any) => updateField(index, "label", e.target.value)}
                                        placeholder="e.g. Your Email"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label className="text-xs text-slate-500 uppercase tracking-wider">Type</Label>
                                    <Select 
                                        value={field.type} 
                                        onValueChange={(val: string) => updateField(index, "type", val)}
                                    >
                                        <SelectTrigger className="bg-white">
                                            <SelectValue />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="text">Text</SelectItem>
                                            <SelectItem value="email">Email</SelectItem>
                                            <SelectItem value="tel">Phone</SelectItem>
                                            <SelectItem value="textarea">Textarea</SelectItem>
                                            <SelectItem value="select">Select</SelectItem>
                                            <SelectItem value="checkbox">Checkbox</SelectItem>
                                            <SelectItem value="radio">Radio</SelectItem>
                                            <SelectItem value="date">Date</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>
                                <div className="space-y-2">
                                    <Label className="text-xs text-slate-500 uppercase tracking-wider">Field ID (Unique)</Label>
                                    <Input 
                                        value={field.id} 
                                        onChange={(e: any) => updateField(index, "id", e.target.value.replace(/\s+/g, '_').toLowerCase())}
                                        placeholder="e.g. user_email"
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label className="text-xs text-slate-500 uppercase tracking-wider">Placeholder</Label>
                                    <Input 
                                        value={field.placeholder} 
                                        onChange={(e: any) => updateField(index, "placeholder", e.target.value)}
                                        placeholder="Optional placeholder text"
                                    />
                                </div>
                            </div>
                            <Button 
                                variant="ghost" 
                                size="icon" 
                                onClick={() => removeField(index)}
                                className="text-slate-400 hover:text-red-500 hover:bg-red-50"
                            >
                                <Trash2 className="h-4 w-4" />
                            </Button>
                        </CardContent>
                    </Card>
                ))}

                <Button 
                    variant="outline" 
                    className="w-full border-dashed border-2 py-8 text-slate-500 hover:text-indigo-600 hover:border-indigo-300 hover:bg-indigo-50/50"
                    onClick={addField}
                >
                    <Plus className="mr-2 h-4 w-4" /> Add New Field
                </Button>
            </div>

            <div className="space-y-4">
                <Card className="border-slate-200 shadow-none sticky top-6">
                    <CardContent className="p-4 space-y-4">
                        <h3 className="font-semibold text-slate-900">Form Details</h3>
                        <div className="space-y-2">
                            <Label>Form Name</Label>
                            <Input 
                                value={form.name} 
                                onChange={(e: any) => setForm({ ...form, name: e.target.value })}
                                placeholder="Contact Us"
                            />
                        </div>
                        <div className="space-y-2">
                            <Label>Form Slug</Label>
                            <Input 
                                value={form.slug} 
                                onChange={(e: any) => setForm({ ...form, slug: e.target.value.replace(/\s+/g, '-').toLowerCase() })}
                                placeholder="contact-us"
                            />
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
