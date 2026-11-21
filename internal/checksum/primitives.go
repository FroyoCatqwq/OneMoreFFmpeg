package checksum

import (
	"crypto/sha256"
	"encoding/hex"
)

// SumHex returns the lowercase hex-encoded SHA-256 of data.
// It is the single shared primitive for the small in-memory hashes used by
// detect / ffexec / capabilities (archive digests use StreamSHA256 instead).
func SumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// IsLowerHex64 reports whether s is exactly 64 lowercase hexadecimal chars.
// Length is part of the primitive so callers cannot forget to check it.
func IsLowerHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		digit := r >= '0' && r <= '9'
		lower := r >= 'a' && r <= 'f'
		if !digit && !lower {
			return false
		}
	}
	return true
}

// IsHex64 is IsLowerHex64 with uppercase hex accepted (for upstream digests).
func IsHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		digit := r >= '0' && r <= '9'
		lower := r >= 'a' && r <= 'f'
		upper := r >= 'A' && r <= 'F'
		if !digit && !lower && !upper {
			return false
		}
	}
	return true
}
