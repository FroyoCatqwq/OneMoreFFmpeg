//go:build windows

package platform

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// RenamePath renames a file or directory, retrying transient lock errors.
// Defender and search indexers briefly hold handles on freshly extracted
// directories, which surfaces as ERROR_ACCESS_DENIED or ERROR_SHARING_VIOLATION
// on an otherwise valid rename.
func RenamePath(oldPath, newPath string) error {
	const attempts = 5
	for attempt := 0; ; attempt++ {
		err := os.Rename(oldPath, newPath)
		if err == nil {
			return nil
		}
		var linkErr *os.LinkError
		if attempt >= attempts-1 || !errors.As(err, &linkErr) || !retryableRenameError(linkErr.Err) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
}

func retryableRenameError(err error) bool {
	var errno windows.Errno
	if errors.As(err, &errno) {
		return errno == windows.ERROR_ACCESS_DENIED || errno == windows.ERROR_SHARING_VIOLATION
	}
	return false
}