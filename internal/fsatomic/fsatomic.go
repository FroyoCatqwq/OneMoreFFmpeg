// Package fsatomic provides the small set of filesystem primitives that are
// genuinely shared by more than one caller.
//
// It deliberately does NOT unify persistence semantics: atomic replacement,
// cross-process locked writes, unique path reservation, and link/junction
// reservation are different operations with different guarantees. Callers
// keep their own error-code mapping and concurrency rules.
package fsatomic

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/rogpeppe/go-internal/lockedfile"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

// WriteFile replaces path with payload using a temp file created in the SAME
// directory (therefore the same volume). Execution order:
//
//	CreateTemp -> Write -> [Chmod] -> Sync -> Close -> RenamePath
//
// mode == 0 means "do not Chmod": the temp file keeps os.CreateTemp's 0600
// mode. A non-zero mode is applied via Chmod before Sync (mode semantics, not
// open-mode: umask does not further restrict it).
//
// Replacement semantics are deliberately conservative:
//   - Unix: rename(2) within one filesystem atomically replaces the directory
//     entry; readers see either the old or the new file, never a partial one.
//   - Windows: platform.RenamePath uses os.Rename with retries for sharing
//     violations; it does NOT promise the same atomicity on every filesystem.
//   - No cross-volume replacement (the temp file is always same-directory).
//   - If RenamePath fails, the previous target is left untouched.
//
// There is NO parent-directory fsync and NO crash-durability guarantee: a
// successful return means the running system observes the new file, not that
// a power loss would preserve it. Callers needing stronger guarantees must
// handle that at a higher layer.
//
// Errors are returned as apperrors.E5009 with a cause; callers may re-map.
func WriteFile(path string, payload []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return apperrors.New(apperrors.E5009, "atomic write temp file cannot be created").WithCause(err.Error())
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }

	if _, err := temp.Write(payload); err != nil {
		_ = temp.Close()
		cleanup()
		return apperrors.New(apperrors.E5009, "atomic write failed").WithCause(err.Error())
	}
	if mode != 0 {
		if err := temp.Chmod(mode); err != nil {
			_ = temp.Close()
			cleanup()
			return apperrors.New(apperrors.E5009, "atomic write chmod failed").WithCause(err.Error())
		}
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		cleanup()
		return apperrors.New(apperrors.E5009, "atomic write sync failed").WithCause(err.Error())
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return apperrors.New(apperrors.E5009, "atomic write close failed").WithCause(err.Error())
	}
	if err := platform.RenamePath(tempPath, path); err != nil {
		cleanup()
		return apperrors.New(apperrors.E5009, "atomic replace failed").WithCause(err.Error())
	}
	return nil
}

// WriteLockedFile writes payload under the existing sidecar lock (<path>.lock)
// using lockedfile.Write. It is a cross-process locked overwrite, NOT a
// temp+rename atomic replacement; the two semantics are not interchangeable.
// mode is passed through to lockedfile.Write.
//
// The underlying error is returned verbatim (no typed wrapper) so callers
// keep the exact error chain and cause text they had before migration; the
// caller maps it to its own domain code (e.g. journal -> E6005).
func WriteLockedFile(path string, payload []byte, mode os.FileMode) error {
	return lockedfile.Write(path, bytes.NewReader(payload), mode)
}

// reservation records the identity of a reserved resource and caches the
// first release result. Both fsatomic.ReserveFile and platform.ReserveLink
// keep their own package-private copy of this state (Go does not allow
// sharing an unexported type across packages, and adding a fourth
// infrastructure package is explicitly out of scope).
type reservation struct {
	mu       sync.Mutex
	path     string
	info     os.FileInfo
	file     *os.File // nil for link reservations; closed by releaseOnce
	resolved bool
	result   error
}

// releaseOnce executes fn at most once and caches its result. Repeated calls
// return the SAME error (including a first-call failure); there is no silent
// success on retry and no automatic retry state machine.
func (r *reservation) releaseOnce(fn func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.resolved {
		return r.result
	}
	// Close the reserved handle first: on Windows an open handle makes the
	// following os.Remove fail with a sharing violation. A caller that already
	// closed it gets os.ErrClosed, which is benign here.
	var closeErr error
	if r.file != nil {
		if err := r.file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			closeErr = err
		}
	}
	r.result = errors.Join(closeErr, fn())
	r.resolved = true
	return r.result
}

// owns returns true when path still refers to the same file identity that was
// reserved. It is only meaningful in a trusted, isolated directory: Lstat +
// SameFile + Remove has a TOCTOU window that cannot be closed by this check
// alone in a directory writable by untrusted processes.
func (r *reservation) owns(path string, info os.FileInfo) bool {
	if info == nil {
		return false
	}
	return os.SameFile(r.info, info)
}
