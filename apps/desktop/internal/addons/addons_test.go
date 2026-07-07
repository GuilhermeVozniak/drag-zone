package addons

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFiltersDzbundleDirs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header: %q", r.Header.Get("Accept"))
		}
		w.Write([]byte(`[
			{"name":"Alpha.dzbundle","type":"dir"},
			{"name":"Beta.dzbundle","type":"dir"},
			{"name":"README.md","type":"file"},
			{"name":"NotABundle","type":"dir"}
		]`))
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()

	names, err := List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Errorf("names = %v, want [Alpha Beta]", names)
	}
}

func TestListServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()
	if _, err := List(context.Background()); err == nil {
		t.Error("403 should error")
	}
}
