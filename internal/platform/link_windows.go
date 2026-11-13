//go:build windows

package platform

import (
	"encoding/binary"
	"os"

	"golang.org/x/sys/windows"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	fsctlSetReparsePoint = 0x000900A4
	reparseTagMountPoint = 0xA0000003
)

// CreateLink creates a directory junction. Junctions do not require the
// developer-mode symlink privilege.
func CreateLink(link, target string) error {
	if _, err := os.Stat(target); err != nil {
		return apperrors.New(apperrors.E5004, "junction target does not exist").WithCause(err.Error())
	}
	pointer, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return apperrors.New(apperrors.E5004, "junction path is invalid").WithCause(err.Error())
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		return apperrors.New(apperrors.E5004, "junction directory cannot be created").WithCause(err.Error())
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(link)
		}
	}()
	handle, err := windows.CreateFile(pointer, windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return apperrors.New(apperrors.E5004, "junction directory cannot be opened").WithCause(err.Error())
	}
	defer windows.CloseHandle(handle)

	substitute := `\??\` + target
	printName := target
	buffer := buildMountPointBuffer(substitute, printName)
	var returned uint32
	if err := windows.DeviceIoControl(handle, fsctlSetReparsePoint,
		&buffer[0], uint32(len(buffer)), nil, 0, &returned, nil); err != nil {
		return apperrors.New(apperrors.E5004, "junction reparse point cannot be written").WithCause(err.Error())
	}
	cleanup = false
	return nil
}

func buildMountPointBuffer(substitute, printName string) []byte {
	subUTF16 := windows.StringToUTF16(substitute)
	printUTF16 := windows.StringToUTF16(printName)
	subLength := uint16((len(subUTF16) - 1) * 2)
	printLength := uint16((len(printUTF16) - 1) * 2)
	pathBytes := (len(subUTF16) + len(printUTF16)) * 2
	buffer := make([]byte, 16+pathBytes)
	binary.LittleEndian.PutUint32(buffer[0:], reparseTagMountPoint)
	binary.LittleEndian.PutUint16(buffer[4:], uint16(8+pathBytes))
	binary.LittleEndian.PutUint16(buffer[6:], 0)
	binary.LittleEndian.PutUint16(buffer[8:], 0)
	binary.LittleEndian.PutUint16(buffer[10:], subLength)
	binary.LittleEndian.PutUint16(buffer[12:], subLength)
	binary.LittleEndian.PutUint16(buffer[14:], printLength)
	offset := 16
	for _, value := range subUTF16 {
		binary.LittleEndian.PutUint16(buffer[offset:], value)
		offset += 2
	}
	for _, value := range printUTF16 {
		binary.LittleEndian.PutUint16(buffer[offset:], value)
		offset += 2
	}
	return buffer
}

// RemoveLink removes a link without following it.
func RemoveLink(link string) error {
	info, err := os.Lstat(link)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperrors.New(apperrors.E5004, "link cannot be inspected").WithCause(err.Error())
	}
	if info.IsDir() {
		return os.Remove(link)
	}
	return os.Remove(link)
}
