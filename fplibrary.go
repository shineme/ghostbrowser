package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The user's own fingerprint library: custom values per OS and field, plus
// whole-machine presets saved from the editor. Stored next to data.json in
// library.json so it survives upgrades and can be backed up on its own.

var libraryFields = map[string]bool{
	"osVersions": true, "browserMajors": true, "userAgents": true, "platforms": true,
	"vendors": true, "gpus": true, "screens": true, "fonts": true, "cores": true, "memory": true,
}

type fpLibraryData struct {
	Values  map[string][]string `json:"values"` // key "<os>:<field>"; gpus are "vendor\x1frenderer"
	Presets []DevicePreset      `json:"presets"`
}

type fpLibrary struct {
	mu   sync.Mutex
	path string
	d    fpLibraryData
}

var userLib = &fpLibrary{}

func (l *fpLibrary) open(dir string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.path = filepath.Join(dir, "library.json")
	l.d = fpLibraryData{Values: map[string][]string{}}
	if b, e := os.ReadFile(l.path); e == nil {
		_ = json.Unmarshal(b, &l.d)
	}
	if l.d.Values == nil {
		l.d.Values = map[string][]string{}
	}
}

func (l *fpLibrary) saveLocked() error {
	if l.path == "" {
		return nil
	}
	b, _ := json.MarshalIndent(l.d, "", "  ")
	tmp := l.path + ".tmp"
	if e := os.WriteFile(tmp, b, 0o600); e != nil {
		return e
	}
	return os.Rename(tmp, l.path)
}

func libKey(osName, field string) (string, error) {
	if _, ok := osVersionsByOS[osName]; !ok {
		return "", fmt.Errorf("unknown OS %q", osName)
	}
	if !libraryFields[field] {
		return "", fmt.Errorf("unknown field %q", field)
	}
	return osName + ":" + field, nil
}

func (l *fpLibrary) values(osName, field string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.d.Values[osName+":"+field]...)
}

func (l *fpLibrary) Add(osName, field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value is empty")
	}
	k, e := libKey(osName, field)
	if e != nil {
		return e
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, v := range l.d.Values[k] {
		if v == value {
			return nil
		}
	}
	l.d.Values[k] = append([]string{value}, l.d.Values[k]...)
	return l.saveLocked()
}

func (l *fpLibrary) Remove(osName, field, value string) error {
	k, e := libKey(osName, field)
	if e != nil {
		return e
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.d.Values[k][:0]
	for _, v := range l.d.Values[k] {
		if v != value {
			out = append(out, v)
		}
	}
	l.d.Values[k] = out
	return l.saveLocked()
}

func (l *fpLibrary) presets() []DevicePreset {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]DevicePreset{}, l.d.Presets...)
}

func (l *fpLibrary) SavePreset(p DevicePreset) (DevicePreset, error) {
	if _, ok := osVersionsByOS[p.OS]; !ok {
		return p, fmt.Errorf("unknown OS %q", p.OS)
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		p.Name = fmt.Sprintf("我的预设 · %s · %dx%d", gpuShort(p.Renderer), p.Width, p.Height)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if p.ID == "" || !strings.HasPrefix(p.ID, "my-") {
		p.ID = fmt.Sprintf("my-%d", time.Now().UnixNano())
	}
	p.Custom = true
	for i, x := range l.d.Presets {
		if x.ID == p.ID {
			l.d.Presets[i] = p
			return p, l.saveLocked()
		}
	}
	l.d.Presets = append([]DevicePreset{p}, l.d.Presets...)
	return p, l.saveLocked()
}

func (l *fpLibrary) DeletePreset(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.d.Presets[:0]
	for _, x := range l.d.Presets {
		if x.ID != id {
			out = append(out, x)
		}
	}
	l.d.Presets = out
	return l.saveLocked()
}

// mergeFront puts the user's own values first, without duplicates.
func mergeFront(custom, base []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, xs := range [][]string{custom, base} {
		for _, v := range xs {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}
