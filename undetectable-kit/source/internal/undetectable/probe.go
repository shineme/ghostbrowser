package undetectable

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ProbeNewTab checks a fresh target so retained CDP coverage is part of the
// observation. The local DevTools HTTP origin is a secure localhost context.
func ProbeNewTab(port int) (*Fingerprint, error) {
	ver, err := HTTPVersion(port)
	if err != nil {
		return nil, err
	}
	b, err := DialCDP(ver.WebSocketDebuggerURL, 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	var target struct {
		TargetID string `json:"targetId"`
	}
	if err = b.Call("Target.createTarget", map[string]any{"url": fmt.Sprintf("http://localhost:%d/json/version", port)}, &target); err != nil {
		return nil, err
	}
	defer b.Call("Target.closeTarget", map[string]any{"targetId": target.TargetID}, nil)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		targets, err := ListTargets(port)
		if err != nil {
			return nil, err
		}
		for _, t := range targets {
			if t.ID == target.TargetID {
				c, err := DialCDP(t.WebSocketDebuggerURL, 10*time.Second)
				if err != nil {
					continue
				}
				var ready string
				_ = c.Evaluate("document.readyState", &ready)
				if ready != "complete" {
					c.Close()
					continue
				}
				var raw string
				err = c.Evaluate(fpProbe, &raw)
				c.Close()
				if err != nil {
					return nil, err
				}
				var fp Fingerprint
				err = json.Unmarshal([]byte(raw), &fp)
				return &fp, err
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("fingerprint probe page did not become ready")
}

func CompareFull(c *Config, fp *Fingerprint) []string {
	e := c.Expected()
	bad := Compare(e, fp)
	check := func(name string, a, b any) {
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		if string(x) != string(y) {
			bad = append(bad, fmt.Sprintf("%s: want %s got %s", name, x, y))
		}
	}
	check("appVersion", strings.TrimPrefix(e.UA, "Mozilla/"), fp.AppVersion)
	check("platform", e.Platform, fp.Platform)
	check("languages", e.Languages, fp.Languages)
	if len(fp.Screen) >= 6 {
		check("colorDepth", e.ColorDepth, int64(fp.Screen[4]))
		check("pixelDepth", e.PixelDepth, int64(fp.Screen[5]))
	}
	if len(fp.WebGL) >= 4 {
		check("webglVendor", e.WebGLVendor, fp.WebGL[2])
	}
	if c.UAD != nil {
		low, _ := fp.UAData.(map[string]any)
		high, _ := fp.HighEntropy.(map[string]any)
		check("brands", c.UAD.Brands, low["brands"])
		check("fullVersionList", c.UAD.FullVersionList, high["fullVersionList"])
		for k, w := range map[string]any{"architecture": c.UAD.Architecture, "bitness": c.UAD.Bitness, "platform": c.UAD.Platform, "platformVersion": c.UAD.PlatformVersion, "uaFullVersion": c.UAD.UAFullVersion, "wow64": c.UAD.Wow64, "mobile": c.UAD.Mobile, "model": c.UAD.Model, "formFactors": c.UAD.FormFactors} {
			check(k, w, high[k])
		}
	}
	return bad
}
