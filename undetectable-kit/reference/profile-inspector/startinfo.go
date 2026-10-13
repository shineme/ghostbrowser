package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"unicode/utf16"
)

// ---------------------------------------------------------------------------
// startInfo binary format
//
// The file is a flat stream of length-prefixed records with no alignment:
//
//	<u64 little-endian payload length><payload>
//
// A payload is treated as a record when it is
//
//	1. all bytes in 0x20..0x7E            -> ASCII string
//	2. even length, all UTF-16LE code
//	   units >= 0x20 with no surrogates    -> UTF-16LE string
//	3. zero length                         -> empty record
//
// Anything else is not a record: the single byte is accumulated verbatim into
// the current RAW slot. RAW slots carry the packed scalar fields (CPU cores,
// memory, screen metrics, canvas noise, storage quota, JS heap limit).
// ---------------------------------------------------------------------------

type SlotKind int

const (
	KindRaw SlotKind = iota
	KindEmpty
	KindStr
)

type Slot struct {
	Kind SlotKind
	Enc  string // "ascii" or "utf16" for KindStr
	Data []byte
}

func (s Slot) Size() int {
	if s.Kind == KindRaw {
		return len(s.Data)
	}
	return 8 + len(s.Data)
}

func (s Slot) IsZero() bool {
	for _, b := range s.Data {
		if b != 0 {
			return false
		}
	}
	return true
}

func (s Slot) Text() string {
	switch s.Enc {
	case "ascii":
		return string(s.Data)
	case "utf16":
		if len(s.Data)%2 != 0 {
			return ""
		}
		u := make([]uint16, len(s.Data)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(s.Data[i*2:])
		}
		return string(utf16.Decode(u))
	}
	return ""
}

func (s Slot) String() string {
	switch s.Kind {
	case KindRaw:
		return fmt.Sprintf("RAW(%d) %x", len(s.Data), s.Data)
	case KindEmpty:
		return "EMPTY"
	default:
		t := s.Text()
		if len(t) > 60 {
			t = t[:60] + "..."
		}
		return fmt.Sprintf("STR/%s %q", s.Enc, t)
	}
}

func AsciiSlot(s string) Slot { return Slot{Kind: KindStr, Enc: "ascii", Data: []byte(s)} }

func UTF16Slot(s string) Slot {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return Slot{Kind: KindStr, Enc: "utf16", Data: b}
}

func EmptySlot() Slot { return Slot{Kind: KindEmpty} }

func RawSlot(b ...byte) Slot { return Slot{Kind: KindRaw, Data: append([]byte(nil), b...)} }

func U32Raw(v uint32) Slot {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return RawSlot(b...)
}

func U64Raw(v uint64) Slot {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return RawSlot(b...)
}

func classify(p []byte) (SlotKind, string, bool) {
	if len(p) == 0 {
		return KindEmpty, "", true
	}
	ascii := true
	for _, b := range p {
		if b < 0x20 || b > 0x7E {
			ascii = false
			break
		}
	}
	if ascii {
		return KindStr, "ascii", true
	}
	if len(p)%2 == 0 {
		ok := true
		for i := 0; i+1 < len(p); i += 2 {
			u := binary.LittleEndian.Uint16(p[i:])
			if u < 0x20 || (u >= 0xD800 && u <= 0xDFFF) {
				ok = false
				break
			}
		}
		if ok {
			return KindStr, "utf16", true
		}
	}
	return 0, "", false
}

func ParseSlots(d []byte) []Slot {
	out := make([]Slot, 0, 512)
	n := len(d)
	for i := 0; i < n; {
		made := false
		if i+8 <= n {
			l := binary.LittleEndian.Uint64(d[i:])
			if l <= uint64(n-i-8) {
				payload := d[i+8 : i+8+int(l)]
				if k, enc, ok := classify(payload); ok {
					out = append(out, Slot{Kind: k, Enc: enc, Data: append([]byte(nil), payload...)})
					i += 8 + int(l)
					made = true
				}
			}
		}
		if !made {
			if len(out) > 0 && out[len(out)-1].Kind == KindRaw {
				out[len(out)-1].Data = append(out[len(out)-1].Data, d[i])
			} else {
				out = append(out, Slot{Kind: KindRaw, Data: []byte{d[i]}})
			}
			i++
		}
	}
	return out
}

func EncodeSlots(sl []Slot) []byte {
	var b bytes.Buffer
	for _, s := range sl {
		if s.Kind == KindRaw {
			b.Write(s.Data)
			continue
		}
		var hdr [8]byte
		binary.LittleEndian.PutUint64(hdr[:], uint64(len(s.Data)))
		b.Write(hdr[:])
		b.Write(s.Data)
	}
	return b.Bytes()
}

// ---------------------------------------------------------------------------
// Layout resolution
// ---------------------------------------------------------------------------

type Layout struct {
	Slots []Slot
	IsB   bool

	Name        int
	OSVers      int
	CoresRaw    int
	GPUArch     int
	GPUVendor   int
	UA1, UA2    int
	Platform    int
	ProductSub  int
	Vendor      int
	VerShort    int // layout B only
	VerFull     int // layout B only
	UAPlatform  int
	OSVerFull   int
	Arch        int
	Bitness     int
	AppVersion  int
	FormFactor  int
	BrandCount  int // layout B only
	FullCount   int // layout B only
	PublicRTC   int
	WebGLRender int
	WebGLVendor int
	TZ          int
	Lang        int
	ZhTW        int
	ScreenBase  int
}

var (
	reIPv4 = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	reTZ   = regexp.MustCompile(`^[A-Za-z][A-Za-z_+-]*/[A-Za-z][A-Za-z_+0-9-]*$`)
)

func findSlot(sl []Slot, from int, pred func(Slot) bool) int {
	for i := from; i < len(sl); i++ {
		if pred(sl[i]) {
			return i
		}
	}
	return -1
}

func isText(s Slot) bool  { return s.Kind == KindStr && s.Enc != "" }
func textEq(w string) func(Slot) bool {
	return func(s Slot) bool { return isText(s) && s.Text() == w }
}

func ResolveLayout(sl []Slot) (*Layout, error) {
	l := &Layout{Slots: sl, Name: -1, OSVers: -1, CoresRaw: -1, GPUArch: -1, GPUVendor: -1,
		UA1: -1, UA2: -1, Platform: -1, ProductSub: -1, Vendor: -1, VerShort: -1, VerFull: -1,
		UAPlatform: -1, OSVerFull: -1, Arch: -1, Bitness: -1, AppVersion: -1, FormFactor: -1,
		BrandCount: -1, FullCount: -1, PublicRTC: -1, WebGLRender: -1, WebGLVendor: -1,
		TZ: -1, Lang: -1, ZhTW: -1, ScreenBase: -1}
	if len(sl) < 40 {
		return nil, fmt.Errorf("startInfo too small (%d slots)", len(sl))
	}
	l.IsB = findSlot(sl, 0, textEq("Not_A Brand")) >= 0

	l.Name = 0
	if !isText(sl[0]) {
		return nil, fmt.Errorf("slot 0 is not the profile name")
	}
	// CPU cores / memory packed record: first RAW slot >= 25 bytes near the head.
	l.CoresRaw = findSlot(sl, 1, func(s Slot) bool { return s.Kind == KindRaw && len(s.Data) >= 25 })
	if l.CoresRaw < 0 {
		return nil, fmt.Errorf("CPU/memory record not found")
	}
	l.OSVers = findSlot(sl, 1, func(s Slot) bool {
		if !isText(s) || s.Enc != "utf16" {
			return false
		}
		t := s.Text()
		return t == "10" || t == "11"
	})
	l.GPUArch = findSlot(sl, l.CoresRaw+1, func(s Slot) bool { return isText(s) && s.Enc == "ascii" })
	if l.GPUArch >= 0 {
		l.GPUVendor = findSlot(sl, l.GPUArch+1, func(s Slot) bool { return isText(s) && s.Enc == "ascii" })
	}
	// User agent: two adjacent identical ASCII strings starting with Mozilla/5.0
	for i := 0; i+1 < len(sl); i++ {
		a, b := sl[i], sl[i+1]
		if isText(a) && isText(b) && a.Enc == "ascii" && b.Enc == "ascii" &&
			strings.HasPrefix(a.Text(), "Mozilla/5.0") && a.Text() == b.Text() {
			l.UA1, l.UA2 = i, i+1
			break
		}
	}
	if l.UA1 < 0 {
		return nil, fmt.Errorf("user agent pair not found")
	}
	l.Platform = l.UA2 + 1
	l.ProductSub = l.UA2 + 2
	l.Vendor = l.UA2 + 3
	if !isText(sl[l.Platform]) || !isText(sl[l.ProductSub]) || !isText(sl[l.Vendor]) {
		return nil, fmt.Errorf("platform/vendor block not in expected shape")
	}
	l.UAPlatform = findSlot(sl, l.Vendor+1, textEq("Windows"))
	if l.UAPlatform < 0 {
		l.UAPlatform = findSlot(sl, l.Vendor+1, textEq("macOS"))
	}
	if l.UAPlatform < 0 {
		return nil, fmt.Errorf("UA platform string not found")
	}
	l.OSVerFull = findSlot(sl, l.UAPlatform+1, func(s Slot) bool {
		return isText(s) && s.Enc == "ascii" && regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(s.Text())
	})
	l.Arch = findSlot(sl, l.OSVerFull+1, func(s Slot) bool {
		t := s.Text()
		return isText(s) && s.Enc == "ascii" && (t == "x86" || t == "arm")
	})
	l.Bitness = findSlot(sl, l.Arch+1, func(s Slot) bool {
		t := s.Text()
		return isText(s) && s.Enc == "ascii" && (t == "64" || t == "32")
	})
	l.AppVersion = findSlot(sl, l.Bitness+1, func(s Slot) bool {
		return isText(s) && s.Enc == "ascii" && strings.HasPrefix(s.Text(), "5.0 (")
	})
	l.FormFactor = findSlot(sl, l.AppVersion+1, func(s Slot) bool {
		t := s.Text()
		return isText(s) && s.Enc == "ascii" && (t == "Desktop" || t == "Mobile" || t == "Tablet")
	})
	if l.AppVersion < 0 || l.FormFactor < 0 {
		return nil, fmt.Errorf("appVersion/formFactor not found")
	}
	if l.IsB {
		// short version record lives between the name and osVers
		l.VerShort = findSlot(sl, 1, func(s Slot) bool {
			return isText(s) && s.Enc == "utf16" && regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(s.Text())
		})
		l.VerFull = findSlot(sl, l.Vendor+1, func(s Slot) bool {
			return isText(s) && s.Enc == "ascii" && regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`).MatchString(s.Text())
		})
		l.BrandCount = findSlot(sl, l.FormFactor+1, func(s Slot) bool { return s.Kind == KindRaw && len(s.Data) == 4 })
		if l.BrandCount >= 0 {
			l.FullCount = findSlot(sl, l.BrandCount+1, func(s Slot) bool { return s.Kind == KindRaw && len(s.Data) == 4 })
		}
	}
	l.PublicRTC = findSlot(sl, l.FormFactor+1, func(s Slot) bool {
		return isText(s) && s.Enc == "ascii" && reIPv4.MatchString(s.Text())
	})
	l.WebGLRender = findSlot(sl, l.FormFactor+1, func(s Slot) bool {
		return isText(s) && s.Enc == "ascii" && strings.HasPrefix(s.Text(), "ANGLE (")
	})
	if l.WebGLRender >= 0 {
		l.WebGLVendor = findSlot(sl, l.WebGLRender+1, func(s Slot) bool {
			return isText(s) && s.Enc == "ascii" && strings.Contains(s.Text(), "Inc.")
		})
	}
	l.ZhTW = findSlot(sl, 0, textEq("zh-TW"))
	// time zone: last ASCII slot matching Area/Location before the language tag
	l.TZ = findSlot(sl, 0, func(s Slot) bool {
		return isText(s) && s.Enc == "ascii" && reTZ.MatchString(s.Text())
	})
	if l.TZ < 0 {
		return nil, fmt.Errorf("time zone not found")
	}
	// language: ASCII tag right after the raw byte following the time zone
	if l.TZ+2 < len(sl) && isText(sl[l.TZ+2]) {
		l.Lang = l.TZ + 2
	}
	return l, nil
}

func (l *Layout) raw() []byte { return EncodeSlots(l.Slots) }

// ---------------------------------------------------------------------------
// Field accessors
// ---------------------------------------------------------------------------

func (l *Layout) SlotOffset(i int) int {
	off := 0
	for k := 0; k < i; k++ {
		off += l.Slots[k].Size()
	}
	return off
}

func (l *Layout) SetSlot(i int, s Slot) {
	if i >= 0 && i < len(l.Slots) {
		l.Slots[i] = s
	}
}

// Scalar fields live inside RAW slots, which the browser reads as fixed-size
// values at fixed offsets. They are patched directly into the serialized
// buffer: re-parsing after a patch could reclassify a RAW run into records and
// shift every later offset.

func putU32(b []byte, off int, v uint32) error {
	if off < 0 || off+4 > len(b) {
		return fmt.Errorf("u32 write out of range at %d (size %d)", off, len(b))
	}
	binary.LittleEndian.PutUint32(b[off:], v)
	return nil
}

func putU64(b []byte, off int, v uint64) error {
	if off < 0 || off+8 > len(b) {
		return fmt.Errorf("u64 write out of range at %d (size %d)", off, len(b))
	}
	binary.LittleEndian.PutUint64(b[off:], v)
	return nil
}

func putF64(b []byte, off int, v float64) error {
	if off < 0 || off+8 > len(b) {
		return fmt.Errorf("f64 write out of range at %d (size %d)", off, len(b))
	}
	binary.LittleEndian.PutUint64(b[off:], math.Float64bits(v))
	return nil
}

func (l *Layout) GetU32(off int) (uint32, error) {
	b := l.raw()
	if off < 0 || off+4 > len(b) {
		return 0, fmt.Errorf("u32 read out of range at %d", off)
	}
	return binary.LittleEndian.Uint32(b[off:]), nil
}

func (l *Layout) GetU64(off int) (uint64, error) {
	b := l.raw()
	if off < 0 || off+8 > len(b) {
		return 0, fmt.Errorf("u64 read out of range at %d", off)
	}
	return binary.LittleEndian.Uint64(b[off:]), nil
}

func (l *Layout) GetF64(off int) (float64, error) {
	b := l.raw()
	if off < 0 || off+8 > len(b) {
		return 0, fmt.Errorf("f64 read out of range at %d", off)
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(b[off:])), nil
}

// Scalar values live inside RAW slots that the browser reads at fixed offsets.
// After the string slots have been rewritten the file is serialized once, the
// scalar anchors are located in that final buffer, and the numbers are patched
// in place. Re-parsing the buffer between patches would risk reclassifying a
// RAW run as records and shifting every following offset.

type Anchor struct {
	CoresBase  int // packed CPU/memory/canvas record
	ScreenBase int // screen metrics block
}

type Scalars struct {
	MemoryGB      uint32
	Cores         uint32
	CanvasRandom0 uint32
	CanvasRandom1 uint32
	Pixel1        uint32
	Pixel2        uint32
	Width         uint32
	Height        uint32
	AvailWidth    uint32
	AvailHeight   uint32
	DPR           float64
	ColorDepth    uint32
	PixelDepth    uint32
	Acc           float64
	Storage       uint64
	HeapLimit     uint64
}

// locateScalars finds both anchors in a serialized startInfo buffer.
func locateScalars(sl []Slot, buf []byte, l *Layout) (*Anchor, error) {
	s := &Anchor{CoresBase: -1, ScreenBase: -1}
	if l.CoresRaw >= 0 {
		off := 0
		for i := 0; i < l.CoresRaw; i++ {
			off += sl[i].Size()
		}
		if off+29 <= len(buf) {
			s.CoresBase = off
		}
	}
	if l.ZhTW >= 0 && l.ZhTW < len(sl) {
		off := 0
		for i := 0; i < l.ZhTW; i++ {
			off += sl[i].Size()
		}
		off += sl[l.ZhTW].Size()
		for j := off; j+8 <= len(buf); j++ {
			if binary.LittleEndian.Uint64(buf[j:]) == 0 {
				s.ScreenBase = j + 8
				break
			}
		}
	}
	if s.ScreenBase < 0 || s.ScreenBase+80 > len(buf) {
		return nil, fmt.Errorf("screen metrics block not found after serialization")
	}
	return s, nil
}

// PatchScalars writes the packed numeric fields.
//
// Packed CPU/memory record:
//
//	+0  u8  1
//	+1  u8  0
//	+2  u32 deviceMemory (GB)
//	+6  u32 hardwareConcurrency
//	+10 u8  0
//	+11 u32 CanvasWebglRandomParameter
//	+15 u32 CanvasWebglRandomParameter2
//	+19 u32 pixel1
//	+23 u32 pixel2
//	+27 u8  1
//	+28 u8  1
//
// Screen block:
//
//	+0  u32 screen.width
//	+4  u32 screen.height
//	+8  u32 screen.availWidth
//	+12 u32 screen.availHeight
//	+16 f64 devicePixelRatio
//	+28 u32 screen.colorDepth
//	+32 u32 screen.pixelDepth
//	+54 f64 canvas "acc" noise
//	+63 u64 navigator.storage.estimate() quota
//	+72 u64 performance.jsHeapSizeLimit
func PatchScalars(buf []byte, s *Anchor, v Scalars) error {
	if s.CoresBase >= 0 {
		for _, w := range []struct {
			off int
			val uint32
		}{{s.CoresBase + 2, v.MemoryGB}, {s.CoresBase + 6, v.Cores},
			{s.CoresBase + 11, v.CanvasRandom0}, {s.CoresBase + 15, v.CanvasRandom1},
			{s.CoresBase + 19, v.Pixel1}, {s.CoresBase + 23, v.Pixel2}} {
			if err := putU32(buf, w.off, w.val); err != nil {
				return err
			}
		}
	}
	b := s.ScreenBase
	for _, w := range []struct {
		off int
		val uint32
	}{{b + 0, v.Width}, {b + 4, v.Height}, {b + 8, v.AvailWidth}, {b + 12, v.AvailHeight},
		{b + 28, v.ColorDepth}, {b + 32, v.PixelDepth}} {
		if err := putU32(buf, w.off, w.val); err != nil {
			return err
		}
	}
	if err := putF64(buf, b+16, v.DPR); err != nil {
		return err
	}
	if err := putF64(buf, b+54, v.Acc); err != nil {
		return err
	}
	if err := putU64(buf, b+63, v.Storage); err != nil {
		return err
	}
	return putU64(buf, b+72, v.HeapLimit)
}

// ReadScalars decodes the numeric fields for display.
func ReadScalars(buf []byte, s *Anchor) (Scalars, error) {
	var v Scalars
	if s.CoresBase >= 0 {
		v.MemoryGB = binary.LittleEndian.Uint32(buf[s.CoresBase+2:])
		v.Cores = binary.LittleEndian.Uint32(buf[s.CoresBase+6:])
		v.CanvasRandom0 = binary.LittleEndian.Uint32(buf[s.CoresBase+11:])
		v.CanvasRandom1 = binary.LittleEndian.Uint32(buf[s.CoresBase+15:])
		v.Pixel1 = binary.LittleEndian.Uint32(buf[s.CoresBase+19:])
		v.Pixel2 = binary.LittleEndian.Uint32(buf[s.CoresBase+23:])
	}
	b := s.ScreenBase
	if b+80 > len(buf) {
		return v, fmt.Errorf("screen block out of range")
	}
	v.Width = binary.LittleEndian.Uint32(buf[b+0:])
	v.Height = binary.LittleEndian.Uint32(buf[b+4:])
	v.AvailWidth = binary.LittleEndian.Uint32(buf[b+8:])
	v.AvailHeight = binary.LittleEndian.Uint32(buf[b+12:])
	v.DPR = math.Float64frombits(binary.LittleEndian.Uint64(buf[b+16:]))
	v.ColorDepth = binary.LittleEndian.Uint32(buf[b+28:])
	v.PixelDepth = binary.LittleEndian.Uint32(buf[b+32:])
	v.Acc = math.Float64frombits(binary.LittleEndian.Uint64(buf[b+54:]))
	v.Storage = binary.LittleEndian.Uint64(buf[b+63:])
	v.HeapLimit = binary.LittleEndian.Uint64(buf[b+72:])
	return v, nil
}

func LoadStartInfo(path string) (*Layout, []byte, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	sl := ParseSlots(d)
	if !bytes.Equal(EncodeSlots(sl), d) {
		return nil, nil, fmt.Errorf("startInfo does not round-trip; refusing to modify %s", path)
	}
	l, err := ResolveLayout(sl)
	if err != nil {
		return nil, nil, err
	}
	return l, d, nil
}
