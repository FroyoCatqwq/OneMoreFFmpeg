//go:build windows

package logging

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
	"golang.org/x/sys/windows"
)

// supportsColor reports whether the console can render ANSI escapes. Windows
// terminals need ENABLE_VIRTUAL_TERMINAL_PROCESSING; emitting escapes without
// it prints garbage, so auto mode stays colorless when enabling fails.
func supportsColor(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()
	if !isatty.IsTerminal(fd) && !isatty.IsCygwinTerminal(fd) {
		return false
	}
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING) == nil
}
