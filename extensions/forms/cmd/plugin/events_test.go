package main

import (
	"testing"
)

func TestEmitSubmitted_PayloadFields(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	data := map[string]any{"name": "Alice", "email": "alice@example.com"}
	meta := map[string]any{"ip": "1.2.3.4"}
	p.emitSubmitted(ctx(), 7, "contact-us", 99, data, meta)

	if len(h.Emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(h.Emitted))
	}
	ev := h.Emitted[0]
	if ev.Action != "forms:submitted" {
		t.Errorf("action: got %q, want forms:submitted", ev.Action)
	}
	if ev.Payload["form_id"] != float64(7) && ev.Payload["form_id"] != uint(7) && ev.Payload["form_id"] != 7 {
		t.Errorf("form_id: got %v (%T)", ev.Payload["form_id"], ev.Payload["form_id"])
	}
	if ev.Payload["form_slug"] != "contact-us" {
		t.Errorf("form_slug: got %v", ev.Payload["form_slug"])
	}
	if ev.Payload["submission_id"] != float64(99) && ev.Payload["submission_id"] != uint(99) && ev.Payload["submission_id"] != 99 {
		t.Errorf("submission_id: got %v (%T)", ev.Payload["submission_id"], ev.Payload["submission_id"])
	}
}

func TestEmitSubmitted_DataPresent(t *testing.T) {
	h := NewFakeHost()
	p := newPlugin(h)

	data := map[string]any{"field1": "value1"}
	meta := map[string]any{}
	p.emitSubmitted(ctx(), 1, "slug", 1, data, meta)

	if len(h.Emitted) == 0 {
		t.Fatal("expected event to be emitted")
	}
	payload := h.Emitted[0].Payload
	if payload["data"] == nil {
		t.Error("expected data in payload")
	}
	if payload["metadata"] == nil {
		t.Error("expected metadata in payload")
	}
}
