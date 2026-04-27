import { UploadCloud } from "@vibecms/icons";

export default function DragOverlay({ active }: { active: boolean }) {
  if (!active) return null;
  return (
    <div className="fixed inset-0 z-[60] pointer-events-none">
      <div className="absolute inset-4 rounded-2xl border-2 border-dashed border-indigo-400 bg-indigo-500/10 backdrop-blur-sm grid place-items-center">
        <div className="text-center animate-pulse">
          <div className="mx-auto w-16 h-16 rounded-2xl bg-indigo-600 text-white grid place-items-center shadow-lg">
            <UploadCloud className="h-8 w-8" />
          </div>
          <div className="mt-3 text-[15px] font-semibold text-indigo-900">Drop anywhere to upload</div>
          <div className="text-[12px] text-indigo-700">We'll handle the rest — optimize, variants, CDN</div>
        </div>
      </div>
    </div>
  );
}
