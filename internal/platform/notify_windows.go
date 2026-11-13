//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	hwndBroadcast   = 0xFFFF
	wmSettingChange = 0x001A
	smtoAbortIfHung = 0x0002
)

var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procSendMessageW = user32.NewProc("SendMessageTimeoutW")
)

// NotifyEnvChange asks running shells/Explorer to reload the user environment.
func NotifyEnvChange() error {
	parameter, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return apperrors.New(apperrors.E6008, "environment notification payload is invalid").WithCause(err.Error())
	}
	var result uintptr
	code, _, callErr := procSendMessageW.Call(
		uintptr(hwndBroadcast), uintptr(wmSettingChange), 0,
		uintptr(unsafe.Pointer(parameter)), uintptr(smtoAbortIfHung), 3000,
		uintptr(unsafe.Pointer(&result)),
	)
	if code == 0 {
		cause := "SendMessageTimeoutW returned 0"
		if callErr != nil {
			cause = callErr.Error()
		}
		return apperrors.New(apperrors.E6008, "WM_SETTINGCHANGE broadcast failed").WithCause(cause)
	}
	return nil
}
