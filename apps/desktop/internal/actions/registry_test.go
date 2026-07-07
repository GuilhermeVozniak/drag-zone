package actions

import (
	"testing"

	"dragzone/internal/model"
)

type fakeAction struct{ id string }

func (f fakeAction) Spec() model.ActionSpec { return model.ActionSpec{ID: f.id, Name: f.id} }

func TestRegistryRegisterGetSpecsOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeAction{"a"})
	r.Register(fakeAction{"b"})

	if _, err := r.Get("a"); err != nil {
		t.Errorf("Get(a): %v", err)
	}
	if _, err := r.Get("missing"); err == nil {
		t.Error("Get(missing) should error")
	}
	specs := r.Specs()
	if len(specs) != 2 || specs[0].ID != "a" || specs[1].ID != "b" {
		t.Errorf("Specs order = %+v, want [a b]", specs)
	}
}

func TestSpecsNeverNil(t *testing.T) {
	if specs := NewRegistry().Specs(); specs == nil {
		t.Error("Specs() on empty registry returned nil")
	}
}

func TestTryRegisterRejectsDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.TryRegister(fakeAction{"dup"}); err != nil {
		t.Fatal(err)
	}
	if err := r.TryRegister(fakeAction{"dup"}); err == nil {
		t.Error("duplicate TryRegister should error")
	}
}

func TestRegisterPanicsOnDuplicate(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeAction{"x"})
	defer func() {
		if recover() == nil {
			t.Error("Register of a duplicate should panic")
		}
	}()
	r.Register(fakeAction{"x"})
}
