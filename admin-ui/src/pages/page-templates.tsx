import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2, LayoutTemplate } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  listPageTemplates,
  getPageTemplate,
  type PageTemplate,
} from "@/api/client";

export default function PageTemplatesPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [templates, setTemplates] = useState<PageTemplate[]>([]);
  const [blockCounts, setBlockCounts] = useState<Record<string, { count: number; types: string[] }>>({});
  const [loadingDetails, setLoadingDetails] = useState<Set<string>>(new Set());

  useEffect(() => {
    listPageTemplates()
      .then((data) => {
        setTemplates(data);
        // Fetch details for each template to get block info
        data.forEach((tpl) => {
          setLoadingDetails((prev) => new Set(prev).add(tpl.slug));
          getPageTemplate(tpl.slug)
            .then((detail) => {
              const types = [...new Set(detail.blocks.map((b) => b.type))];
              setBlockCounts((prev) => ({
                ...prev,
                [tpl.slug]: { count: detail.blocks.length, types },
              }));
            })
            .catch(() => {})
            .finally(() => {
              setLoadingDetails((prev) => {
                const next = new Set(prev);
                next.delete(tpl.slug);
                return next;
              });
            });
        });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function useTemplate(slug: string) {
    navigate(`/admin/pages/new?template=${slug}`);
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Page Templates</h1>
          <p className="text-sm text-slate-500 mt-1">
            Pre-built page layouts you can use as a starting point for new pages.
          </p>
        </div>
      </div>

      {templates.length === 0 ? (
        <Card className="rounded-xl border border-slate-200 shadow-sm">
          <CardContent className="flex flex-col items-center justify-center py-16 text-slate-400">
            <LayoutTemplate className="h-12 w-12 mb-3" />
            <p className="text-lg font-medium">No page templates available</p>
            <p className="text-sm mt-1">Page templates will appear here when configured in your theme.</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {templates.map((tpl) => {
            const info = blockCounts[tpl.slug];
            const isLoadingDetail = loadingDetails.has(tpl.slug);

            return (
              <Card
                key={tpl.slug}
                className="rounded-xl border border-slate-200 shadow-sm hover:shadow-md transition-shadow overflow-hidden"
              >
                {/* Thumbnail */}
                {tpl.thumbnail ? (
                  <img
                    src={tpl.thumbnail}
                    alt={tpl.name}
                    className="w-full h-40 object-cover bg-slate-100"
                  />
                ) : (
                  <div className="w-full h-40 flex items-center justify-center bg-slate-100">
                    <LayoutTemplate className="h-12 w-12 text-slate-300" />
                  </div>
                )}

                <CardContent className="p-4 space-y-3">
                  <div>
                    <h3 className="text-base font-semibold text-slate-900">{tpl.name}</h3>
                    {tpl.description && (
                      <p className="text-sm text-slate-500 mt-1 line-clamp-2">{tpl.description}</p>
                    )}
                  </div>

                  {/* Block info */}
                  <div className="space-y-2">
                    {isLoadingDetail ? (
                      <div className="flex items-center gap-2 text-xs text-slate-400">
                        <Loader2 className="h-3 w-3 animate-spin" />
                        Loading details...
                      </div>
                    ) : info ? (
                      <>
                        <p className="text-xs text-slate-500">
                          {info.count} block{info.count !== 1 ? "s" : ""}
                        </p>
                        <div className="flex flex-wrap gap-1">
                          {info.types.map((type) => (
                            <Badge
                              key={type}
                              variant="secondary"
                              className="text-[10px] px-1.5 py-0 font-mono bg-slate-100 text-slate-500"
                            >
                              {type}
                            </Badge>
                          ))}
                        </div>
                      </>
                    ) : null}
                  </div>

                  <Button
                    className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg h-9 text-sm"
                    onClick={() => useTemplate(tpl.slug)}
                  >
                    Use Template
                  </Button>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
