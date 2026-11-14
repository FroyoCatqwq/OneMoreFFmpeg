//go:build !windows

package platform

import (
	"os"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

type processEnvStore struct{}

// NewEnvStore is read-only on non-Windows; Linux persists via shims instead.
func NewEnvStore() EnvStore { return processEnvStore{} }

// NewSystemEnvStore is read-only on non-Windows; Linux system scope persists
// via /usr/local/bin shims and /etc/profile.d instead.
func NewSystemEnvStore() EnvStore { return processEnvStore{} }

func (processEnvStore) Get(name string) (string, bool, error) {
	value, ok := os.LookupEnv(name)
	return value, ok, nil
}

func (processEnvStore) Set(string, string) error {
	return apperrors.New(apperrors.E6002, "this platform does not support persistent environment writes")
}

func (processEnvStore) Remove(string) error {
	return apperrors.New(apperrors.E6002, "this platform does not support persistent environment writes")
}
