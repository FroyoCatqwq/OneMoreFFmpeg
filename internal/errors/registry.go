// Package errors provides the stable OneMoreFFmpeg error and warning registry.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

type Code string

const (
	E1001 Code = "OMF-E1001"
	E1002 Code = "OMF-E1002"
	E1003 Code = "OMF-E1003"
	E1004 Code = "OMF-E1004"
	E1005 Code = "OMF-E1005"
	E1006 Code = "OMF-E1006"
	E1007 Code = "OMF-E1007"
	E1008 Code = "OMF-E1008"
	E1009 Code = "OMF-E1009"
	E1010 Code = "OMF-E1010"
	E1011 Code = "OMF-E1011"
	E1012 Code = "OMF-E1012"

	E2001 Code = "OMF-E2001"
	E2002 Code = "OMF-E2002"
	E2003 Code = "OMF-E2003"
	E2004 Code = "OMF-E2004"
	E2005 Code = "OMF-E2005"
	E2006 Code = "OMF-E2006"

	E3001 Code = "OMF-E3001"
	E3002 Code = "OMF-E3002"
	E3003 Code = "OMF-E3003"
	E3004 Code = "OMF-E3004"
	E3005 Code = "OMF-E3005"
	E3006 Code = "OMF-E3006"
	E3007 Code = "OMF-E3007"
	E3008 Code = "OMF-E3008"
	E3009 Code = "OMF-E3009"
	E3010 Code = "OMF-E3010"

	E4001 Code = "OMF-E4001"
	E4002 Code = "OMF-E4002"
	E4003 Code = "OMF-E4003"
	E4004 Code = "OMF-E4004"
	E4005 Code = "OMF-E4005"
	E4006 Code = "OMF-E4006"

	E5001 Code = "OMF-E5001"
	E5002 Code = "OMF-E5002"
	E5003 Code = "OMF-E5003"
	E5004 Code = "OMF-E5004"
	E5005 Code = "OMF-E5005"
	E5006 Code = "OMF-E5006"
	E5007 Code = "OMF-E5007"
	E5008 Code = "OMF-E5008"

	E6001 Code = "OMF-E6001"
	E6002 Code = "OMF-E6002"
	E6003 Code = "OMF-E6003"
	E6004 Code = "OMF-E6004"
	E6005 Code = "OMF-E6005"
	E6006 Code = "OMF-E6006"
	E6007 Code = "OMF-E6007"
	E6008 Code = "OMF-E6008"
	E6009 Code = "OMF-E6009"

	E7001 Code = "OMF-E7001"
	E7002 Code = "OMF-E7002"
	E7003 Code = "OMF-E7003"
	E7004 Code = "OMF-E7004"
	E7005 Code = "OMF-E7005"
	E7006 Code = "OMF-E7006"
	E7007 Code = "OMF-E7007"

	E9001 Code = "OMF-E9001"
	E9002 Code = "OMF-E9002"
	E9003 Code = "OMF-E9003"
	E9004 Code = "OMF-E9004"

	W1001 Code = "OMF-W1001"
	W1002 Code = "OMF-W1002"
	W1003 Code = "OMF-W1003"
	W1004 Code = "OMF-W1004"
	W1005 Code = "OMF-W1005"
	W1006 Code = "OMF-W1006"
	W1007 Code = "OMF-W1007"
)

type Entry struct {
	Name        string
	Description string
	Exit        int
	Warning     bool
}

var codeInfo = map[Code]struct {
	Name        string
	Description string
}{
	E1001: {"DIR_MISSING", "explicit directory does not exist"},
	E1002: {"BINARY_MISSING", "ffmpeg/ffprobe/ffplay missing"},
	E1003: {"VERSION_MISMATCH", "trio versions do not match"},
	E1004: {"INSTALLATION_UNUSABLE", "installation cannot be used"},
	E1005: {"PERMISSION_DENIED", "permission denied"},
	E1006: {"PROCESS_TIMEOUT", "process timeout"},
	E1007: {"PROCESS_FAILED", "process exited non-zero"},
	E1008: {"SCAN_LIMIT_EXCEEDED", "scan limit exceeded"},
	E1009: {"SCAN_ROOT_INVALID", "scan root invalid or unreadable"},
	E1010: {"SYMLINK_LOOP", "symlink loop detected"},
	E1011: {"CANCELED", "user canceled"},
	E1012: {"EXEC_FAILED", "process could not be started"},

	E2001: {"CAPABILITY_COMMAND_FAILED", "capability command failed"},
	E2002: {"CAPABILITY_PARSE_FAILED", "capability output cannot be parsed"},
	E2003: {"CAPABILITY_CACHE_READ", "capability cache read failed"},
	E2004: {"CAPABILITY_CACHE_WRITE", "capability cache write failed"},
	E2005: {"HELP_TOPIC_REJECTED", "help topic not allowed"},
	E2006: {"OUTPUT_LIMIT_EXCEEDED", "stdout limit exceeded"},

	E3001: {"PROVIDER_UNKNOWN", "provider id not registered"},
	E3002: {"RELEASE_NOT_FOUND", "release or tag not found"},
	E3003: {"ASSET_NOT_FOUND", "matching asset not found"},
	E3004: {"CHANNEL_UNSUPPORTED", "provider does not support channel"},
	E3005: {"CHECKSUMS_FILE_MISSING", "trusted checksums file missing"},
	E3006: {"GITHUB_RATE_LIMITED", "GitHub API rate limited"},
	E3007: {"GITHUB_API_FAILED", "GitHub API failed"},
	E3008: {"CUSTOM_URL_INVALID", "custom URL invalid or not https"},
	E3009: {"CUSTOM_SHA256_REQUIRED", "custom direct asset requires SHA-256"},
	E3010: {"RELEASE_NOT_PINNED", "release result still uses latest"},

	E4001: {"DOWNLOAD_FAILED", "download failed"},
	E4002: {"CHECKSUM_MISMATCH", "asset SHA-256 mismatch"},
	E4003: {"REDIRECT_LIMIT", "redirect limit exceeded"},
	E4004: {"DOWNLOAD_SIZE_LIMIT", "download size limit exceeded"},
	E4005: {"TEMP_WRITE_FAILED", "temporary file write failed"},
	E4006: {"DOWNLOAD_CANCELED", "download canceled"},

	E5001: {"ARCHIVE_UNSUPPORTED", "archive format unsupported"},
	E5002: {"ARCHIVE_UNSAFE_PATH", "archive path is unsafe"},
	E5003: {"INSTALL_DIR_CONFLICT", "install directory conflict"},
	E5004: {"CURRENT_SWITCH_FAILED", "current pointer switch failed"},
	E5005: {"ROLLBACK_FAILED", "rollback failed"},
	E5006: {"EXECUTABLE_BUSY", "executable is busy"},
	E5007: {"QUARANTINE_FAILED", "quarantine failed"},
	E5008: {"INSTALL_VALIDATION_FAILED", "post-install validation failed"},

	E6001: {"ENV_READ_FAILED", "environment read failed"},
	E6002: {"ENV_WRITE_FAILED", "environment write failed"},
	E6003: {"SYSTEM_SCOPE_DENIED", "system scope denied or not elevated"},
	E6004: {"SHELL_RC_DENIED", "shell rc not allowed or backup failed"},
	E6005: {"JOURNAL_FAILED", "journal operation failed"},
	E6006: {"DOCTOR_UNHEALTHY", "doctor detected unrecoverable state"},
	E6007: {"NON_MANAGED_FILE", "file is not managed by OneMoreFFmpeg"},
	E6008: {"WM_SETTINGCHANGE_FAILED", "WM_SETTINGCHANGE broadcast failed"},
	E6009: {"TRANSACTION_LOCKED", "data root is locked by another transaction"},

	E7001: {"CLI_USAGE", "invalid command usage"},
	E7002: {"TOML_PARSE_FAILED", "TOML 1.1 syntax error"},
	E7003: {"SCHEMA_MISMATCH", "schema mismatch"},
	E7004: {"UNKNOWN_KEY", "unknown key rejected"},
	E7005: {"INVALID_VALUE", "invalid value"},
	E7006: {"JSON_OUTPUT_FAILED", "stdout JSON serialization failed"},
	E7007: {"CONFIG_PATH_INVALID", "config path invalid or unreadable"},

	E9001: {"INTERNAL_INVARIANT", "internal invariant failed"},
	E9002: {"INTERNAL_PANIC", "panic caught at boundary"},
	E9003: {"INTERNAL_UNEXPECTED", "unexpected internal error"},
	E9004: {"CONTEXT_CANCELED", "context canceled"},

	W1001: {"FFPLAY_MISSING", "ffplay missing"},
	W1002: {"CAPABILITY_PARSE_WARNING", "capability parse warning"},
	W1003: {"CACHE_STALE", "cache stale or unusable"},
	W1004: {"NETWORK_DRIVE_SKIPPED", "network drive skipped"},
	W1005: {"ENV_NOT_EFFECTIVE_CURRENT_PROCESS", "new PATH not effective in current process"},
	W1006: {"NONFREE_BINARY", "nonfree binary compliance warning"},
	W1007: {"CACHE_WRITE_FAILED", "cache write failed"},
}

func exitFor(code Code) int {
	switch code {
	case E1011, E4006, E9004:
		return 130
	case E2005, E3001, E3004, E3008, E3009, E6003, E6004, E7001, E7002, E7003, E7004, E7005, E7007:
		return 2
	case E7006:
		return 1
	}
	if strings.HasPrefix(string(code), "OMF-W") {
		return 0
	}
	return 1
}

func KnownCodes() map[Code]Entry {
	out := make(map[Code]Entry, len(codeInfo))
	for code, info := range codeInfo {
		out[code] = Entry{
			Name:        info.Name,
			Description: info.Description,
			Exit:        exitFor(code),
			Warning:     strings.HasPrefix(string(code), "OMF-W"),
		}
	}
	return out
}

func IsKnown(code string) bool {
	_, ok := codeInfo[Code(code)]
	return ok
}

func ExitCodeFor(code Code) int {
	if _, ok := codeInfo[code]; !ok {
		return 1
	}
	return exitFor(code)
}

type Error struct {
	Code    Code
	Message string
	Details map[string]any
	Cause   string
}

func New(code Code, message string) *Error {
	if _, ok := codeInfo[code]; !ok {
		return &Error{
			Code:    E9001,
			Message: "unknown error code",
			Details: map[string]any{"requested_code": string(code)},
		}
	}
	return &Error{Code: code, Message: message, Details: map[string]any{}}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) ExitCode() int {
	if e == nil {
		return 1
	}
	return ExitCodeFor(e.Code)
}

func (e *Error) WithDetails(details map[string]any) *Error {
	if e == nil {
		return nil
	}
	if details != nil {
		e.Details = details
	}
	return e
}

func (e *Error) WithCause(cause string) *Error {
	if e == nil {
		return nil
	}
	e.Cause = cause
	return e
}

func As(err error) (*Error, bool) {
	var target *Error
	ok := stderrors.As(err, &target)
	return target, ok
}

func From(err error) *Error {
	if err == nil {
		return New(E9003, "nil error")
	}
	if typed, ok := As(err); ok {
		return typed
	}
	return New(E9003, "unexpected internal error").WithCause(err.Error())
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if typed, ok := As(err); ok {
		return typed.ExitCode()
	}
	return 1
}
