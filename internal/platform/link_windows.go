//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"strings"

	"golang.org/x/sys/windows"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	fsctlSetReparsePoint = 0x000900A4
	reparseTagMountPoint = 0xA0000003
)

// CreateLink creates a directory link. It prefers a junction (no privilege
// needed) and falls back to a directory symlink, which developer mode, sudo
// (Win11 24H2+) or gsudo permits.
func CreateLink(link, target string) error {
	if _, err := os.Stat(target); err != nil {
		return apperrors.New(apperrors.E5004, "link target does not exist").WithCause(err.Error())
	}
	if junctionErr := createJunction(link, target); junctionErr == nil {
		return nil
	}
	if err := os.Symlink(target, link); err != nil {
		return apperrors.New(apperrors.E5004, "link cannot be created").WithCause(err.Error())
	}
	return nil
}

func createJunction(link, target string) error {
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

	substitute := junctionSubstituteName(target)
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

// junctionSubstituteName maps a target to the NT object-manager path stored in
// the reparse buffer. UNC targets need the \??\UNC\ prefix instead of \??\.
func junctionSubstituteName(target string) string {
	// Windows accepts both \\server\share and //server/share as UNC paths.
	// Normalize before choosing the NT prefix, otherwise a forward-slash UNC
	// data root produces a junction to the invalid path \??\//server/share.
	normalized := strings.ReplaceAll(target, "/", `\`)
	if strings.HasPrefix(normalized, `\\`) {
		return `\??\UNC\` + strings.TrimPrefix(normalized, `\\`)
	}
	return `\??\` + normalized
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

// createOneLink creates a link at an exact (previously reserved) name.
// os.Symlink and createJunction both fail if the name already exists, so a
// racing process cannot steal the reserved name without detection.
func createOneLink(link, target string, kind LinkTargetKind) error {
	if kind == LinkTargetDir {
		// Windows directory links require an existing directory target; a
		// dangling directory symlink would behave as a FILE link under
		// os.Symlink and silently break later directory resolution.
		info, err := os.Stat(target)
		if err != nil {
			return apperrors.New(apperrors.E5009, "directory link target must exist").WithCause(err.Error()).
				WithDetails(map[string]any{"reason": "junction_target_missing", "target": target})
		}
		if !info.IsDir() {
			return apperrors.New(apperrors.E5009, "directory link target is not a directory").
				WithDetails(map[string]any{"reason": "link_type_mismatch", "target": target})
		}
		return CreateLink(link, target) // junction first, then directory symlink
	}
	// LinkTargetFile: reject an existing directory target so the link kind
	// cannot silently disagree with the caller's intent.
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return apperrors.New(apperrors.E5009, "file link target is a directory").
			WithDetails(map[string]any{"reason": "link_type_mismatch", "target": target})
	}
	if err := os.Symlink(target, link); err != nil {
		return apperrors.New(apperrors.E5004, "file symlink cannot be created").WithCause(err.Error())
	}
	return nil
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
