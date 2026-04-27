import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import FieldEditor from "../FieldEditor";

function makeField(overrides: Record<string, any> = {}) {
  return {
    id: "field_1",
    label: "My Field",
    type: "text",
    required: false,
    ...overrides,
  };
}

describe("FieldEditor — label and key inputs", () => {
  it("renders label and key inputs with correct values", () => {
    render(
      <FieldEditor
        field={makeField({ label: "Your Name", id: "your_name" })}
        updateField={vi.fn()}
      />,
    );
    expect(screen.getByDisplayValue("Your Name")).toBeInTheDocument();
    expect(screen.getByDisplayValue("your_name")).toBeInTheDocument();
  });

  it("calls updateField when label input changes", () => {
    const updateField = vi.fn();
    render(
      <FieldEditor field={makeField({ label: "Old Label" })} updateField={updateField} />,
    );
    const labelInput = screen.getByDisplayValue("Old Label");
    fireEvent.change(labelInput, { target: { value: "New Label" } });
    expect(updateField).toHaveBeenCalledWith({ label: "New Label" });
  });

  it("strips non-alphanumeric chars from key input", () => {
    const updateField = vi.fn();
    render(
      <FieldEditor field={makeField({ id: "my_key" })} updateField={updateField} />,
    );
    const keyInput = screen.getByDisplayValue("my_key");
    fireEvent.change(keyInput, { target: { value: "my key!" } });
    // onChange handler does .replace(/[^a-z0-9_]/g, "")
    expect(updateField).toHaveBeenCalledWith({ id: "my key!".replace(/[^a-z0-9_]/g, "") });
  });
});

describe("FieldEditor — validation rules", () => {
  it("shows Min/Max Length inputs for text type", () => {
    render(<FieldEditor field={makeField({ type: "text" })} updateField={vi.fn()} />);
    expect(screen.getByPlaceholderText("No minimum")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("No maximum")).toBeInTheDocument();
  });

  it("shows numeric min/max/step inputs for number type", () => {
    render(<FieldEditor field={makeField({ type: "number" })} updateField={vi.fn()} />);
    expect(screen.getByPlaceholderText("No min")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("No max")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("1")).toBeInTheDocument();
  });

  it("does not show length inputs for checkbox type", () => {
    render(<FieldEditor field={makeField({ type: "checkbox" })} updateField={vi.fn()} />);
    expect(screen.queryByPlaceholderText("No minimum")).not.toBeInTheDocument();
  });
});

describe("FieldEditor — required toggle", () => {
  it("shows Required switch", () => {
    render(<FieldEditor field={makeField()} updateField={vi.fn()} />);
    expect(screen.getByRole("switch")).toBeInTheDocument();
  });

  it("calls updateField with required:true when switch toggled on", () => {
    const updateField = vi.fn();
    render(<FieldEditor field={makeField({ required: false })} updateField={updateField} />);
    const switchEl = screen.getByRole("switch");
    fireEvent.click(switchEl);
    expect(updateField).toHaveBeenCalledWith({ required: true });
  });

  it("disables Required switch for GDPR fields", () => {
    render(
      <FieldEditor
        field={makeField({ type: "gdpr_consent", required: true })}
        updateField={vi.fn()}
      />,
    );
    const switchEl = screen.getByRole("switch");
    expect(switchEl).toBeDisabled();
  });
});

describe("FieldEditor — conditional visibility section", () => {
  it("renders 'Show this field when…' toggle button", () => {
    render(<FieldEditor field={makeField()} updateField={vi.fn()} />);
    expect(screen.getByText("Show this field when…")).toBeInTheDocument();
  });

  it("reveals ConditionBuilder when toggle clicked", () => {
    render(<FieldEditor field={makeField()} updateField={vi.fn()} allFields={[]} />);
    fireEvent.click(screen.getByText("Show this field when…"));
    // ConditionBuilder renders mode toggle buttons
    expect(screen.getByText("ALL (AND)")).toBeInTheDocument();
  });

  it("shows Active badge when display_when has conditions", () => {
    render(
      <FieldEditor
        field={makeField({
          display_when: { all: [{ field: "email", operator: "equals", value: "x" }] },
        })}
        updateField={vi.fn()}
      />,
    );
    expect(screen.getByText("Active")).toBeInTheDocument();
  });
});

describe("FieldEditor — width selector", () => {
  it("renders Full, 1/2, 1/3 buttons", () => {
    render(<FieldEditor field={makeField()} updateField={vi.fn()} />);
    expect(screen.getByText("Full")).toBeInTheDocument();
    expect(screen.getByText("1/2")).toBeInTheDocument();
    expect(screen.getByText("1/3")).toBeInTheDocument();
  });

  it("calls updateField with chosen width when a width button clicked", () => {
    const updateField = vi.fn();
    render(<FieldEditor field={makeField()} updateField={updateField} />);
    fireEvent.click(screen.getByText("1/2"));
    expect(updateField).toHaveBeenCalledWith({ width: "half" });
  });
});
