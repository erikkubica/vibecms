import { OptionPair } from "./types";

export function keyify(str: string): string {
  return str
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .replace(/_+/g, "_");
}

export function normalizeOptions(options: any): OptionPair[] {
  if (!options) return [];
  if (!Array.isArray(options)) return [];
  if (options.length === 0) return [];
  if (
    typeof options[0] === "object" &&
    options[0] !== null &&
    "label" in options[0]
  ) {
    return options;
  }
  return options
    .filter((o: any) => typeof o === "string" && o.trim())
    .map((o: string) => ({ label: o.trim(), value: o.trim() }));
}
