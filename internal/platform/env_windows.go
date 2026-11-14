//go:build windows

package platform

import (
	"golang.org/x/sys/windows/registry"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const systemEnvKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`

type registryEnvStore struct {
	root registry.Key
	key  string
}

// NewEnvStore returns the HKCU\Environment (user) store.
func NewEnvStore() EnvStore { return registryEnvStore{registry.CURRENT_USER, `Environment`} }

// NewSystemEnvStore returns the HKLM system environment store; writing it
// requires an elevated (Administrator) process.
func NewSystemEnvStore() EnvStore { return registryEnvStore{registry.LOCAL_MACHINE, systemEnvKey} }

func (s registryEnvStore) Get(name string) (string, bool, error) {
	key, err := registry.OpenKey(s.root, s.key, registry.QUERY_VALUE)
	if err != nil {
		return "", false, apperrors.New(apperrors.E6001, "environment cannot be opened").WithCause(err.Error())
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err == registry.ErrNotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, apperrors.New(apperrors.E6001, "environment value cannot be read").WithCause(err.Error())
	}
	return value, true, nil
}

func (s registryEnvStore) Set(name, value string) error {
	key, _, err := registry.CreateKey(s.root, s.key, registry.SET_VALUE)
	if err != nil {
		return apperrors.New(apperrors.E6002, "environment cannot be opened for writing").WithCause(err.Error())
	}
	defer key.Close()
	_, valueType, err := key.GetValue(name, nil)
	if err == registry.ErrNotExist {
		valueType = registry.SZ
	}
	if valueType == registry.EXPAND_SZ {
		err = key.SetExpandStringValue(name, value)
	} else {
		err = key.SetStringValue(name, value)
	}
	if err != nil {
		return apperrors.New(apperrors.E6002, "environment value cannot be written").WithCause(err.Error())
	}
	return nil
}

func (s registryEnvStore) Remove(name string) error {
	key, err := registry.OpenKey(s.root, s.key, registry.SET_VALUE)
	if err != nil {
		return apperrors.New(apperrors.E6002, "environment cannot be opened for writing").WithCause(err.Error())
	}
	defer key.Close()
	if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return apperrors.New(apperrors.E6002, "environment value cannot be removed").WithCause(err.Error())
	}
	return nil
}