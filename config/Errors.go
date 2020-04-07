package config

import (
	"errors"
)

var (
	// ErrUnlockingKeyring error if keyring is available but can't be unlocked
	ErrUnlockingKeyring = errors.New("Error unlocking keyring")
)
