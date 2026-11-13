//go:build windows

package platform

import (
	"golang.org/x/sys/windows/registry"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

type userEnvStore struct{}

// NewEnvStore returns the HKCU\Environment store.
func NewEnvStore() EnvStore { return userEnvStore{} }

func (userEnvStore) Get(name string) (string, bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", false, apperrors.New(apperrors.E6001, "user environment cannot be opened").WithCause(err.Error())
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err == registry.ErrNotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, apperrors.New(apperrors.E6001, "user environment value cannot be read").WithCause(err.Error())
	}
	return value, true, nil
}

func (userEnvStore) Set(name, value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return apperrors.New(apperrors.E6002, "user environment cannot be opened for writing").WithCause(err.Error())
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
		return apperrors.New(apperrors.E6002, "user environment value cannot be written").WithCause(err.Error())
	}
	return nil
}

func (userEnvStore) Remove(name string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
	if err != nil {
		return apperrors.New(apperrors.E6002, "user environment cannot be opened for writing").WithCause(err.Error())
	}
	defer key.Close()
	if err := key.DeleteValue(name); err != nil && err != registry.ErrNotExist {
		return apperrors.New(apperrors.E6002, "user environment value cannot be removed").WithCause(err.Error())
	}
	return nil
}
