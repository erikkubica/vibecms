export interface OptionPair {
  label: string;
  value: string;
}

export type FieldTypeMeta = {
  value: string;
  label: string;
  hasOptions?: boolean;
  hasFileOpts?: boolean;
  hasNumericOpts?: boolean;
  hasLengthOpts?: boolean;
  hasPattern?: boolean;
  hasPlaceholder?: boolean;
  hasGDPR?: boolean;
};

export const FIELD_TYPES: FieldTypeMeta[] = [
  {
    value: "text",
    label: "Text",
    hasLengthOpts: true,
    hasPattern: true,
    hasPlaceholder: true,
  },
  {
    value: "email",
    label: "Email",
    hasLengthOpts: true,
    hasPattern: true,
    hasPlaceholder: true,
  },
  { value: "tel", label: "Phone", hasLengthOpts: true, hasPlaceholder: true },
  {
    value: "url",
    label: "URL",
    hasLengthOpts: true,
    hasPattern: true,
    hasPlaceholder: true,
  },
  {
    value: "number",
    label: "Number",
    hasNumericOpts: true,
    hasPlaceholder: true,
  },
  {
    value: "range",
    label: "Range",
    hasNumericOpts: true,
    hasPlaceholder: true,
  },
  {
    value: "textarea",
    label: "Textarea",
    hasLengthOpts: true,
    hasPlaceholder: true,
  },
  { value: "select", label: "Select", hasOptions: true, hasPlaceholder: true },
  { value: "checkbox", label: "Checkbox" },
  { value: "radio", label: "Radio", hasOptions: true },
  { value: "date", label: "Date" },
  { value: "file", label: "File Upload", hasFileOpts: true },
  { value: "hidden", label: "Hidden" },
  { value: "gdpr_consent", label: "GDPR Consent", hasGDPR: true },
];

export const typeLabelMap: Record<string, string> = {};
FIELD_TYPES.forEach((t) => {
  typeLabelMap[t.value] = t.label;
});
