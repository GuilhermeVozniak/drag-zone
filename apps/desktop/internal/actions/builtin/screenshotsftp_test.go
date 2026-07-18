package builtin

import (
	"context"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func screenshotSFTPBaseOptions() map[string]string {
	return map[string]string{"host": "example.com", "username": "bob", "password": "secret"}
}

func TestScreenshotSFTPSpec(t *testing.T) {
	spec := ScreenshotSFTP{}.Spec()
	if spec.ID != "screenshot-sftp" {
		t.Errorf("ID = %q, want %q", spec.ID, "screenshot-sftp")
	}
	if spec.Name != "Screenshot & Upload" {
		t.Errorf("Name = %q, want %q", spec.Name, "Screenshot & Upload")
	}
	if spec.Category != "Capture" {
		t.Errorf("Category = %q, want %q", spec.Category, "Capture")
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventClicked {
		t.Errorf("Events = %v, want [%s]", spec.Events, model.EventClicked)
	}
	if !spec.Multi {
		t.Error("expected Multi = true")
	}
	byKey := map[string]model.OptionField{}
	for _, o := range spec.Options {
		byKey[o.Key] = o
	}
	for _, key := range []string{"mode", "protocol", "host", "port", "username", "password", "remote_dir", "url_prefix"} {
		if _, ok := byKey[key]; !ok {
			t.Errorf("missing option %q", key)
		}
	}
	if !byKey["host"].Required || !byKey["username"].Required || !byKey["password"].Required {
		t.Error("host, username and password must be required")
	}
}

func TestScreenshotSFTPUploadsCaptureAndBuildsURL(t *testing.T) {
	withScreenRecordingGranted(t)
	withFakeScreenshotCmd(t, true)
	fr := &fakeRemote{}
	withFakeRemote(t, fr)
	svc := &recServices{}

	opts := screenshotSFTPBaseOptions()
	opts["url_prefix"] = "https://cdn.example.com/shots/"
	res, err := ScreenshotSFTP{}.Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{Options: opts},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Clicked: %v", err)
	}

	if len(fr.uploads) != 1 {
		t.Fatalf("upload count = %d, want 1", len(fr.uploads))
	}
	if string(fr.bodies[fr.uploads[0]]) != "fakepng" {
		t.Errorf("uploaded body = %q, want %q", fr.bodies[fr.uploads[0]], "fakepng")
	}
	wantURL := "https://cdn.example.com/shots/" + fr.uploads[0]
	if res.URL != wantURL {
		t.Errorf("URL = %q, want %q", res.URL, wantURL)
	}
	if svc.Clipboard != wantURL {
		t.Errorf("clipboard = %q, want %q", svc.Clipboard, wantURL)
	}
	if !fr.closed {
		t.Error("remote connection was not closed")
	}
}

func TestScreenshotSFTPNoURLPrefixSkipsClipboard(t *testing.T) {
	withScreenRecordingGranted(t)
	withFakeScreenshotCmd(t, true)
	fr := &fakeRemote{}
	withFakeRemote(t, fr)
	svc := &recServices{}

	res, err := ScreenshotSFTP{}.Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{Options: screenshotSFTPBaseOptions()},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Clicked: %v", err)
	}
	if res.URL != "" {
		t.Errorf("URL = %q, want empty when no url_prefix configured", res.URL)
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard = %q, want untouched", svc.Clipboard)
	}
	if len(fr.uploads) != 1 {
		t.Errorf("upload count = %d, want 1", len(fr.uploads))
	}
}

func TestScreenshotSFTPMissingCredentials(t *testing.T) {
	withScreenRecordingGranted(t)
	base := screenshotSFTPBaseOptions()
	for _, missing := range []string{"host", "username", "password"} {
		opts := map[string]string{}
		for k, v := range base {
			if k != missing {
				opts[k] = v
			}
		}
		_, err := ScreenshotSFTP{}.Clicked(context.Background(), actions.Invocation{
			Target:   model.Target{Options: opts},
			Progress: nullProgress{},
			Services: &recServices{},
		})
		if err == nil {
			t.Errorf("missing %s should error", missing)
		}
	}
}

func TestScreenshotSFTPCancelledCaptureSkipsUpload(t *testing.T) {
	withScreenRecordingGranted(t)
	withFakeScreenshotCmd(t, false) // simulate Esc: screencapture writes nothing
	fr := &fakeRemote{}
	withFakeRemote(t, fr)
	svc := &recServices{}

	res, err := ScreenshotSFTP{}.Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{Options: screenshotSFTPBaseOptions()},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("cancelled capture must not be an error, got %v", err)
	}
	if res.Message != "Screenshot cancelled" {
		t.Errorf("Message = %q, want %q", res.Message, "Screenshot cancelled")
	}
	if len(fr.uploads) != 0 {
		t.Errorf("no upload should happen on a cancelled capture, got %v", fr.uploads)
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard should be untouched on a cancelled capture, got %q", svc.Clipboard)
	}
}

func TestScreenshotSFTPRequestsPermissionWhenMissing(t *testing.T) {
	origHas, origReq := hasScreenRecording, requestScreenRecording
	t.Cleanup(func() {
		hasScreenRecording = origHas
		requestScreenRecording = origReq
	})
	hasScreenRecording = func() bool { return false }
	requested := false
	requestScreenRecording = func() { requested = true }

	calls := withFakeScreenshotCmd(t, true)
	fr := &fakeRemote{}
	withFakeRemote(t, fr)
	svc := &recServices{}

	res, err := ScreenshotSFTP{}.Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{Options: screenshotSFTPBaseOptions()},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != screenRecordingPermissionMessage {
		t.Errorf("Message = %q, want %q", res.Message, screenRecordingPermissionMessage)
	}
	if !requested {
		t.Error("expected requestScreenRecording to be called")
	}
	if len(*calls) != 0 {
		t.Errorf("expected screenshotCmd not to be invoked, got %+v", *calls)
	}
	if len(fr.uploads) != 0 {
		t.Errorf("no upload should happen when permission is missing, got %v", fr.uploads)
	}
}
