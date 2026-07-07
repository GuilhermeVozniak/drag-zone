package builtin

import "testing"

func TestS3PublicURL(t *testing.T) {
	// Default virtual-hosted-style URL.
	got := s3PublicURL("mybucket", "us-east-1", "", "uploads/a.png")
	want := "https://mybucket.s3.us-east-1.amazonaws.com/uploads/a.png"
	if got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
	// Custom prefix wins and trailing slash is trimmed.
	got = s3PublicURL("b", "eu-west-1", "https://cdn.example.com/", "k/x.jpg")
	if got != "https://cdn.example.com/k/x.jpg" {
		t.Errorf("custom prefix = %q", got)
	}
}
