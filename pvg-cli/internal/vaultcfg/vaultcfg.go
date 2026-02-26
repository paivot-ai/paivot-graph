// Package vaultcfg provides shared vault configuration for pvg commands.
package vaultcfg

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/vlt"
)

const VaultName = "Claude"

// VaultDir returns the vault directory path, opening it via vlt.
func VaultDir() (string, error) {
	v, err := vlt.OpenByName(VaultName)
	if err != nil {
		// Fallback to conventional iCloud path
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("cannot determine vault directory: %w", err)
		}
		dir := filepath.Join(home, "Library", "Mobile Documents", "iCloud~md~obsidian", "Documents", VaultName)
		if _, serr := os.Stat(dir); serr != nil {
			return "", fmt.Errorf("vault not found via vlt or at %s", dir)
		}
		return dir, nil
	}
	return v.Dir(), nil
}

// OpenVault opens the Claude vault via vlt.
func OpenVault() (*vlt.Vault, error) {
	return vlt.OpenByName(VaultName)
}
