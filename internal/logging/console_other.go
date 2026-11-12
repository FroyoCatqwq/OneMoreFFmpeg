//go:build !windows

package logging

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

func supportsColor(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd())
}
