import { describe, it, expect } from "vitest";
import { keyify, normalizeOptions } from "../key-utils";

describe("keyify", () => {
  it("lowercases input", () => {
    expect(keyify("HELLO")).toBe("hello");
  });

  it("replaces spaces and special chars with underscores", () => {
    expect(keyify("Hello World")).toBe("hello_world");
    expect(keyify("First & Last Name")).toBe("first_last_name");
  });

  it("trims leading and trailing underscores", () => {
    expect(keyify("__foo__")).toBe("foo");
    expect(keyify("  text  ")).toBe("text");
  });

  it("collapses consecutive underscores", () => {
    expect(keyify("a--b__c  d")).toBe("a_b_c_d");
  });

  it("handles empty string", () => {
    expect(keyify("")).toBe("");
  });

  it("preserves digits", () => {
    expect(keyify("field123")).toBe("field123");
  });

  it("handles already-keyified strings unchanged", () => {
    expect(keyify("my_field")).toBe("my_field");
  });
});

describe("normalizeOptions", () => {
  it("returns empty array for null/undefined", () => {
    expect(normalizeOptions(null)).toEqual([]);
    expect(normalizeOptions(undefined)).toEqual([]);
  });

  it("returns empty array for empty input", () => {
    expect(normalizeOptions([])).toEqual([]);
  });

  it("passes through already-normalised object array", () => {
    const input = [{ label: "Red", value: "red" }];
    expect(normalizeOptions(input)).toEqual(input);
  });

  it("converts legacy string array to {label, value} pairs", () => {
    expect(normalizeOptions(["Red", "Green"])).toEqual([
      { label: "Red", value: "Red" },
      { label: "Green", value: "Green" },
    ]);
  });

  it("trims strings in legacy format", () => {
    expect(normalizeOptions(["  hi  "])).toEqual([{ label: "hi", value: "hi" }]);
  });

  it("filters out empty strings in legacy format", () => {
    expect(normalizeOptions(["a", "", "  "])).toEqual([{ label: "a", value: "a" }]);
  });

  it("returns empty array for non-array input", () => {
    expect(normalizeOptions({ label: "x" })).toEqual([]);
  });
});
