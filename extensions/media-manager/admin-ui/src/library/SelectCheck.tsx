interface SelectCheckProps {
  checked: boolean;
  indeterminate?: boolean;
  onClick?: (e: React.MouseEvent) => void;
  size?: number;
  className?: string;
}

export default function SelectCheck({ checked, indeterminate, onClick, size = 18, className = "" }: SelectCheckProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-checked={checked}
      role="checkbox"
      className={`grid place-items-center rounded-md border transition-all cursor-pointer ${
        checked ? "bg-indigo-600 border-indigo-600" : "bg-white/95 border-slate-300 hover:border-indigo-500"
      } ${className}`}
      style={{ width: size, height: size }}
    >
      {checked && !indeterminate && (
        <svg width={size - 6} height={size - 6} viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M5 12l5 5L20 7" />
        </svg>
      )}
      {indeterminate && <div className="w-2 h-[2px] bg-white rounded" />}
    </button>
  );
}
