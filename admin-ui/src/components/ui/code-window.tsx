import CodeEditor from "./code-editor";

interface CodeWindowProps {
  title: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
  height?: string;
  variables?: string[];
}

export function CodeWindow({ title, ...editorProps }: CodeWindowProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 shadow-sm">
      <div className="flex items-center justify-between bg-[#282c34] px-4 py-2">
        <span className="text-xs font-medium text-slate-400">{title}</span>
        <div className="flex gap-1.5">
          <div className="h-2.5 w-2.5 rounded-full bg-red-500/70" />
          <div className="h-2.5 w-2.5 rounded-full bg-amber-500/70" />
          <div className="h-2.5 w-2.5 rounded-full bg-emerald-500/70" />
        </div>
      </div>
      <CodeEditor {...editorProps} />
    </div>
  );
}
