package settings

import (
	"strings"
	"testing"
)

func TestRegistryRegisterValidates(t *testing.T) {
	t.Parallel()
	t.Run("empty id rejected", func(t *testing.T) {
		r := NewRegistry()
		if err := r.Register(Schema{Title: "x", Sections: []Section{{Title: "s", Fields: []Field{{Key: "k", Type: "text"}}}}}); err == nil {
			t.Fatal("expected error for empty ID")
		}
	})
	t.Run("empty title rejected", func(t *testing.T) {
		r := NewRegistry()
		if err := r.Register(Schema{ID: "x", Sections: []Section{{Title: "s", Fields: []Field{{Key: "k", Type: "text"}}}}}); err == nil {
			t.Fatal("expected error for empty title")
		}
	})
	t.Run("no sections rejected", func(t *testing.T) {
		r := NewRegistry()
		if err := r.Register(Schema{ID: "x", Title: "X"}); err == nil {
			t.Fatal("expected error for empty sections")
		}
	})
	t.Run("duplicate keys rejected", func(t *testing.T) {
		r := NewRegistry()
		err := r.Register(Schema{ID: "x", Title: "X", Sections: []Section{
			{Title: "S", Fields: []Field{
				{Key: "k", Type: "text"},
				{Key: "k", Type: "text"},
			}},
		}})
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate-key error, got %v", err)
		}
	})
}

func TestSchemaFieldByKey(t *testing.T) {
	t.Parallel()
	s := Schema{
		ID: "x", Title: "X",
		Sections: []Section{
			{Title: "A", Fields: []Field{{Key: "alpha", Type: "text", Translatable: true}}},
			{Title: "B", Fields: []Field{{Key: "beta", Type: "toggle"}}},
		},
	}
	if f := s.FieldByKey("alpha"); f == nil || !f.Translatable {
		t.Fatalf("expected translatable alpha, got %+v", f)
	}
	if f := s.FieldByKey("beta"); f == nil || f.Translatable {
		t.Fatalf("expected non-translatable beta, got %+v", f)
	}
	if f := s.FieldByKey("missing"); f != nil {
		t.Fatalf("expected nil for missing key, got %+v", f)
	}
}

func TestSchemaHasTranslatable(t *testing.T) {
	t.Parallel()
	all := Schema{ID: "x", Title: "X", Sections: []Section{
		{Title: "A", Fields: []Field{{Key: "a", Type: "toggle"}}},
	}}
	if all.HasTranslatable() {
		t.Fatal("expected false for all-global schema")
	}
	mixed := Schema{ID: "x", Title: "X", Sections: []Section{
		{Title: "A", Fields: []Field{
			{Key: "a", Type: "text", Translatable: true},
			{Key: "b", Type: "toggle"},
		}},
	}}
	if !mixed.HasTranslatable() {
		t.Fatal("expected true for mixed schema")
	}
}

func TestRegistryUnregister(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.MustRegister(Schema{ID: "x", Title: "X", Sections: []Section{{Title: "S", Fields: []Field{{Key: "k", Type: "text"}}}}})
	if _, ok := r.Get("x"); !ok {
		t.Fatal("expected schema present")
	}
	r.Unregister("x")
	if _, ok := r.Get("x"); ok {
		t.Fatal("expected schema removed")
	}
}

func TestRegisterBuiltinsValid(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	RegisterBuiltins(r)
	for _, want := range []string{"security", "site.general", "site.seo", "site.advanced"} {
		if _, ok := r.Get(want); !ok {
			t.Errorf("expected built-in schema %q", want)
		}
	}
}
