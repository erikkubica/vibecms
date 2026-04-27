import { describe, it, expect } from "vitest";
import { FIELD_TYPES, typeLabelMap, FieldTypeMeta } from "../types";

describe("FIELD_TYPES registry", () => {
  it("has at least one entry for every expected type", () => {
    const expectedValues = [
      "text", "email", "tel", "url", "number", "range",
      "textarea", "select", "checkbox", "radio", "date",
      "file", "hidden", "gdpr_consent",
    ];
    const registeredValues = FIELD_TYPES.map((t) => t.value);
    for (const v of expectedValues) {
      expect(registeredValues).toContain(v);
    }
  });

  it("every entry has a non-empty value and label", () => {
    for (const t of FIELD_TYPES) {
      expect(t.value).toBeTruthy();
      expect(t.label).toBeTruthy();
    }
  });

  it("no duplicate values", () => {
    const values = FIELD_TYPES.map((t) => t.value);
    expect(new Set(values).size).toBe(values.length);
  });

  it("types with options flag are select and radio only", () => {
    const withOptions = FIELD_TYPES.filter((t) => t.hasOptions).map((t) => t.value);
    expect(withOptions.sort()).toEqual(["radio", "select"]);
  });

  it("file type has hasFileOpts flag", () => {
    const file = FIELD_TYPES.find((t) => t.value === "file");
    expect(file?.hasFileOpts).toBe(true);
  });

  it("gdpr_consent has hasGDPR flag", () => {
    const gdpr = FIELD_TYPES.find((t) => t.value === "gdpr_consent");
    expect(gdpr?.hasGDPR).toBe(true);
  });

  it("numeric types have hasNumericOpts flag", () => {
    const numericTypes = FIELD_TYPES.filter((t) => t.hasNumericOpts).map((t) => t.value);
    expect(numericTypes.sort()).toEqual(["number", "range"]);
  });
});

describe("typeLabelMap", () => {
  it("maps every FIELD_TYPES value to its label", () => {
    for (const t of FIELD_TYPES) {
      expect(typeLabelMap[t.value]).toBe(t.label);
    }
  });

  it("has the same number of entries as FIELD_TYPES", () => {
    expect(Object.keys(typeLabelMap).length).toBe(FIELD_TYPES.length);
  });
});
