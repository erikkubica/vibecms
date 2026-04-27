import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import SubmissionDetailDialog from "../submissions/SubmissionDetailDialog";

const baseSubmission = {
  id: 42,
  form_id: 1,
  form_name: "Contact Us",
  status: "unread" as const,
  created_at: "2024-01-15T10:30:00Z",
  data: {
    name: "Erik Kubica",
    email: "erik@example.com",
    message: "Hello world",
  },
  metadata: {},
  form_fields: [
    { id: "name", label: "Full Name" },
    { id: "email", label: "Email Address" },
    { id: "message", label: "Message" },
  ],
};

describe("SubmissionDetailDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when submission is null", () => {
    const { container } = render(
      <SubmissionDetailDialog submission={null} onClose={vi.fn()} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders submission details dialog when open", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Submission Details")).toBeInTheDocument();
  });

  it("renders submission ID in dialog header", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByText("ID: #42")).toBeInTheDocument();
  });

  it("renders form name in meta grid", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Contact Us")).toBeInTheDocument();
  });

  it("maps field ids to human labels from form_fields", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    // form_fields provides label mapping
    expect(screen.getByText("Full Name")).toBeInTheDocument();
    expect(screen.getByText("Email Address")).toBeInTheDocument();
    expect(screen.getByText("Message")).toBeInTheDocument();
  });

  it("renders submitted data values", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Erik Kubica")).toBeInTheDocument();
    expect(screen.getByText("erik@example.com")).toBeInTheDocument();
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("shows Archive button when submission is not archived", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Archive")).toBeInTheDocument();
  });

  it("shows Delete button", () => {
    render(
      <SubmissionDetailDialog submission={baseSubmission} onClose={vi.fn()} />,
    );
    expect(screen.getByText("Delete")).toBeInTheDocument();
  });

  it("shows Unarchive button when submission is archived", () => {
    render(
      <SubmissionDetailDialog
        submission={{ ...baseSubmission, status: "archived" }}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("Unarchive")).toBeInTheDocument();
    expect(screen.queryByText("Archive")).not.toBeInTheDocument();
  });

  it("shows Mark Unread button when submission is read", () => {
    render(
      <SubmissionDetailDialog
        submission={{ ...baseSubmission, status: "read" }}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("Mark Unread")).toBeInTheDocument();
  });

  it("calls PATCH on status change", async () => {
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({}),
    } as Response);
    const onStatusChange = vi.fn();

    render(
      <SubmissionDetailDialog
        submission={baseSubmission}
        onClose={vi.fn()}
        onStatusChange={onStatusChange}
      />,
    );

    fireEvent.click(screen.getByText("Archive"));

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/admin/api/ext/forms/submissions/42",
        expect.objectContaining({ method: "PATCH" }),
      );
    });

    fetchSpy.mockRestore();
  });

  it("calls DELETE and onDeleted when Delete confirmed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({}),
    } as Response);
    const onDeleted = vi.fn();

    render(
      <SubmissionDetailDialog
        submission={baseSubmission}
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    );

    fireEvent.click(screen.getByText("Delete"));

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/admin/api/ext/forms/submissions/42",
        expect.objectContaining({ method: "DELETE" }),
      );
    });

    vi.restoreAllMocks();
  });

  it("renders boolean true as checkmark", () => {
    render(
      <SubmissionDetailDialog
        submission={{
          ...baseSubmission,
          data: { gdpr: true },
          form_fields: [{ id: "gdpr", label: "GDPR Consent" }],
        }}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("✓")).toBeInTheDocument();
  });

  it("renders boolean false as cross", () => {
    render(
      <SubmissionDetailDialog
        submission={{
          ...baseSubmission,
          data: { gdpr: false },
          form_fields: [{ id: "gdpr", label: "GDPR Consent" }],
        }}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("✗")).toBeInTheDocument();
  });
});
