import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import NotificationCard from "../notifications/NotificationCard";

const baseNotif = {
  name: "Admin Notification",
  type: "admin",
  enabled: true,
  recipients: "admin@example.com",
  subject: "New submission",
  body: "You have a new message.",
};

const defaultProps = {
  notif: baseNotif,
  index: 0,
  formId: 1,
  emailFields: [{ id: "email", label: "Email" }],
  formFields: [],
  onUpdate: vi.fn(),
  onRemove: vi.fn(),
};

describe("NotificationCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders notification name", () => {
    render(<NotificationCard {...defaultProps} />);
    expect(screen.getByDisplayValue("Admin Notification")).toBeInTheDocument();
  });

  it("renders enabled switch", () => {
    render(<NotificationCard {...defaultProps} />);
    expect(screen.getByRole("switch")).toBeInTheDocument();
  });

  it("calls onUpdate when enabled switch toggled", () => {
    const onUpdate = vi.fn();
    render(<NotificationCard {...defaultProps} onUpdate={onUpdate} />);
    fireEvent.click(screen.getByRole("switch"));
    expect(onUpdate).toHaveBeenCalledWith(0, "enabled", false);
  });

  it("renders Admin badge for admin type", () => {
    render(<NotificationCard {...defaultProps} />);
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });

  it("renders Auto-Responder badge for auto-responder type", () => {
    render(
      <NotificationCard
        {...defaultProps}
        notif={{ ...baseNotif, type: "auto-responder", name: "Auto Reply" }}
      />,
    );
    expect(screen.getByText("Auto-Responder")).toBeInTheDocument();
  });

  it("shows subject input", () => {
    render(<NotificationCard {...defaultProps} />);
    expect(screen.getByDisplayValue("New submission")).toBeInTheDocument();
  });

  it("calls onUpdate when subject input changes", () => {
    const onUpdate = vi.fn();
    render(<NotificationCard {...defaultProps} onUpdate={onUpdate} />);
    const subjectInput = screen.getByDisplayValue("New submission");
    fireEvent.change(subjectInput, { target: { value: "Updated subject" } });
    expect(onUpdate).toHaveBeenCalledWith(0, "subject", "Updated subject");
  });

  it("calls onRemove when remove button clicked", () => {
    const onRemove = vi.fn();
    render(<NotificationCard {...defaultProps} onRemove={onRemove} />);
    // Find the delete button by its icon aria or its parent structure
    const buttons = screen.getAllByRole("button");
    // Remove button is the last icon button in the header area
    const removeBtn = buttons.find(
      (b) => b.querySelector('[data-testid="icon-Trash2"]'),
    );
    expect(removeBtn).toBeTruthy();
    fireEvent.click(removeBtn!);
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("shows CC/BCC section when toggled", () => {
    render(<NotificationCard {...defaultProps} />);
    fireEvent.click(screen.getByText("CC / BCC"));
    expect(screen.getByPlaceholderText("cc@example.com")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("bcc@example.com")).toBeInTheDocument();
  });

  it("calls onUpdate when recipients input changes", () => {
    const onUpdate = vi.fn();
    render(<NotificationCard {...defaultProps} onUpdate={onUpdate} />);
    const recipientsInput = screen.getByDisplayValue("admin@example.com");
    fireEvent.change(recipientsInput, { target: { value: "new@example.com" } });
    expect(onUpdate).toHaveBeenCalledWith(0, "recipients", "new@example.com");
  });

  it("send test: triggers fetch POST when formId is set", async () => {
    const fetchSpy = vi.spyOn(global, "fetch").mockResolvedValue({
      ok: true,
      json: async () => ({ message: "Test sent" }),
    } as Response);

    render(<NotificationCard {...defaultProps} />);

    // Click the Send test icon button (Send icon button)
    const sendBtns = screen.getAllByRole("button").filter((b) =>
      b.querySelector('[data-testid="icon-Send"]'),
    );
    // Click the trigger that opens the popover
    fireEvent.click(sendBtns[0]);

    // Now click the "Send Test" button inside the popover content
    const sendTestBtn = screen.getByText("Send Test");
    fireEvent.click(sendTestBtn);

    await vi.waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith(
        "/admin/api/ext/forms/1/notifications/0/test",
        expect.objectContaining({ method: "POST" }),
      );
    });

    fetchSpy.mockRestore();
  });
});
