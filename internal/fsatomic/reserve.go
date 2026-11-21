package fsatomic

import (
	"os"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// ReserveFile atomically creates a uniquely named placeholder file and
// returns the still-open handle plus a release function. The caller writes to
// the returned handle directly (no close/reopen TOCTOU window).
//
// PRECONDITION: dir must be a trusted, isolated directory (0700 / private
// data-root). In a directory writable by untrusted processes, Lstat +
// SameFile + Remove has a TOCTOU window; this function does not claim absolute
// protection there.
//
// Lifecycle:
//  1. Create with O_CREATE|O_EXCL|O_RDWR via os.CreateTemp semantics.
//  2. Record os.FileInfo as the identity of this reservation.
//  3. Caller writes, then SHOULD f.Sync() and may f.Close(), checking errors.
//     release() owns the handle and closes it if still open (so a caller that
//     forgets to Close cannot cause a Windows sharing violation).
//  4. release() verifies identity via Lstat + os.SameFile:
//     - path gone -> nil (caller renamed/moved it = committed);
//     - identity changed -> E5009 ownership_changed, DO NOT delete;
//     - identity matches -> remove; failure -> E5009 cleanup_failed.
//  5. release is idempotent: repeated calls return the cached first result,
//     including a first failure. There is no silent success and no retry.
//
// On failure paths the caller calls release(); any close error is joined with
// the release error and reported together.
func ReserveFile(dir, pattern string) (path string, f *os.File, release func() error, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, nil, apperrors.New(apperrors.E5009, "reservation directory cannot be created").WithCause(err.Error())
	}
	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, nil, apperrors.New(apperrors.E5009, "reservation cannot be created").WithCause(err.Error())
	}
	path = tmp.Name()
	info, err := tmp.Stat()
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return "", nil, nil, apperrors.New(apperrors.E5009, "reservation identity cannot be read").WithCause(err.Error())
	}
	res := &reservation{path: path, info: info, file: tmp}

	release = func() error {
		return res.releaseOnce(func() error {
			current, statErr := os.Lstat(path)
			if os.IsNotExist(statErr) {
				return nil // resource was committed elsewhere
			}
			if statErr != nil {
				return apperrors.New(apperrors.E5009, "reservation cannot be inspected").WithCause(statErr.Error()).
					WithDetails(map[string]any{"reason": "stat_failed", "path": path})
			}
			if !res.owns(path, current) {
				return apperrors.New(apperrors.E5009, "reserved path now owns a different resource").
					WithDetails(map[string]any{"reason": "ownership_changed", "path": path})
			}
			if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
				return apperrors.New(apperrors.E5009, "reservation cleanup failed").WithCause(removeErr.Error()).
					WithDetails(map[string]any{"reason": "cleanup_failed", "path": path})
			}
			return nil
		})
	}
	return path, tmp, release, nil
}
