import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { FileText, Eye, PenLine, Users, Plus, Loader2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAuth } from "@/hooks/use-auth";
import { getNodes, getUsers, type ContentNode } from "@/api/client";

interface Stats {
  totalPages: number;
  published: number;
  drafts: number;
  totalUsers: number;
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case "published":
      return "bg-emerald-100 text-emerald-700 hover:bg-emerald-100";
    case "draft":
      return "bg-amber-100 text-amber-700 hover:bg-amber-100";
    case "archived":
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
    default:
      return "bg-slate-100 text-slate-600 hover:bg-slate-100";
  }
}

export default function DashboardPage() {
  const { user } = useAuth();
  const [stats, setStats] = useState<Stats | null>(null);
  const [recentNodes, setRecentNodes] = useState<ContentNode[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchData() {
      try {
        const [allNodes, publishedNodes, draftNodes, usersRes] =
          await Promise.all([
            getNodes({ page: 1, per_page: 1 }),
            getNodes({ page: 1, per_page: 1, status: "published" }),
            getNodes({ page: 1, per_page: 1, status: "draft" }),
            getUsers({ page: 1, per_page: 1 }).catch(() => ({
              data: [],
              meta: { total: 0, page: 1, per_page: 1, total_pages: 0 },
            })),
          ]);

        setStats({
          totalPages: allNodes.meta.total,
          published: publishedNodes.meta.total,
          drafts: draftNodes.meta.total,
          totalUsers: usersRes.meta.total,
        });

        const recent = await getNodes({ page: 1, per_page: 5 });
        setRecentNodes(recent.data);
      } catch {
        // Stats will remain null; UI handles gracefully
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  const statCards = [
    {
      label: "Total Pages",
      value: stats?.totalPages ?? 0,
      icon: FileText,
      color: "text-primary-600",
      bg: "bg-primary-50",
    },
    {
      label: "Published",
      value: stats?.published ?? 0,
      icon: Eye,
      color: "text-emerald-600",
      bg: "bg-emerald-50",
    },
    {
      label: "Drafts",
      value: stats?.drafts ?? 0,
      icon: PenLine,
      color: "text-amber-600",
      bg: "bg-amber-50",
    },
    {
      label: "Total Users",
      value: stats?.totalUsers ?? 0,
      icon: Users,
      color: "text-violet-600",
      bg: "bg-violet-50",
    },
  ];

  return (
    <div className="space-y-6">
      {/* Welcome */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-800">
            Welcome back, {user?.full_name || "Admin"}
          </h1>
          <p className="text-sm text-slate-500">
            Here is what is happening with your site.
          </p>
        </div>
        <Button asChild className="bg-primary-600 hover:bg-primary-700">
          <Link to="/admin/pages/new">
            <Plus className="mr-2 h-4 w-4" />
            Create New Page
          </Link>
        </Button>
      </div>

      {/* Stat cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {statCards.map((s) => (
          <Card key={s.label}>
            <CardContent className="flex items-center gap-4 p-6">
              <div
                className={`flex h-12 w-12 items-center justify-center rounded-lg ${s.bg}`}
              >
                <s.icon className={`h-6 w-6 ${s.color}`} />
              </div>
              <div>
                <p className="text-sm text-slate-500">{s.label}</p>
                <p className="text-2xl font-bold text-slate-800">{s.value}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Recent content */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Recent Content</CardTitle>
        </CardHeader>
        <CardContent>
          {recentNodes.length === 0 ? (
            <p className="py-8 text-center text-sm text-slate-500">
              No content yet. Create your first page to get started.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Title</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {recentNodes.map((node) => (
                  <TableRow key={node.id}>
                    <TableCell>
                      <Link
                        to={`/admin/${node.node_type === "post" ? "posts" : "pages"}/${node.id}/edit`}
                        className="font-medium text-primary-600 hover:underline"
                      >
                        {node.title}
                      </Link>
                    </TableCell>
                    <TableCell className="capitalize text-slate-500">
                      {node.node_type}
                    </TableCell>
                    <TableCell>
                      <Badge className={statusBadgeClass(node.status)}>
                        {node.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-slate-500">
                      {new Date(node.updated_at).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
