package main

import "testing"

func TestRecentSharesCapAndOrder(t *testing.T) {
	app := newTestApp(t)
	for i := 0; i < 15; i++ {
		app.addRecentShare("title", "https://x/"+string(rune('a'+i)))
	}
	shares := app.RecentShares()
	if len(shares) != 10 {
		t.Fatalf("recent shares capped at 10, got %d", len(shares))
	}
	// Newest first: the last added URL leads.
	if shares[0].URL != "https://x/o" {
		t.Errorf("newest-first order wrong: %q", shares[0].URL)
	}
}

func TestRecentSharesEmptyIsNonNil(t *testing.T) {
	app := newTestApp(t)
	if got := app.RecentShares(); got == nil || len(got) != 0 {
		t.Errorf("empty RecentShares = %v, want []", got)
	}
}

func TestClearRecentSharesPersists(t *testing.T) {
	app := newTestApp(t)
	app.addRecentShare("t", "https://x/1")
	if err := app.ClearRecentShares(); err != nil {
		t.Fatal(err)
	}
	if len(app.RecentShares()) != 0 {
		t.Error("shares not cleared")
	}
}

func TestSaveTargetOptionSetAndDelete(t *testing.T) {
	app := newTestApp(t)
	tgt, err := app.AddTarget("folder", "F", map[string]string{"path": "/tmp"})
	if err != nil {
		t.Fatal(err)
	}
	app.saveTargetOption(tgt.ID, "token", "abc")
	got, _ := app.grid.Get(tgt.ID)
	if got.Options["token"] != "abc" {
		t.Errorf("option not saved: %+v", got.Options)
	}
	// Empty value deletes the key.
	app.saveTargetOption(tgt.ID, "token", "")
	got, _ = app.grid.Get(tgt.ID)
	if _, ok := got.Options["token"]; ok {
		t.Errorf("empty value should delete key: %+v", got.Options)
	}
}
