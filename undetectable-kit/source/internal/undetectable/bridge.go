package undetectable

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// This Chromium's PathService resolves Windows Local AppData independently of
// its user-data-dir. Only an engine-owned startInfo is staged at that root.
func (s *Spec) bridgeProfile() (string, string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}
	settingsPath := filepath.Join(base, "Undetectable", "settings.json")
	root := filepath.Join(base, "Undetectable", "Profiles")
	b, err := os.ReadFile(settingsPath)
	if err == nil {
		var cfg struct {
			ProfilesPath string `json:"profilesPath"`
		}
		if err = json.Unmarshal(b, &cfg); err != nil {
			return "", "", fmt.Errorf("browser settings %s: %w", settingsPath, err)
		}
		if cfg.ProfilesPath != "" {
			root = cfg.ProfilesPath
		}
	} else if os.IsNotExist(err) {
		// Initialise only absent settings for a machine without Undetectable.
		if err = os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
			return "", "", err
		}
		f, e := os.OpenFile(settingsPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return "", "", e
		}
		e = json.NewEncoder(f).Encode(map[string]any{"profilesPath": filepath.ToSlash(root) + "/"})
		f.Close()
		if e != nil {
			return "", "", e
		}
	} else {
		return "", "", err
	}
	if !filepath.IsAbs(root) {
		return "", "", fmt.Errorf("browser profilesPath must be absolute: %s", root)
	}
	h := sha256.Sum256([]byte(s.ProfilesRoot + "\x00" + s.ProfileID))
	nativeID := "ghost_" + hex.EncodeToString(h[:14])
	dir := filepath.Join(root, nativeID)
	owner := filepath.Join(s.ProfilesRoot, s.ProfileID)
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", "", err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		if !os.IsExist(err) {
			return "", "", err
		}
		marker, e := os.ReadFile(filepath.Join(dir, "ghost-owner.txt"))
		if e != nil || string(marker) != owner {
			return "", "", fmt.Errorf("refusing to overwrite non-GhostBrowser profile: %s", dir)
		}
	} else if err = os.WriteFile(filepath.Join(dir, "ghost-owner.txt"), []byte(owner), 0600); err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(filepath.Join(owner, "startInfo"))
	if err != nil {
		return "", "", err
	}
	if err = os.WriteFile(filepath.Join(dir, "startInfo"), data, 0600); err != nil {
		return "", "", err
	}
	return nativeID, dir, nil
}

// Remove just the two files the bridge created; never recursively remove a
// vendor profile directory or anything Chromium/user tooling added to it.
func cleanupBridge(dir string) {
	if _, err := os.ReadFile(filepath.Join(dir, "ghost-owner.txt")); err != nil {
		return
	}
	_ = os.Remove(filepath.Join(dir, "startInfo"))
	_ = os.Remove(filepath.Join(dir, "ghost-owner.txt"))
	_ = os.Remove(dir)
}
