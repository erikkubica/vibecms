import { useNavigate } from "react-router-dom";
import { iconMap } from "./sdui-components";

// ---------------------------------------------------------------------------
// ContentTypeCard — card displaying a content type with metadata and actions
// ---------------------------------------------------------------------------

export function ContentTypeCard({
  id: _id,
  slug,
  label,
  labelPlural,
  icon,
  description,
  supportsBlocks,
  taxonomyCount,
  editPath,
  onEdit,
  onDelete,
}: {
  id: number;
  slug: string;
  label: string;
  labelPlural?: string;
  icon?: string;
  description?: string;
  supportsBlocks?: boolean;
  taxonomyCount?: number;
  editPath?: string;
  onEdit?: () => void;
  onDelete?: () => void;
}) {
  const navigate = useNavigate();
  const IconComp = icon ? iconMap[icon] : null;
  const displayName = labelPlural ? labelPlural : label;

  const handleEdit = () => {
    if (onEdit) {
      onEdit();
    } else if (editPath) {
      navigate(editPath);
    }
  };

  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-sm transition-colors hover:border-slate-300">
      <div className="p-6">
        {/* Header row: icon + label + slug badge */}
        <div className="flex items-center gap-3">
          {IconComp && (
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-slate-100">
              <IconComp className="h-5 w-5 text-slate-600" />
            </div>
          )}
          <div className="min-w-0 flex-1">
            <h3 className="text-base font-semibold text-slate-900">
              {displayName}
            </h3>
          </div>
          <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-500">
            {slug}
          </span>
        </div>

        {/* Description */}
        {description && (
          <p className="mt-3 text-sm text-slate-500">{description}</p>
        )}

        {/* Footer badges */}
        {(supportsBlocks ||
          (taxonomyCount !== undefined && taxonomyCount > 0)) && (
          <div className="mt-4 flex flex-wrap items-center gap-2">
            {supportsBlocks && (
              <span className="inline-flex items-center rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700">
                Supports blocks
              </span>
            )}
            {taxonomyCount !== undefined && taxonomyCount > 0 && (
              <span className="inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
                {taxonomyCount}{" "}
                {taxonomyCount === 1 ? "taxonomy" : "taxonomies"}
              </span>
            )}
          </div>
        )}

        {/* Action buttons */}
        <div className="mt-4 flex items-center gap-2 border-t border-slate-100 pt-4">
          <button
            onClick={handleEdit}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50"
          >
            Edit
          </button>
          <button
            onClick={onDelete}
            className="inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 hover:text-red-700"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// TaxonomyCard — card displaying a taxonomy with metadata and actions
// ---------------------------------------------------------------------------

export function TaxonomyCard({
  id: _id,
  slug,
  label,
  labelPlural,
  description,
  hierarchical,
  nodeTypes,
  editPath,
  onEdit,
  onDelete,
}: {
  id: number;
  slug: string;
  label: string;
  labelPlural?: string;
  description?: string;
  hierarchical?: boolean;
  nodeTypes?: string[];
  editPath?: string;
  onEdit?: () => void;
  onDelete?: () => void;
}) {
  const navigate = useNavigate();
  const displayName = labelPlural ? labelPlural : label;

  const handleEdit = () => {
    if (onEdit) {
      onEdit();
    } else if (editPath) {
      navigate(editPath);
    }
  };

  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-sm transition-colors hover:border-slate-300">
      <div className="p-6">
        {/* Header row: label + slug badge */}
        <div className="flex items-center gap-3">
          <div className="min-w-0 flex-1">
            <h3 className="text-base font-semibold text-slate-900">
              {displayName}
            </h3>
          </div>
          <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono text-slate-500">
            {slug}
          </span>
        </div>

        {/* Description */}
        {description && (
          <p className="mt-3 text-sm text-slate-500">{description}</p>
        )}

        {/* Footer badges: hierarchical + node types */}
        {(hierarchical || (nodeTypes && nodeTypes.length > 0)) && (
          <div className="mt-4 flex flex-wrap items-center gap-2">
            {hierarchical && (
              <span className="inline-flex items-center rounded-full bg-purple-50 px-2.5 py-0.5 text-xs font-medium text-purple-700">
                Hierarchical
              </span>
            )}
            {nodeTypes &&
              nodeTypes.map((type) => (
                <span
                  key={type}
                  className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600"
                >
                  {type}
                </span>
              ))}
          </div>
        )}

        {/* Action buttons */}
        <div className="mt-4 flex items-center gap-2 border-t border-slate-100 pt-4">
          <button
            onClick={handleEdit}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50"
          >
            Edit
          </button>
          <button
            onClick={onDelete}
            className="inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 hover:text-red-700"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
}
