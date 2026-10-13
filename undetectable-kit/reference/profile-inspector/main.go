// profile-inspector reads the Undetectable profile database and startInfo
// fingerprint records, and can launch the bundled Chromium with a profile's
// fingerprint without starting the Undetectable main application.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const usage = `profile-inspector - inspect and launch Undetectable Chromium profiles

usage:
  profile-inspector list                         list profiles in the database
  profile-inspector dump   <profile-id>          show startInfo slots + DB fields
  profile-inspector config <profile-id>          show the resolved fingerprint
  profile-inspector sync   <profile-id>          rewrite startInfo from the database
  profile-inspector launch <profile-id>          start the fingerprint browser
  profile-inspector verify <profile-id>          launch, read back, compare

common flags:
  -port N          DevTools port (0 = pick a free port)
  -timeout 60s     DevTools startup timeout
  -chrome PATH     chrome.exe path
  -uddir PATH      user data dir (default: the profile directory)
  -proxy URL       --proxy-server value
  -url URL         page to open (default about:blank)
  -window-size WxH
  -maximized       start maximized
  -headless        run headless
  -no-cdp          do not apply UA / UA-CH overrides after startup
  -keep-open       wait for Enter before exiting (launch/verify)
`

func main() {
	// The subcommand and profile id may appear anywhere on the command line, so
	// that a GUI wrapper can use either `tool -flag v cmd id` or `tool cmd id
	// -flag v`. They are lifted out before the standard flag parser runs.
	os.Args = hoistPositionals(os.Args)
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	port := flag.Int("port", 0, "DevTools port (0 = free port)")
	timeout := flag.Duration("timeout", 60*time.Second, "DevTools startup timeout")
	chrome := flag.String("chrome", "", "chrome.exe path")
	uddir := flag.String("uddir", "", "user data dir")
	proxy := flag.String("proxy", "", "--proxy-server value")
	url := flag.String("url", "about:blank", "page to open")
	windowSize := flag.String("window-size", "", "window size WxH")
	maximized := flag.Bool("maximized", false, "start maximized")
	minimized := flag.Bool("minimized", false, "start minimized")
	headless := flag.Bool("headless", false, "run headless")
	noCDP := flag.Bool("no-cdp", false, "skip UA/UA-CH overrides")
	keepOpen := flag.Bool("keep-open", false, "wait for Enter before exiting")
	detach := flag.Bool("detach", false, "launch: return immediately instead of supervising")
	out := flag.String("out", "", "sync: output path (default startInfo in place)")
	logPath := flag.String("log", "", "write browser stdout/stderr to this file")
	force := flag.Bool("force", false, "sync: overwrite without a .bak backup")
	jsonOut := flag.Bool("json", false, "print machine readable JSON")
	flag.Parse()

	if *chrome != "" {
		os.Setenv("UNDETECTABLE_CHROME", *chrome)
	}
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := args[0]
	id := ""
	if len(args) > 1 {
		id = args[1]
	}

	var err error
	switch cmd {
	case "list":
		err = cmdList(*jsonOut)
	case "dump":
		err = cmdDump(id, *jsonOut)
	case "config":
		err = cmdConfig(id, *jsonOut)
	case "sync":
		err = cmdSync(id, *out, *force)
	case "launch":
		err = cmdLaunch(id, launchFlags{
			port: *port, timeout: *timeout, uddir: *uddir, proxy: *proxy, url: *url,
			windowSize: *windowSize, maximized: *maximized, minimized: *minimized,
			headless: *headless, noCDP: *noCDP, keepOpen: *keepOpen,
			detach: *detach, logPath: *logPath,
		})
	case "verify":
		err = cmdVerify(id, launchFlags{
			port: *port, timeout: *timeout, uddir: *uddir, proxy: *proxy, url: *url,
			windowSize: *windowSize, maximized: *maximized, minimized: *minimized,
			headless: *headless, noCDP: *noCDP, keepOpen: *keepOpen,
		}, *jsonOut)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// valueFlags are the options that consume the following argument.
var valueFlags = map[string]bool{
	"-port": true, "--port": true, "-timeout": true, "--timeout": true,
	"-chrome": true, "--chrome": true, "-uddir": true, "--uddir": true,
	"-proxy": true, "--proxy": true, "-url": true, "--url": true,
	"-window-size": true, "--window-size": true, "-out": true, "--out": true,
}

// hoistPositionals rewrites argv so that non-flag arguments are moved to the
// end. This keeps `cmd id -flag v` and `-flag v cmd id` equivalent.
func hoistPositionals(argv []string) []string {
	out := []string{argv[0]}
	var pos []string
	for i := 1; i < len(argv); i++ {
		a := argv[i]
		if strings.HasPrefix(a, "-") {
			out = append(out, a)
			// `-flag=value` carries its own value; otherwise consume the next
			// argument unless it is itself a flag.
			if !strings.Contains(a, "=") && valueFlags[a] && i+1 < len(argv) {
				i++
				out = append(out, argv[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(out, pos...)
}

func mustStore() (*Store, error) {
	s, err := openStore()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func cmdList(asJSON bool) error {
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	profiles, err := s.List()
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profiles)
	}
	fmt.Printf("profiles root: %s\n", s.Profiles)
	fmt.Printf("%-34s %-22s %-9s %-9s %-7s %s\n",
		"ID", "NAME", "STARTINFO", "MODIFIED", "DELETED", "LAYOUT")
	for _, p := range profiles {
		info := "-"
		layout := "-"
		if p.HasInfo {
			info = fmt.Sprintf("%d B", p.InfoSize)
			if l, _, err := LoadStartInfo(s.StartInfoPath(p.ID)); err == nil {
				layout = fmt.Sprintf("%s (%d slots)", layoutName(l), len(l.Slots))
			} else {
				layout = "unreadable"
			}
		}
		fmt.Printf("%-34s %-22s %-9s %-9s %-7d %s\n",
			p.ID, truncate(p.Name, 22), info, truncate(p.Modify, 9), p.IsDeleted, layout)
	}
	return nil
}

func layoutName(l *Layout) string {
	if l.IsB {
		return "B/UA-CH"
	}
	return "A/legacy"
}

func cmdDump(id string, asJSON bool) error {
	if id == "" {
		return fmt.Errorf("dump requires a profile id")
	}
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	cfg, err := s.Config(id)
	if err != nil {
		return err
	}
	l, raw, err := LoadStartInfo(s.StartInfoPath(id))
	if err != nil {
		return err
	}
	if asJSON {
		type slotJSON struct {
			Index int    `json:"index"`
			Kind  string `json:"kind"`
			Enc   string `json:"enc,omitempty"`
			Off   int    `json:"offset"`
			Size  int    `json:"size"`
			Text  string `json:"text,omitempty"`
			Hex   string `json:"hex,omitempty"`
		}
		out := struct {
			Profile string     `json:"profile"`
			Layout  string     `json:"layout"`
			Size    int        `json:"size"`
			Slots   []slotJSON `json:"slots"`
		}{Profile: id, Layout: layoutName(l), Size: len(raw)}
		for i, sl := range l.Slots {
			j := slotJSON{Index: i, Off: l.SlotOffset(i), Size: sl.Size()}
			switch sl.Kind {
			case KindRaw:
				j.Kind = "raw"
				j.Hex = hex.EncodeToString(sl.Data)
			case KindEmpty:
				j.Kind = "empty"
			default:
				j.Kind = "string"
				j.Enc = sl.Enc
				j.Text = sl.Text()
			}
			out.Slots = append(out.Slots, j)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("profile : %s (%s)\n", cfg.Profile.Name, id)
	fmt.Printf("layout  : %s\n", layoutName(l))
	fmt.Printf("file    : %s (%d bytes, %d slots)\n",
		s.StartInfoPath(id), len(raw), len(l.Slots))
	fmt.Println()
	printNamedFields(l, cfg)
	fmt.Println()
	fmt.Printf("%-4s %-7s %-8s %-6s %s\n", "#", "KIND", "OFFSET", "SIZE", "VALUE")
	for i, sl := range l.Slots {
		fmt.Printf("%-4d %-7s %-8d %-6d %s\n", i, kindName(sl), l.SlotOffset(i), sl.Size(), sl)
	}
	return nil
}

func kindName(s Slot) string {
	switch s.Kind {
	case KindRaw:
		return "RAW"
	case KindEmpty:
		return "EMPTY"
	default:
		return "STR"
	}
}

func printNamedFields(l *Layout, cfg *Config) {
	fields := []struct {
		name string
		idx  int
	}{
		{"name", l.Name}, {"osVers", l.OSVers}, {"coresRaw", l.CoresRaw},
		{"gpuArch", l.GPUArch}, {"gpuVendor", l.GPUVendor},
		{"ua1", l.UA1}, {"ua2", l.UA2}, {"platform", l.Platform},
		{"productSub", l.ProductSub}, {"vendor", l.Vendor},
		{"verShort", l.VerShort}, {"verFull", l.VerFull},
		{"uaPlatform", l.UAPlatform}, {"osVerFull", l.OSVerFull},
		{"arch", l.Arch}, {"bitness", l.Bitness},
		{"appVersion", l.AppVersion}, {"formFactor", l.FormFactor},
		{"brandCount", l.BrandCount}, {"fullCount", l.FullCount},
		{"publicRTC", l.PublicRTC}, {"webglRenderer", l.WebGLRender},
		{"webglVendor", l.WebGLVendor}, {"timezone", l.TZ},
		{"lang", l.Lang}, {"zhTW", l.ZhTW},
	}
	fmt.Printf("%-14s %-5s %-8s %s\n", "FIELD", "SLOT", "OFFSET", "VALUE")
	for _, f := range fields {
		if f.idx < 0 {
			fmt.Printf("%-14s %-5s %-8s %s\n", f.name, "-", "-", "(not in this layout)")
			continue
		}
		fmt.Printf("%-14s %-5d %-8d %s\n", f.name, f.idx, l.SlotOffset(f.idx),
			truncate(l.Slots[f.idx].String(), 90))
	}
	buf := EncodeSlots(l.Slots)
	if anchor, err := locateScalars(l.Slots, buf, l); err == nil {
		if sc, err := ReadScalars(buf, anchor); err == nil {
			fmt.Printf("%-14s %-5s %-8d packed CPU/memory record\n", "coresBase", "", anchor.CoresBase)
			fmt.Printf("%-14s %-5s %-8d screen metrics block\n", "screenBase", "", anchor.ScreenBase)
			fmt.Printf("%-14s %-5s %-8s %d GB / %d cores\n", "coresMemory", "", "", sc.MemoryGB, sc.Cores)
			fmt.Printf("%-14s %-5s %-8s %dx%d avail %dx%d dpr %v depth %d/%d\n",
				"screen", "", "", sc.Width, sc.Height, sc.AvailWidth, sc.AvailHeight,
				sc.DPR, sc.ColorDepth, sc.PixelDepth)
			fmt.Printf("%-14s %-5s %-8s %v\n", "canvasAcc", "", "", sc.Acc)
			fmt.Printf("%-14s %-5s %-8s %d\n", "storage", "", "", sc.Storage)
			fmt.Printf("%-14s %-5s %-8s %d\n", "heapLimit", "", "", sc.HeapLimit)
		}
	}
	fmt.Println()
	e := cfg.Expected()
	fmt.Printf("database expectation:\n")
	fmt.Printf("  UA            %s\n", e.UA)
	fmt.Printf("  cores/memory  %d / %d GB\n", e.Cores, e.MemoryGB)
	fmt.Printf("  timezone      %s\n", e.Timezone)
	fmt.Printf("  languages     %v\n", e.Languages)
	fmt.Printf("  screen        %dx%d dpr %v\n", e.Width, e.Height, e.DPR)
	fmt.Printf("  webgl         %s\n", e.WebGLRender)
	fmt.Printf("  publicRTC     %s\n", e.PublicRTC)
}

func cmdConfig(id string, asJSON bool) error {
	if id == "" {
		return fmt.Errorf("config requires a profile id")
	}
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	cfg, err := s.Config(id)
	if err != nil {
		return err
	}
	e := cfg.Expected()
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"id": cfg.Profile.ID, "name": cfg.Profile.Name,
			"expected": e, "userAgentData": cfg.UAD, "gpu": cfg.GPU,
		})
	}
	fmt.Printf("profile        %s (%s)\n", cfg.Profile.Name, id)
	fmt.Printf("user agent     %s\n", e.UA)
	fmt.Printf("full version   %s\n", e.FullVersion)
	fmt.Printf("platform       %s, UA-CH %s, Windows %s (%s)\n",
		e.Platform, e.UADPlatform, e.OSVersion, e.Arch+"/"+e.Bitness)
	fmt.Printf("cores/memory   %d / %d GB\n", e.Cores, e.MemoryGB)
	fmt.Printf("timezone       %s\n", e.Timezone)
	fmt.Printf("languages      %v\n", e.Languages)
	fmt.Printf("screen         %dx%d dpr %v depth %d/%d\n",
		e.Width, e.Height, e.DPR, e.ColorDepth, e.PixelDepth)
	fmt.Printf("webgl vendor   %s\n", e.WebGLVendor)
	fmt.Printf("webgl renderer %s\n", e.WebGLRender)
	fmt.Printf("public RTC     %s\n", e.PublicRTC)
	fmt.Printf("storage quota  %d\n", e.StorageQuota)
	fmt.Printf("heap limit     %d\n", e.HeapLimit)
	if cfg.UAD != nil {
		fmt.Printf("ua-ch brands   ")
		for i, b := range cfg.UAD.Brands {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s/%s", b.Brand, b.Version)
		}
		fmt.Println()
	}
	fmt.Println()
	fmt.Println("configMeta keys:")
	for _, k := range cfg.SortedMetaKeys() {
		fmt.Printf("  %-32s %d bytes\n", k, len(cfg.Meta[k]))
	}
	fmt.Println()
	fmt.Println("additional_info keys:")
	for _, k := range SortedKeys(cfg.Info.Raw) {
		fmt.Printf("  %-32s %d bytes\n", k, len(cfg.Info.Raw[k]))
	}
	return nil
}

// cmdSync rewrites startInfo from the database, using the existing file as a
// template so the unknown parts of the record stream stay untouched.
func cmdSync(id, out string, force bool) error {
	if id == "" {
		return fmt.Errorf("sync requires a profile id")
	}
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	cfg, err := s.Config(id)
	if err != nil {
		return err
	}
	path := s.StartInfoPath(id)
	l, before, err := LoadStartInfo(path)
	if err != nil {
		return err
	}
	e := cfg.Expected()
	after, err := ApplyExpected(l, cfg, e)
	if err != nil {
		return err
	}
	dest := out
	if dest == "" {
		dest = path
	}
	if dest == path && !force {
		bak := path + ".bak"
		if _, err := os.Stat(bak); os.IsNotExist(err) {
			if err := os.WriteFile(bak, before, 0o600); err != nil {
				return err
			}
			fmt.Printf("backup written to %s\n", bak)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(dest, after, 0o600); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes, was %d)\n", dest, len(after), len(before))
	printChanges(l, before, after)
	return nil
}

// ApplyExpected writes every database value into the slot model and returns the
// serialized startInfo buffer.
func ApplyExpected(l *Layout, cfg *Config, e *Expected) ([]byte, error) {
	// name
	l.SetSlot(l.Name, UTF16Slot(cfg.Profile.Name))
	// user agent (both copies)
	l.SetSlot(l.UA1, AsciiSlot(e.UA))
	l.SetSlot(l.UA2, AsciiSlot(e.UA))
	// platform / vendor block
	l.SetSlot(l.Platform, AsciiSlot(e.Platform))
	l.SetSlot(l.Vendor, AsciiSlot("Google Inc."))
	l.SetSlot(l.UAPlatform, AsciiSlot(e.UADPlatform))
	l.SetSlot(l.OSVerFull, AsciiSlot("19.0.0"))
	l.SetSlot(l.Arch, AsciiSlot(e.Arch))
	l.SetSlot(l.Bitness, AsciiSlot(e.Bitness))
	appVersion := "5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/" +
		e.FullVersion + " Safari/537.36"
	l.SetSlot(l.AppVersion, AsciiSlot(appVersion))
	l.SetSlot(l.FormFactor, AsciiSlot("Desktop"))
	if l.IsB {
		if l.VerShort >= 0 {
			l.SetSlot(l.VerShort, UTF16Slot(chromeMajor(e.FullVersion)+".0.0.0"))
		}
		if l.VerFull >= 0 {
			l.SetSlot(l.VerFull, AsciiSlot(e.FullVersion))
		}
	}
	// GPU strings
	if l.GPUArch >= 0 && cfg.GPU != nil {
		l.SetSlot(l.GPUArch, AsciiSlot(cfg.GPU.Architecture))
	}
	if l.GPUVendor >= 0 && cfg.GPU != nil {
		l.SetSlot(l.GPUVendor, AsciiSlot(cfg.GPU.Vendor))
	}
	if l.WebGLRender >= 0 && e.WebGLRender != "" {
		l.SetSlot(l.WebGLRender, AsciiSlot(e.WebGLRender))
	}
	if l.WebGLVendor >= 0 {
		l.SetSlot(l.WebGLVendor, AsciiSlot(e.WebGLVendor))
	}
	// network / locale
	if l.PublicRTC >= 0 && e.PublicRTC != "" {
		l.SetSlot(l.PublicRTC, AsciiSlot(e.PublicRTC))
	}
	if l.TZ >= 0 && e.Timezone != "" {
		l.SetSlot(l.TZ, AsciiSlot(e.Timezone))
	}
	if l.Lang >= 0 && len(e.Languages) > 0 {
		l.SetSlot(l.Lang, AsciiSlot(e.Languages[0]))
	}
	// UA-CH brand lists (layout B)
	if l.IsB && cfg.UAD != nil {
		if l.BrandCount >= 0 {
			l.SetSlot(l.BrandCount, U32Raw(uint32(len(cfg.UAD.Brands))))
			for i, b := range cfg.UAD.Brands {
				if l.BrandCount+1+i*2+1 < len(l.Slots) {
					l.SetSlot(l.BrandCount+1+i*2, AsciiSlot(b.Brand))
					l.SetSlot(l.BrandCount+1+i*2+1, AsciiSlot(b.Version))
				}
			}
		}
		if l.FullCount >= 0 {
			l.SetSlot(l.FullCount, U32Raw(uint32(len(cfg.UAD.FullVersionList))))
			for i, b := range cfg.UAD.FullVersionList {
				if l.FullCount+1+i*2+1 < len(l.Slots) {
					l.SetSlot(l.FullCount+1+i*2, AsciiSlot(b.Brand))
					l.SetSlot(l.FullCount+1+i*2+1, AsciiSlot(b.Version))
				}
			}
		}
	}
	// Packed scalar fields are patched after serialization, because the string
	// rewrites above change every offset that follows them.
	buf := EncodeSlots(l.Slots)
	anchor, err := locateScalars(l.Slots, buf, l)
	if err != nil {
		return nil, err
	}
	p1, p2, r0, r1 := cfg.CanvasParams()
	availH := e.Height - 48
	if availH <= 0 {
		availH = e.Height
	}
	vals := Scalars{
		MemoryGB: uint32(e.MemoryGB), Cores: uint32(e.Cores),
		CanvasRandom0: r0, CanvasRandom1: r1, Pixel1: p1, Pixel2: p2,
		Width: uint32(e.Width), Height: uint32(e.Height),
		AvailWidth: uint32(e.Width), AvailHeight: uint32(availH),
		DPR: e.DPR, ColorDepth: uint32(e.ColorDepth), PixelDepth: uint32(e.PixelDepth),
		Acc: cfg.Acc(), Storage: e.StorageQuota, HeapLimit: e.HeapLimit,
	}
	if err := PatchScalars(buf, anchor, vals); err != nil {
		return nil, err
	}
	return buf, nil
}

func chromeMajor(v string) string {
	if i := strings.IndexByte(v, '.'); i > 0 {
		return v[:i]
	}
	return v
}

func printChanges(l *Layout, before, after []byte) {
	oldSlots := ParseSlots(before)
	fmt.Printf("slot count %d -> %d, size %d -> %d\n", len(oldSlots), len(l.Slots), len(before), len(after))
	n := len(oldSlots)
	if len(l.Slots) < n {
		n = len(l.Slots)
	}
	changed := 0
	for i := 0; i < n; i++ {
		a, b := oldSlots[i].String(), l.Slots[i].String()
		if a != b {
			changed++
			fmt.Printf("  slot %-4d %s -> %s\n", i, truncate(a, 70), truncate(b, 70))
		}
	}
	fmt.Printf("changed slots: %d\n", changed)
}

type launchFlags struct {
	port       int
	timeout    time.Duration
	uddir      string
	proxy      string
	url        string
	windowSize string
	maximized  bool
	minimized  bool
	headless   bool
	noCDP      bool
	keepOpen   bool
	detach     bool
	logPath    string
}

func (f launchFlags) options(s *Store, id string, cfg *Config) LaunchOptions {
	uddir := f.uddir
	if uddir == "" {
		uddir = s.Dir(id)
	}
	proxy := f.proxy
	if proxy == "" {
		if p, err := s.Proxy(id); err == nil && p != nil {
			proxy = p.Server()
		}
	}
	return LaunchOptions{
		ProfileID:   id,
		Port:        f.port,
		UserDataDir: uddir,
		Proxy:       proxy,
		WindowSize:  f.windowSize,
		Maximized:   f.maximized,
		Minimized:   f.minimized,
		Headless:    f.headless,
		URL:         f.url,
		Timeout:     f.timeout,
		ApplyCDP:    !f.noCDP,
		LogPath:     f.logPath,
	}
}

func cmdLaunch(id string, f launchFlags) error {
	if id == "" {
		return fmt.Errorf("launch requires a profile id")
	}
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	cfg, err := s.Config(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(s.StartInfoPath(id)); err != nil {
		return fmt.Errorf("profile %s has no startInfo yet; the browser will refuse to start", id)
	}
	o := f.options(s, id, cfg)
	fmt.Printf("chrome   %s\n", ChromePath())
	fmt.Printf("profile  %s (%s)\n", cfg.Profile.Name, id)
	fmt.Printf("uddir    %s\n", o.UserDataDir)
	if o.Proxy != "" {
		fmt.Printf("proxy    %s\n", o.Proxy)
	}
	run, err := Launch(o)
	if err != nil {
		if run != nil {
			fmt.Printf("command  %s %s\n", run.ExecPath, strings.Join(run.Args, " "))
		}
		return err
	}
	fmt.Printf("command  %s %s\n", run.ExecPath, strings.Join(run.Args, " "))
	fmt.Printf("pid      %d\n", run.Cmd.Process.Pid)
	fmt.Printf("devtools http://127.0.0.1:%d/json/version\n", run.Port)
	fmt.Printf("browser  %s\n", run.Version.Browser)
	fmt.Printf("ua       %s\n", run.Version.UserAgent)
	var stop func()
	if !f.noCDP {
		ov := NewOverrider(cfg)
		stop, err = ov.Start(run.Port)
		if err != nil {
			fmt.Printf("cdp      partial: %v\n", err)
		} else {
			fmt.Printf("cdp      applied %s\n", strings.Join(ov.Applied(), ", "))
		}
	}
	if f.detach {
		if stop != nil {
			stop()
		}
		fmt.Println("detached: the browser keeps running, but new tabs will not")
		fmt.Println("receive the fingerprint overrides; use verify to check")
		return nil
	}
	// Supervise: the override watcher has to stay connected for as long as the
	// browser runs, otherwise tabs opened later fall back to the default hints.
	fmt.Println("supervising; press Ctrl-C to stop watching (the browser keeps running)")
	waitForExit(run, stop)
	return nil
}

func waitForExit(run *Running, stop func()) {
	done := make(chan error, 1)
	go func() { done <- run.Cmd.Wait() }()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	select {
	case err := <-done:
		fmt.Printf("browser exited (%v)\n", err)
	case <-sig:
		fmt.Println("stopped watching; the browser keeps running")
	}
	if stop != nil {
		stop()
	}
}

func cmdVerify(id string, f launchFlags, asJSON bool) error {
	if id == "" {
		return fmt.Errorf("verify requires a profile id")
	}
	s, err := mustStore()
	if err != nil {
		return err
	}
	defer s.DB.Close()
	cfg, err := s.Config(id)
	if err != nil {
		return err
	}
	o := f.options(s, id, cfg)
	run, err := Launch(o)
	if err != nil {
		return err
	}
	note := func(format string, a ...any) {
		if !asJSON {
			fmt.Printf(format, a...)
		}
	}
	note("launched pid %d on port %d (%s)\n", run.Cmd.Process.Pid, run.Port, run.Version.Browser)
	if !f.noCDP {
		ov := NewOverrider(cfg)
		stop, err := ov.Start(run.Port)
		if err != nil {
			note("cdp partial: %v\n", err)
		} else {
			defer stop()
			note("cdp applied %s\n", strings.Join(ov.Applied(), ", "))
		}
	}
	time.Sleep(1 * time.Second)
	fp, err := ReadFingerprint(run.Port, "https://example.com/")
	if err != nil {
		return err
	}
	e := cfg.Expected()
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"id": id, "name": cfg.Profile.Name, "port": run.Port,
			"pid": run.Cmd.Process.Pid, "expected": e, "live": fp,
			"differences": Compare(e, fp),
		})
	}
	fmt.Println()
	fmt.Println("expected vs live")
	fmt.Print(Summarize(e, fp))
	if bad := Compare(e, fp); len(bad) > 0 {
		fmt.Println()
		fmt.Println("differences:")
		sort.Strings(bad)
		for _, d := range bad {
			fmt.Printf("  - %s\n", d)
		}
	} else {
		fmt.Println()
		fmt.Println("all compared fields match the database")
	}
	fmt.Println()
	fmt.Println("full live fingerprint:")
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(fp)
	if f.keepOpen {
		fmt.Println("browser left running; press Enter to exit")
		_, _ = os.Stdin.Read(make([]byte, 1))
	}
	return nil
}
