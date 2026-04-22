import React, { useEffect, useState } from "react";
import { Search, Eye, Download, Calendar, ArrowLeft, Filter } from "lucide-react";

const { Button, Card, CardContent, Table, TableBody, TableCell, TableHead, TableHeader, TableRow, Input, Badge, Dialog, DialogContent, DialogHeader, DialogTitle } = (window as any).__VIBECMS_SHARED__.ui;
const { useSearchParams, useNavigate } = (window as any).__VIBECMS_SHARED__.ReactRouterDOM;
const { toast } = (window as any).__VIBECMS_SHARED__.Sonner;

interface Submission {
    id: number;
    form_id: number;
    data: Record<string, any>;
    metadata: Record<string, any>;
    created_at: string;
    form_name?: string;
}

export default function SubmissionsList() {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const formId = searchParams.get("form_id");
    
    const [submissions, setSubmissions] = useState<Submission[]>([]);
    const [loading, setLoading] = useState(true);
    const [selectedSubmission, setSelectedSubmission] = useState<Submission | null>(null);
    const [search, setSearch] = useState("");

    const fetchSubmissions = async () => {
        try {
            const url = formId 
                ? `/admin/api/ext/forms/submissions?form_id=${formId}` 
                : "/admin/api/ext/forms/submissions";
            const res = await fetch(url, { credentials: "include" });
            const body = await res.json();
            setSubmissions(body.rows || []);
        } catch (err) {
            toast.error("Failed to load submissions");
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchSubmissions();
    }, [formId]);

    const filteredSubmissions = submissions.filter(s => {
        const dataStr = JSON.stringify(s.data).toLowerCase();
        return dataStr.includes(search.toLowerCase());
    });

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    {formId && (
                        <Button variant="ghost" size="icon" onClick={() => navigate("/admin/ext/forms")} className="rounded-full">
                            <ArrowLeft className="h-4 w-4" />
                        </Button>
                    )}
                    <div>
                        <h1 className="text-3xl font-bold tracking-tight text-slate-900">Submissions</h1>
                        <p className="text-slate-500">
                            {formId ? `Viewing entries for ${submissions[0]?.form_name || "Form #" + formId}` : "All form submissions across the site."}
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-3">
                    <Button variant="outline" size="sm">
                        <Download className="mr-2 h-4 w-4" /> Export CSV
                    </Button>
                </div>
            </div>

            <Card className="rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                <CardContent className="p-0">
                    <div className="flex items-center gap-4 p-4 border-b border-slate-100 bg-slate-50/50">
                        <div className="relative flex-1 max-w-sm">
                            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                            <Input 
                                placeholder="Search in submissions..." 
                                className="pl-9 bg-white border-slate-200 focus:border-indigo-500" 
                                value={search}
                                onChange={(e: any) => setSearch(e.target.value)}
                            />
                        </div>
                    </div>

                    <Table>
                        <TableHeader>
                            <TableRow className="hover:bg-transparent border-slate-100">
                                <TableHead className="text-slate-500 font-medium">Date</TableHead>
                                {!formId && <TableHead className="text-slate-500 font-medium">Form</TableHead>}
                                <TableHead className="text-slate-500 font-medium">Summary</TableHead>
                                <TableHead className="text-right text-slate-500 font-medium">Actions</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {loading ? (
                                <TableRow>
                                    <TableCell colSpan={formId ? 3 : 4} className="h-32 text-center text-slate-400">Loading entries...</TableCell>
                                </TableRow>
                            ) : filteredSubmissions.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={formId ? 3 : 4} className="h-32 text-center text-slate-400">No submissions found.</TableCell>
                                </TableRow>
                            ) : filteredSubmissions.map((sub) => (
                                <TableRow key={sub.id} className="border-slate-100 hover:bg-slate-50/50 transition-colors">
                                    <TableCell className="text-slate-600 text-sm whitespace-nowrap">
                                        <div className="flex items-center gap-2">
                                            <Calendar className="h-3 w-3 opacity-40" />
                                            {new Date(sub.created_at).toLocaleString()}
                                        </div>
                                    </TableCell>
                                    {!formId && (
                                        <TableCell>
                                            <Badge variant="outline" className="bg-slate-100 text-slate-600 border-slate-200">
                                                {sub.form_name || "Unknown"}
                                            </Badge>
                                        </TableCell>
                                    )}
                                    <TableCell className="max-w-[400px]">
                                        <div className="text-xs text-slate-500 truncate">
                                            {Object.entries(sub.data).slice(0, 3).map(([k, v]) => (
                                                <span key={k} className="mr-3">
                                                    <strong className="text-slate-700">{k}:</strong> {String(v)}
                                                </span>
                                            ))}
                                            {Object.keys(sub.data).length > 3 && "..."}
                                        </div>
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <Button 
                                            variant="ghost" 
                                            size="sm" 
                                            onClick={() => setSelectedSubmission(sub)}
                                            className="text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50"
                                        >
                                            <Eye className="mr-2 h-4 w-4" /> View Details
                                        </Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>

            <Dialog open={!!selectedSubmission} onOpenChange={(open) => !open && setSelectedSubmission(null)}>
                <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
                    <DialogHeader>
                        <DialogTitle className="flex items-center justify-between">
                            Submission Details
                            <Badge variant="outline" className="ml-4 font-normal text-xs">ID: #{selectedSubmission?.id}</Badge>
                        </DialogTitle>
                    </DialogHeader>
                    {selectedSubmission && (
                        <div className="space-y-6 py-4">
                            <div className="grid grid-cols-2 gap-4 p-4 bg-slate-50 rounded-lg border border-slate-100 text-sm">
                                <div>
                                    <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">Date Submitted</p>
                                    <p className="font-medium text-slate-900">{new Date(selectedSubmission.created_at).toLocaleString()}</p>
                                </div>
                                <div>
                                    <p className="text-slate-400 uppercase text-[10px] tracking-wider mb-1">Form Name</p>
                                    <p className="font-medium text-slate-900">{selectedSubmission.form_name || "N/A"}</p>
                                </div>
                            </div>

                            <div className="space-y-4">
                                <h4 className="font-semibold text-slate-900 flex items-center gap-2">
                                    <Filter className="h-4 w-4 text-indigo-500" />
                                    Submitted Data
                                </h4>
                                <div className="divide-y divide-slate-100 border border-slate-100 rounded-lg overflow-hidden">
                                    {Object.entries(selectedSubmission.data).map(([key, value]) => (
                                        <div key={key} className="grid grid-cols-3 p-3 text-sm">
                                            <div className="font-medium text-slate-600 capitalize">{key.replace(/_/g, ' ')}</div>
                                            <div className="col-span-2 text-slate-900 bg-white p-1 rounded min-h-[1.5rem]">
                                                {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>

                            {Object.keys(selectedSubmission.metadata || {}).length > 0 && (
                                <div className="space-y-2">
                                    <h4 className="font-semibold text-slate-900 text-sm">Technical Metadata</h4>
                                    <pre className="p-3 bg-slate-900 text-indigo-300 rounded-lg text-[11px] font-mono overflow-x-auto">
                                        {JSON.stringify(selectedSubmission.metadata, null, 2)}
                                    </pre>
                                </div>
                            )}
                        </div>
                    )}
                </DialogContent>
            </Dialog>
        </div>
    );
}
