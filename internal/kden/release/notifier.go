package release

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/adrg/xdg"
	"golang.org/x/term"
)

// checkInterval throttles background update checks: at most once per day, gh-style.
const checkInterval = 24 * time.Hour

const stateFile = "kden/update-check.json"

type checkState struct {
	CheckedAt   time.Time `json:"checked_at"`
	LatestKnown string    `json:"latest_known"`
}

// Notifier performs a throttled, non-blocking check for a newer release and, if
// one is found, prints a one-line hint to stderr after the command runs. It
// follows gh's contract: gated on an interactive TTY, disabled in CI and under
// KDEN_NO_UPDATE_NOTIFIER, and never blocks the command it decorates.
type Notifier struct {
	current string
	result  chan *Release
}

// StartUpdateCheck kicks off a background check unless notifications are gated
// off. It returns a Notifier whose Notify method should be called after the
// command completes. A nil Notifier is valid and does nothing.
func StartUpdateCheck(ctx context.Context, current string) *Notifier {
	if !shouldCheck(current) {
		return nil
	}

	n := &Notifier{current: current, result: make(chan *Release, 1)}
	go func() {
		defer close(n.result)
		// A stale check is worthless; bound it tightly so a slow network never
		// delays the hint past the command's own runtime.
		cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		rel, err := New().Latest(cctx)
		if err != nil {
			return
		}
		recordCheck(rel.Tag)
		if Newer(current, rel.Tag) {
			n.result <- rel
		}
	}()
	return n
}

// Notify prints the update hint to w if the background check found a newer
// release. It never blocks: if the check hasn't finished, it says nothing.
func (n *Notifier) Notify(w io.Writer) {
	if n == nil {
		return
	}
	select {
	case rel, ok := <-n.result:
		if ok && rel != nil {
			_, _ = fmt.Fprintf(w, "\nkden %s is available (you have %s). Run 'kden upgrade'.\n", rel.Tag, n.current)
		}
	default:
		// check still running — don't wait, don't nag
	}
}

// shouldCheck applies the gh gating rules BEFORE any network call: a bare time
// throttle still spams CI logs, so gate on environment first.
func shouldCheck(current string) bool {
	if os.Getenv("KDEN_NO_UPDATE_NOTIFIER") != "" {
		return false
	}
	if isCI() {
		return false
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return false
	}
	// dev builds and non-semver refs have no "newer release" to point at.
	if _, err := semverParse(current); err != nil {
		return false
	}
	return !recentlyChecked()
}

func isCI() bool {
	// CI is set by virtually every CI system; the others catch the stragglers.
	return os.Getenv("CI") != "" ||
		os.Getenv("BUILD_NUMBER") != "" ||
		os.Getenv("RUN_ID") != ""
}

func recentlyChecked() bool {
	path, err := xdg.StateFile(stateFile)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var s checkState
	if json.Unmarshal(data, &s) != nil {
		return false
	}
	return time.Since(s.CheckedAt) < checkInterval
}

func recordCheck(latest string) {
	path, err := xdg.StateFile(stateFile)
	if err != nil {
		return
	}
	data, err := json.Marshal(checkState{CheckedAt: time.Now(), LatestKnown: latest})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o600)
}
