import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import ConditionBuilder, { ConditionGroup } from "../ConditionBuilder";

const sampleFields = [
  { id: "name", label: "Name" },
  { id: "email", label: "Email" },
  { id: "age", label: "Age" },
];

function renderBuilder(
  group: ConditionGroup,
  onChange = vi.fn(),
  excludeFieldId?: string,
) {
  const { rerender } = render(
    <ConditionBuilder
      group={group}
      onChange={onChange}
      fields={sampleFields}
      excludeFieldId={excludeFieldId}
    />,
  );
  return { onChange, rerender };
}

describe("ConditionBuilder", () => {
  it("renders ALL (AND) mode toggle by default on empty group", () => {
    renderBuilder({});
    expect(screen.getByText("ALL (AND)")).toBeInTheDocument();
    expect(screen.getByText("ANY (OR)")).toBeInTheDocument();
  });

  it("shows ANY mode when group has any key", () => {
    renderBuilder({ any: [] });
    // The ANY button should appear selected (bg-indigo-600 class)
    const anyBtn = screen.getByText("ANY (OR)");
    expect(anyBtn).toBeInTheDocument();
  });

  it("calls onChange with any mode when ANY button clicked", () => {
    const onChange = vi.fn();
    renderBuilder({ all: [] }, onChange);
    fireEvent.click(screen.getByText("ANY (OR)"));
    expect(onChange).toHaveBeenCalledWith({ any: [] });
  });

  it("calls onChange with all mode when ALL button clicked while in any mode", () => {
    const onChange = vi.fn();
    renderBuilder({ any: [] }, onChange);
    fireEvent.click(screen.getByText("ALL (AND)"));
    expect(onChange).toHaveBeenCalledWith({ all: [] });
  });

  it("adds a condition row when 'Add condition' button is clicked", () => {
    const onChange = vi.fn();
    renderBuilder({ all: [] }, onChange);
    fireEvent.click(screen.getByText("Add condition"));
    const called = onChange.mock.calls[0][0] as ConditionGroup;
    expect(called.all).toHaveLength(1);
    expect((called.all![0] as any).operator).toBe("equals");
  });

  it("adds a nested group when 'Add group' button is clicked", () => {
    const onChange = vi.fn();
    renderBuilder({ all: [] }, onChange);
    fireEvent.click(screen.getByText("Add group"));
    const called = onChange.mock.calls[0][0] as ConditionGroup;
    expect(called.all).toHaveLength(1);
    expect((called.all![0] as any).all).toBeDefined();
  });

  it("renders existing condition rows", () => {
    renderBuilder({
      all: [
        { field: "name", operator: "equals", value: "Erik" },
      ],
    });
    // The field select should show 'name' option; row should exist
    const selects = screen.getAllByRole("combobox");
    expect(selects.length).toBeGreaterThan(0);
  });

  it("hides 'Add group' at max depth (depth=3)", () => {
    render(
      <ConditionBuilder
        group={{ all: [] }}
        onChange={vi.fn()}
        fields={sampleFields}
        depth={3}
      />,
    );
    expect(screen.queryByText("Add group")).not.toBeInTheDocument();
  });

  it("excludes the target field from condition field options", () => {
    renderBuilder(
      { all: [{ field: "", operator: "equals", value: "" }] },
      vi.fn(),
      "name",
    );
    // The 'name' field should not appear as an option in the select
    const selects = screen.getAllByRole("combobox");
    const fieldSelect = selects[0] as HTMLSelectElement;
    const optionValues = Array.from(fieldSelect.options).map((o) => o.value);
    expect(optionValues).not.toContain("name");
    expect(optionValues).toContain("email");
  });
});
