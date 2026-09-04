package release

import (
	"bytes"
	"testing"
)

func TestShouldCheckGating(t *testing.T) {
	// KDEN_NO_UPDATE_NOTIFIER disables regardless of everything else.
	t.Setenv("KDEN_NO_UPDATE_NOTIFIER", "1")
	if shouldCheck("v0.1.0") {
		t.Error("KDEN_NO_UPDATE_NOTIFIER set but shouldCheck returned true")
	}

	// CI disables it (checked before TTY, so this holds even on a TTY-less CI box).
	t.Setenv("KDEN_NO_UPDATE_NOTIFIER", "")
	t.Setenv("CI", "true")
	if shouldCheck("v0.1.0") {
		t.Error("CI set but shouldCheck returned true")
	}

	// A dev/non-semver build never checks — nothing to point an upgrade at.
	t.Setenv("CI", "")
	if shouldCheck("dev") {
		t.Error("non-semver current version should not trigger a check")
	}
}

func TestNotifyNilAndEmpty(t *testing.T) {
	var buf bytes.Buffer

	// nil Notifier must be a safe no-op (the gated-off path returns nil).
	var n *Notifier
	n.Notify(&buf)
	if buf.Len() != 0 {
		t.Errorf("nil Notifier wrote %q", buf.String())
	}

	// A Notifier whose check found nothing newer prints nothing.
	n2 := &Notifier{current: "v1.0.0", result: make(chan *Release, 1)}
	close(n2.result) // no newer release delivered
	n2.Notify(&buf)
	if buf.Len() != 0 {
		t.Errorf("Notifier with no update wrote %q", buf.String())
	}

	// A Notifier with a newer release prints the hint to the writer.
	n3 := &Notifier{current: "v1.0.0", result: make(chan *Release, 1)}
	n3.result <- &Release{Tag: "v1.1.0"}
	n3.Notify(&buf)
	if !bytes.Contains(buf.Bytes(), []byte("v1.1.0")) || !bytes.Contains(buf.Bytes(), []byte("kden upgrade")) {
		t.Errorf("expected upgrade hint, got %q", buf.String())
	}
}
