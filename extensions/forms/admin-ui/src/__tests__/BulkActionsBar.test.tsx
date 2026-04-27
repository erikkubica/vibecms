import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import BulkActionsBar from "../submissions/BulkActionsBar";

describe("BulkActionsBar", () => {
  const defaultProps = {
    selectedIds: new Set([1, 2, 3]),
    onClearSelection: vi.fn(),
    onBulkComplete: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("displays the selected count", () => {
    render(<BulkActionsBar {...defaultProps} />);
    expect(screen.getByText("3 selected")).toBeInTheDocument();
  });

  it("renders all bulk action buttons", () => {
    render(<BulkActionsBar {...defaultProps} />);
    expect(screen.getByText("Mark Read")).toBeInTheDocument();
    expect(screen.getByText("Mark Unread")).toBeInTheDocument();
    expect(screen.getByText("Archive")).toBeInTheDocument();
    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("renders Clear selection button", () => {
    render(<BulkActionsBar {...defaultProps} />);
    expect(screen.getByText("Clear selection")).toBeInTheDocument();
  });

  it("calls onClearSelection when Clear selection clicked", () => {
    const onClearSelection = vi.fn();
    render(
      <BulkActionsBar
        {...defaultProps}
        onClearSelection={onClearSelection}
      />,
    );
    fireEvent.click(screen.getByText("Clear selection"));
    expect(onClearSelection).toHaveBeenCalledTimes(1);
  });

  it("calls fetch with mark_read action when Mark Read clicked", async () => {
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({ count: 3 }),
    } as Response);

    render(<BulkActionsBar {...defaultProps} />);
    fireEvent.click(screen.getByText("Mark Read"));

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/admin/api/ext/forms/submissions/bulk",
        expect.objectContaining({
          method: "POST",
        }),
      );
    });

    const bodyStr = (fetchSpy.mock.calls[0][1] as RequestInit).body as string;
    expect(JSON.parse(bodyStr)).toMatchObject({ action: "mark_read" });
    fetchSpy.mockRestore();
  });

  it("calls fetch with archive action when Archive clicked", async () => {
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({ count: 3 }),
    } as Response);

    render(<BulkActionsBar {...defaultProps} />);
    fireEvent.click(screen.getByText("Archive"));

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalled();
    });

    const bodyStr = (fetchSpy.mock.calls[0][1] as RequestInit).body as string;
    expect(JSON.parse(bodyStr)).toMatchObject({ action: "archive" });
    fetchSpy.mockRestore();
  });

  it("shows confirm dialog before delete and aborts if cancelled", async () => {
    const confirmSpy = vi.spyOn(window, "confirm").mockReturnValue(false);
    const fetchSpy = vi.spyOn(global, "fetch");

    render(<BulkActionsBar {...defaultProps} />);
    fireEvent.click(screen.getByText("Delete"));

    expect(confirmSpy).toHaveBeenCalled();
    expect(fetchSpy).not.toHaveBeenCalled();

    confirmSpy.mockRestore();
    fetchSpy.mockRestore();
  });

  it("proceeds with delete when confirm returns true", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({ count: 3 }),
    } as Response);

    render(<BulkActionsBar {...defaultProps} />);
    fireEvent.click(screen.getByText("Delete"));

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalled();
    });

    const bodyStr = (fetchSpy.mock.calls[0][1] as RequestInit).body as string;
    expect(JSON.parse(bodyStr)).toMatchObject({ action: "delete" });

    vi.restoreAllMocks();
  });
});
