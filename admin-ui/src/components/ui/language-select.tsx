import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Language } from "@/api/client";

// LanguageSelect is the canonical language picker used everywhere in the
// admin (node editor, term editor, settings panels, translation pickers).
// Reasons to consolidate:
//   - Flags are intentionally omitted. They had inconsistent rendering
//     (some surfaces showed them, others didn't), they read as
//     country flags which conflict with locale ≠ country, and they
//     duplicated visual weight that the language name already carries.
//   - One component means visual changes (sizing, ordering, exclusion
//     rules) land everywhere with a single edit.
//
// Two modes share the same surface:
//   * <LanguageSelect value=... onChange=... /> — pick one language.
//   * <LanguageSelect mode="add" existing=[...] onAdd=... /> — choose a
//     language that isn't already present (translation creation).

interface BaseProps {
  languages: Language[];
  className?: string;
  triggerClassName?: string;
  disabled?: boolean;
  /** Optional ordering override. Defaults to language sort_order then name. */
  sort?: (a: Language, b: Language) => number;
  /** When true, only languages with is_active=true are listed. Default true. */
  activeOnly?: boolean;
}

interface PickProps extends BaseProps {
  mode?: "pick";
  value: string;
  onChange: (code: string) => void;
  placeholder?: string;
}

interface AddProps extends BaseProps {
  mode: "add";
  /** Language codes that are already present and should be hidden. */
  existing: string[];
  onAdd: (code: string) => void;
  placeholder?: string;
}

type Props = PickProps | AddProps;

export function LanguageSelect(props: Props): React.ReactElement | null {
  const { languages, className, triggerClassName, disabled, sort, activeOnly = true } = props;

  const visible = languages
    .filter((l) => (activeOnly ? l.is_active : true))
    .slice()
    .sort(sort || ((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name)));

  if (props.mode === "add") {
    const taken = new Set(props.existing);
    const remaining = visible.filter((l) => !taken.has(l.code));
    if (remaining.length === 0) return null;
    return (
      <Select
        value=""
        onValueChange={(v) => v && props.onAdd(v)}
        disabled={disabled}
      >
        <SelectTrigger className={triggerClassName ?? "h-9 rounded-lg border-slate-300 text-sm"}>
          <SelectValue placeholder={props.placeholder ?? "+ Add language"} />
        </SelectTrigger>
        <SelectContent className={className}>
          {remaining.map((lang) => (
            <SelectItem key={lang.code} value={lang.code}>
              {lang.name || lang.code}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }

  return (
    <Select value={props.value} onValueChange={props.onChange} disabled={disabled}>
      <SelectTrigger className={triggerClassName ?? "h-9 rounded-lg border-slate-300 text-sm"}>
        <SelectValue placeholder={props.placeholder ?? "Select a language"} />
      </SelectTrigger>
      <SelectContent className={className}>
        {visible.map((lang) => (
          <SelectItem key={lang.code} value={lang.code}>
            {lang.name || lang.code}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// LanguageLabel renders a language's display name from a code, falling back
// to the code itself when the language list hasn't loaded or the code is
// unknown. No flags. Used in pills, lists, and breadcrumbs that show 'the
// language this thing is in' without offering a picker.
export function LanguageLabel({
  languages,
  code,
}: {
  languages: Language[];
  code: string;
}): string {
  const found = languages.find((l) => l.code === code);
  return found?.name || code;
}
