//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	dbShell32          = syscall.NewLazyDLL("shell32.dll")
	procSHChangeNotify = dbShell32.NewProc("SHChangeNotify")
	dbUser32           = syscall.NewLazyDLL("user32.dll")
	procAllowSetFg     = dbUser32.NewProc("AllowSetForegroundWindow")
)

const (
	shcneAssocChanged = 0x08000000
	shcnfIDList       = 0x0000
	asfwAny           = 0xFFFFFFFF
)

func defaultBrowserSupported() bool { return true }

func assocChanged() { _, _, _ = procSHChangeNotify.Call(shcneAssocChanged, shcnfIDList, 0, 0) }

// allowForegroundHandoff: the process Windows starts for a link click may
// take the foreground; let the running instance / the browser window take it
// over (the second instance exits right after forwarding the link).
func allowForegroundHandoff() { _, _, _ = procAllowSetFg.Call(uintptr(asfwAny)) }

func registerDefaultBrowser(exe string) error {
	for _, k := range defaultBrowserKeys(exe) {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, k.Path, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("%s: %w", k.Path, err)
		}
		for name, v := range k.Values {
			if err = key.SetStringValue(name, v); err != nil {
				key.Close()
				return fmt.Errorf("%s\\%s: %w", k.Path, name, err)
			}
		}
		key.Close()
	}
	assocChanged()
	return nil
}

func deleteTree(root registry.Key, path string) error {
	k, err := registry.OpenKey(root, path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) || errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	if err != nil {
		return err
	}
	subs, _ := k.ReadSubKeyNames(-1)
	k.Close()
	for _, s := range subs {
		if err := deleteTree(root, path+`\`+s); err != nil {
			return err
		}
	}
	err = registry.DeleteKey(root, path)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return err
}

func unregisterDefaultBrowser() error {
	var first error
	for _, t := range defaultBrowserTrees() {
		if err := deleteTree(registry.CURRENT_USER, t); err != nil && first == nil {
			first = err
		}
	}
	for path, name := range defaultBrowserSharedValues() {
		k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
		if err != nil {
			continue
		}
		if err = k.DeleteValue(name); err != nil && !errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) && first == nil {
			first = err
		}
		k.Close()
	}
	assocChanged()
	return first
}

func readString(path, name string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue(name)
	return v
}

func isRegistered() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, dbRegAppsKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(dbAppName)
	return err == nil
}

func defaultBrowserState() (registered, isDefault bool, err error) {
	registered = isRegistered()
	isDefault = isSystemDefaultChoice(
		readString(fmt.Sprintf(dbUserChoiceFm, "http"), "ProgId"),
		readString(fmt.Sprintf(dbUserChoiceFm, "https"), "ProgId"))
	return registered, isDefault, nil
}

// ensureDefaultBrowserRegistration re-registers silently when GhostBrowser is
// registered but the EXE moved (portable build).
func ensureDefaultBrowserRegistration() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if p, e := filepath.EvalSymlinks(exe); e == nil {
		exe = p
	}
	cur := readString(dbClassesKey+`\`+dbURLProgID+`\shell\open\command`, "")
	if !needsReRegister(isRegistered(), cur, exe) {
		return ""
	}
	if err := registerDefaultBrowser(exe); err != nil {
		return "重新注册失败：" + err.Error()
	}
	return "EXE 位置变化，已重新注册"
}

func shellOpen(target string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, t, nil, nil, windows.SW_SHOWNORMAL)
}

// openDefaultAppsSettings: Windows 11 jumps straight to GhostBrowser's page;
// older builds ignore the parameter or fail, then the plain page opens.
func openDefaultAppsSettings() error {
	if err := shellOpen("ms-settings:defaultapps?registeredAppUser=" + dbAppName); err == nil {
		return nil
	}
	return shellOpen("ms-settings:defaultapps")
}
