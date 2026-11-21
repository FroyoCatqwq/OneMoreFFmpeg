package platform

import (
	"os"
	"sync"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// LinkTargetKind declares what a reserved link is meant to point at. Windows
// needs this explicitly: os.Symlink detects the target type via Stat and
// creates a FILE symlink when the target is missing, which silently breaks a
// caller that intended a directory link. Unix has no type bit and keeps
// allowing dangling links; kind is still validated for API consistency.
type LinkTargetKind uint8

const (
	LinkTargetFile LinkTargetKind = iota + 1
	LinkTargetDir
)

// linkReservation is platform's own private copy of the release state. It is
// intentionally NOT shared with fsatomic.reservation: the two packages must
// stay independently implementable (PR-5b does not depend on PR-5 internals).
type linkReservation struct {
	mu       sync.Mutex
	path     string
	info     os.FileInfo
	resolved bool
	result   error
}

func (r *linkReservation) release(fn func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.resolved {
		return r.result // cached, including a first-call failure
	}
	r.result = fn()
	r.resolved = true
	return r.result
}

// ReserveLink atomically creates a uniquely named link pointing at target.
//
// PRECONDITION: dir must be a trusted, isolated directory. Lstat + SameFile +
// Remove has a TOCTOU window and does not provide absolute protection against
// untrusted concurrent directory writers.
//
// Lifecycle: pattern expands to a unique name; the link is created with
// createOneLink (platform-specific, fails if the name already exists); its
// Lstat identity is recorded. release() verifies identity, returns nil if the
// link was renamed/committed, E5009 ownership_changed if replaced by another
// resource, and removes it otherwise. Repeated release calls return the
// cached first result. On Windows, removal is os.Remove (NEVER RemoveAll) so
// a junction/symlink cannot recursively delete its target.
func ReserveLink(dir, pattern, target string, kind LinkTargetKind) (path string, release func() error, err error) {
	if kind != LinkTargetFile && kind != LinkTargetDir {
		return "", nil, apperrors.New(apperrors.E7005, "link target kind is invalid").WithDetails(map[string]any{"kind": kind})
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, apperrors.New(apperrors.E5009, "link reservation directory cannot be created").WithCause(err.Error())
	}
	// reservePath gives us a unique, non-existing name (it creates and removes
	// a placeholder atomically). createOneLink then fails if a racer claims it.
	name, err := reservePath(dir, pattern)
	if err != nil {
		return "", nil, err
	}
	if createErr := createOneLink(name, target, kind); createErr != nil {
		return "", nil, createErr
	}
	info, statErr := os.Lstat(name) // MUST be Lstat: Stat would follow the link
	if statErr != nil {
		_ = RemoveLink(name)
		return "", nil, apperrors.New(apperrors.E5009, "reserved link identity cannot be read").WithCause(statErr.Error())
	}
	res := &linkReservation{path: name, info: info}
	release = func() error {
		return res.release(func() error {
			current, statErr := os.Lstat(name)
			if os.IsNotExist(statErr) {
				return nil
			}
			if statErr != nil {
				return apperrors.New(apperrors.E5009, "reserved link cannot be inspected").WithCause(statErr.Error()).
					WithDetails(map[string]any{"reason": "stat_failed", "path": name})
			}
			if !os.SameFile(res.info, current) {
				return apperrors.New(apperrors.E5009, "reserved link now owns a different resource").
					WithDetails(map[string]any{"reason": "ownership_changed", "path": name})
			}
			if removeErr := RemoveLink(name); removeErr != nil && !os.IsNotExist(removeErr) {
				return apperrors.New(apperrors.E5009, "reserved link cleanup failed").WithCause(removeErr.Error()).
					WithDetails(map[string]any{"reason": "cleanup_failed", "path": name})
			}
			return nil
		})
	}
	return name, release, nil
}
