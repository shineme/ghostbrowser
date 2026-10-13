package undetectable

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Spec is the standalone JSON contract. Paths are relative to the JSON file.
// ConfigMeta is retained in full; only recovered startInfo fields are rewritten.
type Spec struct {
	SchemaVersion int                        `json:"schemaVersion"`
	ProfileID     string                     `json:"profileId"`
	Name          string                     `json:"name"`
	ChromePath    string                     `json:"chromePath"`
	ProfilesRoot  string                     `json:"profilesRoot"`
	TemplatePath  string                     `json:"templatePath"`
	Data          map[string]json.RawMessage `json:"data"`
	ConfigMeta    map[string]json.RawMessage `json:"configMeta"`
	Runtime       RuntimeValues              `json:"runtime"`
	StartURLs     []string                   `json:"startUrls"`
	Proxy         string                     `json:"proxy,omitempty"`
	ExtraArgs     []string                   `json:"extraArgs,omitempty"`
	Headless      bool                       `json:"headless,omitempty"`
}

type RuntimeValues struct {
	Platform    string  `json:"platform,omitempty"`
	PixelRatio  float64 `json:"pixelRatio,omitempty"`
	ColorDepth  int64   `json:"colorDepth,omitempty"`
	WebGLVendor string  `json:"webglVendor,omitempty"`
}

func LoadSpec(path string) (*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 10<<20 {
		return nil, errors.New("profile JSON exceeds 10 MiB")
	}
	var s Spec
	if err = json.Unmarshal(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf}), &s); err != nil {
		return nil, fmt.Errorf("profile JSON: %w", err)
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	for _, p := range []*string{&s.ChromePath, &s.ProfilesRoot, &s.TemplatePath} {
		if *p != "" && !filepath.IsAbs(*p) {
			*p = filepath.Join(base, *p)
		}
	}
	return &s, s.Validate()
}

var safeID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func (s *Spec) Validate() error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion %d; use 1", s.SchemaVersion)
	}
	if !safeID.MatchString(s.ProfileID) {
		return errors.New("profileId must contain 1-64 letters, digits, '_' or '-'")
	}
	if s.Name == "" {
		s.Name = s.ProfileID
	}
	if s.ChromePath == "" || s.ProfilesRoot == "" || s.TemplatePath == "" {
		return errors.New("chromePath, profilesRoot and templatePath are required")
	}
	if len(s.Data) == 0 || len(s.ConfigMeta) == 0 {
		return errors.New("data and configMeta must be non-empty objects")
	}
	c, err := s.Config()
	if err != nil {
		return err
	}
	e := c.Expected()
	if !reRes.MatchString(c.Data.Str("resConfig")) {
		return errors.New("resConfig must be WIDTHxHEIGHT")
	}
	if e.Platform != "Win32" {
		return errors.New("the verified Windows template requires platform Win32")
	}
	if !strings.HasPrefix(e.UA, "Mozilla/") || !strings.Contains(e.UA, "Chrome/") {
		return errors.New("UA must be a Chromium Mozilla user agent")
	}
	if e.UADPlatform != "Windows" {
		return errors.New("the bundled template is verified for Windows only")
	}
	if e.Cores < 1 || e.Cores > 256 || e.MemoryGB < 1 || e.MemoryGB > 1024 {
		return errors.New("CPUCores or memory is out of range")
	}
	if e.Width < 320 || e.Height < 200 || e.Width > 16384 || e.Height > 16384 {
		return errors.New("resConfig must be WIDTHxHEIGHT between 320x200 and 16384x16384")
	}
	if e.DPR < 0.25 || e.DPR > 8 {
		return errors.New("pixelRatio must be between 0.25 and 8")
	}
	if _, err = time.LoadLocation(e.Timezone); err != nil {
		return fmt.Errorf("tzId: %w", err)
	}
	if c.UAD == nil || len(c.UAD.Brands) == 0 || len(c.UAD.FullVersionList) == 0 {
		return errors.New("additional_info.userAgentDataObj must include brands and fullVersionList")
	}
	if c.UAD.UAFullVersion != "" && c.UAD.UAFullVersion != e.FullVersion {
		return errors.New("uaFullVersion differs from resolved Chrome version")
	}
	if c.UAD.UAFullVersion == "" {
		return errors.New("userAgentDataObj.uaFullVersion is required")
	}
	if reFullVer.FindString(e.UA) != e.FullVersion {
		return errors.New("UA Chrome version must equal userAgentDataObj.uaFullVersion")
	}
	for _, a := range s.ExtraArgs {
		if strings.HasPrefix(a, "--user-data-dir") || strings.HasPrefix(a, "--profile-id") || strings.HasPrefix(a, "--remote-debugging") {
			return fmt.Errorf("extraArgs cannot override %s", a)
		}
	}
	return nil
}

func (s *Spec) Config() (*Config, error) {
	c := &Config{Profile: &Profile{ID: s.ProfileID, Name: s.Name}, Data: &Data{Raw: s.Data}, Meta: s.ConfigMeta, Info: &AdditionalInfo{Raw: map[string]json.RawMessage{}}, FromDB: true}
	raw := s.ConfigMeta["additional_info"]
	if len(raw) > 0 {
		var str string
		if json.Unmarshal(raw, &str) == nil {
			raw = []byte(str)
		}
		if err := json.Unmarshal(raw, &c.Info.Raw); err != nil {
			return nil, fmt.Errorf("additional_info: %w", err)
		}
	}
	if raw := c.Info.Raw["userAgentDataObj"]; len(raw) > 0 {
		c.UAD = &UserAgentData{}
		if err := json.Unmarshal(raw, c.UAD); err != nil {
			return nil, err
		}
	}
	if raw := c.Info.Raw["gpuDataObj"]; len(raw) > 0 {
		var g struct {
			AdapterInfo GPUAdapterInfo `json:"adapterInfo"`
		}
		if err := json.Unmarshal(raw, &g); err != nil {
			return nil, err
		}
		c.GPU = &g.AdapterInfo
	}
	e := c.Expected()
	if s.Runtime.Platform != "" {
		e.Platform = s.Runtime.Platform
	}
	// Use the same plain language list for expected values and CDP. Database
	// snapshots may have a one-language `langs` but a two-language header.
	if e.LangHeader != "" {
		e.Languages = strings.Split(languageList(e.LangHeader), ",")
	}
	if len(e.Languages) == 1 && strings.Contains(e.Languages[0], "-") {
		e.Languages = append(e.Languages, strings.SplitN(e.Languages[0], "-", 2)[0])
	}
	e.LangHeader = strings.Join(e.Languages, ",")
	if s.Runtime.PixelRatio > 0 {
		e.DPR = s.Runtime.PixelRatio
	}
	if s.Runtime.ColorDepth > 0 {
		e.ColorDepth = s.Runtime.ColorDepth
		e.PixelDepth = e.ColorDepth
	}
	if s.Runtime.WebGLVendor != "" {
		e.WebGLVendor = s.Runtime.WebGLVendor
	}
	c.Override = e
	return c, nil
}

// Prepare preserves the binary template's unknown fields. Browser-root staging
// is performed separately by Start because the native reader uses PathService.
func (s *Spec) Prepare() (*Config, string, error) {
	if err := s.Validate(); err != nil {
		return nil, "", err
	}
	root, err := filepath.Abs(s.ProfilesRoot)
	if err != nil {
		return nil, "", err
	}
	s.ProfilesRoot = root
	c, err := s.Config()
	if err != nil {
		return nil, "", err
	}
	l, before, err := LoadStartInfo(s.TemplatePath)
	if err != nil {
		return nil, "", fmt.Errorf("template: %w", err)
	}
	if len(before) > 64<<10 {
		return nil, "", errors.New("template too large")
	}
	data, err := ApplyExpected(l, c, c.Expected())
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Join(root, s.ProfileID)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, "", err
	}
	if err = os.WriteFile(filepath.Join(dir, "startInfo"), data, 0600); err != nil {
		return nil, "", err
	}
	if err = writeJSON(filepath.Join(dir, "launch.json"), s); err != nil {
		return nil, "", err
	}
	return c, dir, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

type Session struct {
	Cmd          *exec.Cmd
	Port         int
	WSURL        string
	ProfileID    string
	Config       *Config
	Done         chan struct{}
	exitErr      error
	stopOverride func()
	once         sync.Once
	bridgeDir    string
	profileLock  *os.File
	relay        *Relay
}

func (s *Spec) Start(port int) (*Session, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(s.ProfilesRoot)
	if err != nil {
		return nil, err
	}
	s.ProfilesRoot = root
	lock, err := lockProfile(filepath.Join(root, s.ProfileID))
	if err != nil {
		return nil, err
	}
	releaseLock := true
	defer func() {
		if releaseLock {
			lock.Close()
		}
	}()
	c, _, err := s.Prepare()
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(s.ChromePath); err != nil {
		return nil, fmt.Errorf("Chromium runtime: %w", err)
	}
	if port == 0 {
		port, err = FreePort()
		if err != nil {
			return nil, err
		}
	} else {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return nil, fmt.Errorf("CDP port %d is in use: %w", port, err)
		}
		listener.Close()
	}
	nativeID, bridge, err := s.bridgeProfile()
	if err != nil {
		return nil, err
	}
	args := []string{"--profile-id=" + nativeID, "--user-data-dir=" + filepath.Join(s.ProfilesRoot, s.ProfileID), fmt.Sprintf("--remote-debugging-port=%d", port), "--remote-debugging-address=127.0.0.1", "--no-first-run", "--no-default-browser-check", "--enable-chrome-browser-cloud-management", "--enable-privacy-sandbox-ads-apis", "--lang=" + primaryLang(c.Expected().Languages)}
	var relay *Relay
	if s.Proxy != "" {
		proxyArg := s.Proxy
		if NeedsRelay(s.Proxy) {
			// Chromium can't take credentials on the command line: go through a local relay.
			if relay, err = StartRelay(s.Proxy); err != nil {
				return nil, fmt.Errorf("proxy relay: %w", err)
			}
			proxyArg = relay.URL()
		}
		args = append(args, "--proxy-server="+proxyArg)
	}
	if s.Headless {
		args = append(args, "--headless=new")
	}
	args = append(args, s.ExtraArgs...)
	// Configure a blank page first, then navigate after overrides are installed.
	args = append(args, "about:blank")
	cmd := exec.Command(s.ChromePath, args...)
	cmd.Dir = filepath.Dir(s.ChromePath)
	session := &Session{Cmd: cmd, Port: port, ProfileID: s.ProfileID, Config: c, Done: make(chan struct{}), bridgeDir: bridge, profileLock: lock, relay: relay}
	if err = cmd.Start(); err != nil {
		cleanupBridge(bridge)
		relay.Close()
		return nil, err
	}
	go func() { session.exitErr = cmd.Wait(); relay.Close(); close(session.Done) }()
	ver, err := WaitForCDP(port, 30*time.Second, func() (int, bool) {
		select {
		case <-session.Done:
			return cmd.ProcessState.ExitCode(), true
		default:
			return 0, false
		}
	})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("Chromium start: %w", err)
	}
	session.WSURL = ver.WebSocketDebuggerURL
	session.stopOverride, err = NewOverrider(c).Start(port)
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("CDP fingerprint: %w", err)
	}
	client, err := DialCDP(session.WSURL, 10*time.Second)
	if err != nil {
		session.Close()
		return nil, err
	}
	urls := s.StartURLs
	if len(urls) == 0 {
		urls = []string{"about:blank"}
	}
	targets, _ := ListTargets(port)
	for i, u := range urls {
		if i == 0 && len(targets) > 0 {
			page, e := PageTarget(port)
			if e != nil {
				err = e
				break
			}
			pc, e := DialCDP(page.WebSocketDebuggerURL, 10*time.Second)
			if e != nil {
				err = e
				break
			}
			err = pc.Call("Page.navigate", map[string]any{"url": u}, nil)
			pc.Close()
		} else {
			err = client.Call("Target.createTarget", map[string]any{"url": u}, nil)
		}
		if err != nil {
			break
		}
	}
	client.Close()
	if err != nil {
		session.Close()
		return nil, err
	}
	releaseLock = false
	return session, nil
}

func (s *Session) Close() {
	s.once.Do(func() {
		if s.stopOverride != nil {
			s.stopOverride()
		}
		if s.WSURL != "" {
			if c, err := DialCDP(s.WSURL, 3*time.Second); err == nil {
				_ = c.Call("Browser.close", nil, nil)
				c.Close()
				// Let Chromium flush cookies / storage before forcing it.
				select {
				case <-s.Done:
				case <-time.After(8 * time.Second):
				}
			}
		}
		if s.Cmd != nil && s.Cmd.Process != nil {
			_ = s.Cmd.Process.Kill()
		}
		if s.bridgeDir != "" {
			select {
			case <-s.Done:
			case <-time.After(5 * time.Second):
			}
			cleanupBridge(s.bridgeDir)
		}
		if s.profileLock != nil {
			s.profileLock.Close()
		}
		s.relay.Close()
	})
}

func (s *Session) Wait() error { <-s.Done; return s.exitErr }
