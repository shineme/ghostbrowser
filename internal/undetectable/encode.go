package undetectable

import (
	"fmt"
	"strings"
)

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
	if cfg.UAD != nil {
		l.SetSlot(l.OSVerFull, AsciiSlot(cfg.UAD.PlatformVersion))
	}
	l.SetSlot(l.Arch, AsciiSlot(e.Arch))
	l.SetSlot(l.Bitness, AsciiSlot(e.Bitness))
	appVersion := strings.TrimPrefix(e.UA, "Mozilla/")
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
	if l.OSVers >= 0 {
		l.SetSlot(l.OSVers, UTF16Slot(e.OSVersion))
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
		if len(cfg.UAD.Brands) != 3 || len(cfg.UAD.FullVersionList) != 3 {
			return nil, fmt.Errorf("layout B requires exactly 3 brands and fullVersionList entries")
		}
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
