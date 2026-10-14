package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Extension Center: Chrome extensions downloaded once into
// %APPDATA%/GhostBrowser/extensions/<id> (unpacked) and assigned to profiles.
// At launch the assigned folders are passed with --load-extension.
//
// CRX files are a small header followed by an ordinary zip:
//   CRX2: "Cr24" | u32 version=2 | u32 keyLen | u32 sigLen | key | sig | zip
//   CRX3: "Cr24" | u32 version=3 | u32 headerLen | CrxFileHeader (protobuf) | zip
// The CRX3 header carries the public keys and, inside signed_header_data, the
// 16-byte crx_id. We read the matching key and write it into manifest.json as
// "key", so the unpacked copy keeps the same extension ID as the store build
// (extension storage, OAuth redirects and native messaging depend on the ID).

type Extension struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Source      string `json:"source"` // store | url | file
	Origin      string `json:"origin"` // store ID, URL or file path it came from
	Dir         string `json:"dir"`
	Icon        string `json:"icon"` // data: URL of the largest icon (≤256 KiB), "" if none
	MV          int    `json:"manifestVersion"`
	Size        int64  `json:"size"` // bytes of the downloaded package
	AddedAt     int64  `json:"addedAt"`
	Profiles    int    `json:"profiles"` // number of profiles using it (filled by ListExtensions)
}

var extIDRe = regexp.MustCompile(`^[a-p]{32}$`)

func extensionsDir() string {
	d := filepath.Join(appDataDir(), "extensions")
	_ = os.MkdirAll(d, 0o755)
	return d
}

// ---------- registry (extensions/index.json) ----------

type extRegistry struct {
	mu   sync.Mutex
	root string
}

var extReg = &extRegistry{}

func (r *extRegistry) dir() string {
	if r.root != "" {
		return r.root
	}
	return extensionsDir()
}

func (r *extRegistry) load() map[string]Extension {
	m := map[string]Extension{}
	b, err := os.ReadFile(filepath.Join(r.dir(), "index.json"))
	if err == nil {
		var list []Extension
		if json.Unmarshal(b, &list) == nil {
			for _, e := range list {
				m[e.ID] = e
			}
		}
	}
	return m
}

func (r *extRegistry) save(m map[string]Extension) error {
	list := make([]Extension, 0, len(m))
	for _, e := range m {
		e.Profiles = 0
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].AddedAt > list[j].AddedAt })
	b, _ := json.MarshalIndent(list, "", "  ")
	tmp := filepath.Join(r.dir(), "index.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(r.dir(), "index.json"))
}

func (r *extRegistry) list() []Extension {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.load()
	out := make([]Extension, 0, len(m))
	for _, e := range m {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AddedAt > out[j].AddedAt })
	return out
}

func (r *extRegistry) get(id string) (Extension, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.load()[id]
	return e, ok
}

func (r *extRegistry) put(e Extension) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.load()
	if old, ok := m[e.ID]; ok && old.AddedAt != 0 {
		e.AddedAt = old.AddedAt
	}
	m[e.ID] = e
	return r.save(m)
}

func (r *extRegistry) remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m := r.load()
	e, ok := m[id]
	if !ok {
		return fmt.Errorf("扩展不存在")
	}
	delete(m, id)
	if err := r.save(m); err != nil {
		return err
	}
	if e.Dir != "" && strings.HasPrefix(filepath.Clean(e.Dir), filepath.Clean(r.dir())) {
		_ = os.RemoveAll(e.Dir)
	}
	return nil
}

// ---------- CRX parsing ----------

type crxInfo struct {
	Version int
	CrxID   string // a-p encoded, "" when unknown
	PubKey  []byte // public key whose ID matches CrxID (or the CRX2 key)
	Zip     []byte
}

// parseCRX strips the CRX2/CRX3 header and returns the zip payload. A plain zip
// (PK\x03\x04) is accepted as is.
func parseCRX(b []byte) (*crxInfo, error) {
	if len(b) >= 4 && bytes.Equal(b[:4], []byte("PK\x03\x04")) {
		return &crxInfo{Version: 0, Zip: b}, nil
	}
	if len(b) < 16 || string(b[:4]) != "Cr24" {
		return nil, fmt.Errorf("不是有效的 CRX 文件（缺少 Cr24 文件头）")
	}
	ver := binary.LittleEndian.Uint32(b[4:8])
	switch ver {
	case 2:
		kl := binary.LittleEndian.Uint32(b[8:12])
		sl := binary.LittleEndian.Uint32(b[12:16])
		start := 16 + uint64(kl) + uint64(sl)
		if start > uint64(len(b)) {
			return nil, fmt.Errorf("CRX2 文件头长度错误")
		}
		key := b[16 : 16+kl]
		return &crxInfo{Version: 2, PubKey: key, CrxID: extIDFromKey(key), Zip: b[start:]}, nil
	case 3:
		hl := binary.LittleEndian.Uint32(b[8:12])
		start := 12 + uint64(hl)
		if start > uint64(len(b)) {
			return nil, fmt.Errorf("CRX3 文件头长度错误")
		}
		info := &crxInfo{Version: 3, Zip: b[start:]}
		info.CrxID, info.PubKey = crx3Identity(b[12:start])
		return info, nil
	}
	return nil, fmt.Errorf("不支持的 CRX 版本 %d", ver)
}

// crx3Identity reads CrxFileHeader { repeated AsymmetricKeyProof sha256_with_rsa = 2;
// repeated AsymmetricKeyProof sha256_with_ecdsa = 3; bytes signed_header_data = 10000; }
// with SignedData { bytes crx_id = 1; } and AsymmetricKeyProof { bytes public_key = 1; }.
func crx3Identity(h []byte) (string, []byte) {
	var keys [][]byte
	id := ""
	_ = pbWalk(h, func(field int, val []byte) {
		switch field {
		case 2, 3:
			_ = pbWalk(val, func(f int, v []byte) {
				if f == 1 {
					keys = append(keys, v)
				}
			})
		case 10000:
			_ = pbWalk(val, func(f int, v []byte) {
				if f == 1 && len(v) == 16 {
					id = encodeExtID(v)
				}
			})
		}
	})
	for _, k := range keys {
		if id == "" || extIDFromKey(k) == id {
			if id == "" {
				id = extIDFromKey(k)
			}
			return id, k
		}
	}
	return id, nil
}

// pbWalk visits the length-delimited fields of a protobuf message; other wire
// types are skipped.
func pbWalk(b []byte, fn func(field int, val []byte)) error {
	for len(b) > 0 {
		tag, n := binary.Uvarint(b)
		if n <= 0 {
			return errors.New("bad varint")
		}
		b = b[n:]
		field, wt := int(tag>>3), tag&7
		switch wt {
		case 0:
			_, n = binary.Uvarint(b)
			if n <= 0 {
				return errors.New("bad varint")
			}
			b = b[n:]
		case 1:
			if len(b) < 8 {
				return errors.New("short")
			}
			b = b[8:]
		case 5:
			if len(b) < 4 {
				return errors.New("short")
			}
			b = b[4:]
		case 2:
			l, n := binary.Uvarint(b)
			if n <= 0 || uint64(len(b)-n) < l {
				return errors.New("bad length")
			}
			fn(field, b[n:n+int(l)])
			b = b[n+int(l):]
		default:
			return errors.New("unsupported wire type")
		}
	}
	return nil
}

// encodeExtID maps 16 bytes to Chrome's a-p alphabet (one letter per nibble).
func encodeExtID(b []byte) string {
	h := hex.EncodeToString(b)
	out := make([]byte, len(h))
	for i := 0; i < len(h); i++ {
		c := h[i]
		if c >= 'a' {
			c = c - 'a' + 10
		} else {
			c -= '0'
		}
		out[i] = 'a' + c
	}
	return string(out)
}

func extIDFromKey(key []byte) string {
	sum := sha256.Sum256(key)
	return encodeExtID(sum[:16])
}

// ---------- unpack ----------

type extManifest struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	DefaultLocale string            `json:"default_locale"`
	MV            int               `json:"manifest_version"`
	Key           string            `json:"key"`
	Icons         map[string]string `json:"icons"`
}

// readManifest opens manifest.json from a zip payload.
func readManifest(zr *zip.Reader) (extManifest, map[string]json.RawMessage, error) {
	var m extManifest
	for _, f := range zr.File {
		if strings.TrimPrefix(f.Name, "./") != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return m, nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, 4<<20))
		rc.Close()
		if err != nil {
			return m, nil, err
		}
		b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
		raw := map[string]json.RawMessage{}
		if err := json.Unmarshal(b, &raw); err != nil {
			return m, nil, fmt.Errorf("manifest.json 不是合法 JSON：%w", err)
		}
		_ = json.Unmarshal(b, &m)
		return m, raw, nil
	}
	return m, nil, fmt.Errorf("扩展包里没有 manifest.json")
}

// unpackExtension extracts the zip into dir (replacing it), drops the
// _metadata folder (Chrome refuses unpacked extensions with "_"-prefixed
// top-level names other than _locales) and pins the ID through manifest "key".
func unpackExtension(info *crxInfo, dir string) (extManifest, error) {
	zr, err := zip.NewReader(bytes.NewReader(info.Zip), int64(len(info.Zip)))
	if err != nil {
		return extManifest{}, fmt.Errorf("扩展包解压失败：%w", err)
	}
	man, raw, err := readManifest(zr)
	if err != nil {
		return man, err
	}
	if man.Version == "" {
		return man, fmt.Errorf("manifest.json 缺少 version")
	}
	tmp := dir + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return man, err
	}
	var total int64
	for _, f := range zr.File {
		name := strings.TrimPrefix(strings.ReplaceAll(f.Name, "\\", "/"), "./")
		if name == "" || strings.HasPrefix(name, "_metadata/") || name == "_metadata" {
			continue
		}
		dst := filepath.Join(tmp, filepath.FromSlash(name))
		if !strings.HasPrefix(dst, filepath.Clean(tmp)+string(os.PathSeparator)) {
			_ = os.RemoveAll(tmp)
			return man, fmt.Errorf("扩展包包含非法路径 %q", f.Name)
		}
		if f.FileInfo().IsDir() || strings.HasSuffix(name, "/") {
			_ = os.MkdirAll(dst, 0o755)
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > 512<<20 {
			_ = os.RemoveAll(tmp)
			return man, fmt.Errorf("扩展解压后超过 512 MB")
		}
		_ = os.MkdirAll(filepath.Dir(dst), 0o755)
		rc, err := f.Open()
		if err != nil {
			_ = os.RemoveAll(tmp)
			return man, err
		}
		out, err := os.Create(dst)
		if err != nil {
			rc.Close()
			_ = os.RemoveAll(tmp)
			return man, err
		}
		_, err = io.Copy(out, io.LimitReader(rc, 512<<20))
		rc.Close()
		out.Close()
		if err != nil {
			_ = os.RemoveAll(tmp)
			return man, err
		}
	}
	if man.Key == "" && len(info.PubKey) > 0 {
		k, _ := json.Marshal(base64.StdEncoding.EncodeToString(info.PubKey))
		raw["key"] = k
		b, _ := json.MarshalIndent(raw, "", "  ")
		if err := os.WriteFile(filepath.Join(tmp, "manifest.json"), b, 0o644); err != nil {
			_ = os.RemoveAll(tmp)
			return man, err
		}
		man.Key = string(k)
	}
	_ = os.RemoveAll(dir)
	if err := os.Rename(tmp, dir); err != nil {
		_ = os.RemoveAll(tmp)
		return man, err
	}
	return man, nil
}

// extPath resolves a manifest-relative path inside an unpacked extension and
// refuses anything that would leave its folder (manifest values are untrusted:
// an icon of "../../data.json" must not read the app's own files).
func extPath(dir, rel string) (string, bool) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if rel == "" || strings.Contains(rel, ":") || filepath.IsAbs(rel) {
		return "", false
	}
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if !strings.HasPrefix(p, filepath.Clean(dir)+string(os.PathSeparator)) {
		return "", false
	}
	return p, true
}

// localizedName resolves "__MSG_x__" through _locales (Chinese first, then the default locale).
func localizedName(dir string, man extManifest, s string) string {
	if !strings.HasPrefix(s, "__MSG_") || !strings.HasSuffix(s, "__") {
		return s
	}
	key := strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(s, "__MSG_"), "__"))
	for _, loc := range []string{"zh_CN", "zh", man.DefaultLocale, "en", "en_US"} {
		if loc == "" {
			continue
		}
		mp, ok := extPath(dir, "_locales/"+loc+"/messages.json")
		if !ok {
			continue
		}
		b, err := os.ReadFile(mp)
		if err != nil {
			continue
		}
		var msgs map[string]struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &msgs) != nil {
			continue
		}
		for k, v := range msgs {
			if strings.ToLower(k) == key && v.Message != "" {
				return v.Message
			}
		}
	}
	return s
}

func iconDataURL(dir string, icons map[string]string) string {
	best, bestN := "", 0
	for k, v := range icons {
		var n int
		fmt.Sscanf(k, "%d", &n)
		if n > bestN && n <= 256 {
			best, bestN = v, n
		}
	}
	if best == "" {
		return ""
	}
	p, ok := extPath(dir, strings.TrimPrefix(best, "/"))
	if !ok {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil || len(b) > 256<<10 {
		return ""
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(p)) {
	case ".svg":
		mime = "image/svg+xml"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}

// installPackage unpacks a CRX/zip into the registry. wantID is the store ID the
// user asked for ("" for URL/file installs).
func (r *extRegistry) installPackage(b []byte, wantID, source, origin string) (Extension, error) {
	info, err := parseCRX(b)
	if err != nil {
		return Extension{}, err
	}
	id := wantID
	if id == "" {
		id = info.CrxID
	}
	if id == "" {
		// Plain zip: use the manifest key when present, else a stable ID from the origin.
		if zr, e := zip.NewReader(bytes.NewReader(info.Zip), int64(len(info.Zip))); e == nil {
			if man, _, e := readManifest(zr); e == nil && man.Key != "" {
				if k, e := base64.StdEncoding.DecodeString(man.Key); e == nil {
					id = extIDFromKey(k)
				}
			}
		}
		if id == "" {
			sum := sha256.Sum256([]byte(origin))
			id = encodeExtID(sum[:16])
		}
	}
	if wantID != "" && info.CrxID != "" && info.CrxID != wantID {
		return Extension{}, fmt.Errorf("下载到的扩展 ID（%s）与请求的 ID 不一致", info.CrxID)
	}
	dir := filepath.Join(r.dir(), id)
	man, err := unpackExtension(info, dir)
	if err != nil {
		return Extension{}, err
	}
	e := Extension{ID: id, Name: localizedName(dir, man, man.Name), Version: man.Version, Description: localizedName(dir, man, man.Description),
		Source: source, Origin: origin, Dir: dir, Icon: iconDataURL(dir, man.Icons), MV: man.MV, Size: int64(len(b)), AddedAt: time.Now().Unix()}
	if e.Name == "" {
		e.Name = id
	}
	return e, r.put(e)
}

// ---------- download ----------

var storeIDInURL = regexp.MustCompile(`(?:^|/)([a-p]{32})(?:[/?#]|$)`)

// normalizeExtID accepts a bare ID or a Chrome Web Store URL.
func normalizeExtID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if extIDRe.MatchString(s) {
		return s, nil
	}
	if m := storeIDInURL.FindStringSubmatch(s); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("扩展 ID 应为 32 位 a-p 字母，或粘贴 Chrome 应用商店的扩展页面链接")
}

func crxUpdateURL(id, prodversion string) string {
	if prodversion == "" {
		prodversion = "131.0.0.0"
	}
	return "https://clients2.google.com/service/update2/crx?response=redirect&prodversion=" + url.QueryEscape(prodversion) +
		"&acceptformat=crx2,crx3&x=" + url.QueryEscape("id="+id+"&installsource=ondemand&uc")
}

// extHTTP is replaceable in tests.
var extHTTP = func(proxy *url.URL) *http.Client {
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if proxy != nil {
		tr.Proxy = http.ProxyURL(proxy)
	}
	return &http.Client{Timeout: 90 * time.Second, Transport: tr}
}

func fetchPackage(u string, proxy *url.URL) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	resp, err := extHTTP(proxy).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, fmt.Errorf("应用商店没有返回扩展（HTTP 204）：ID 不存在、已下架，或该扩展不提供下载")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 200<<20))
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("下载到的文件为空：ID 可能不存在")
	}
	return b, nil
}

// downloadWithFallback tries a direct download, then through the Mihomo
// listener when one is known (the store is unreachable from some networks).
func downloadWithFallback(u string) ([]byte, error) {
	b, err := fetchPackage(u, nil)
	if err == nil || strings.Contains(err.Error(), "HTTP 204") {
		return b, err
	}
	if px := mihomoListenerURL(); px != nil {
		if b2, err2 := fetchPackage(u, px); err2 == nil {
			return b2, nil
		}
	}
	return nil, fmt.Errorf("%w。若无法直连 Google，请先在「代理 → Clash / Mihomo」连接本地 Mihomo，或下载 CRX 后用链接 / 本地文件添加", err)
}

func (s *Store) AddExtensionByID(raw, prodversion string) (Extension, error) {
	id, err := normalizeExtID(raw)
	if err != nil {
		return Extension{}, err
	}
	b, err := downloadWithFallback(crxUpdateURL(id, prodversion))
	if err != nil {
		return Extension{}, err
	}
	return extReg.installPackage(b, id, "store", id)
}

func (s *Store) AddExtensionFromURL(raw string) (Extension, error) {
	raw = strings.Trim(strings.TrimSpace(raw), "\"")
	if raw == "" {
		return Extension{}, fmt.Errorf("请填写 CRX 链接或本地文件路径")
	}
	if strings.Contains(raw, "chromewebstore.google.com") || strings.Contains(raw, "chrome.google.com/webstore") {
		if id, err := normalizeExtID(raw); err == nil {
			return s.AddExtensionByID(id, "")
		}
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		b, err := downloadWithFallback(raw)
		if err != nil {
			return Extension{}, err
		}
		return extReg.installPackage(b, "", "url", raw)
	}
	path := strings.TrimPrefix(raw, "file://")
	if strings.HasPrefix(path, "/") && len(path) > 2 && path[2] == ':' { // file:///C:/...
		path = path[1:]
	}
	st, err := os.Stat(path)
	if err != nil {
		return Extension{}, fmt.Errorf("找不到文件 %s", path)
	}
	if st.IsDir() {
		return Extension{}, fmt.Errorf("请选择 .crx 或 .zip 文件，而不是文件夹")
	}
	if st.Size() > 200<<20 {
		return Extension{}, fmt.Errorf("文件超过 200 MB")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Extension{}, err
	}
	return extReg.installPackage(b, "", "file", path)
}

// extensionDirs returns the unpacked folders assigned to a profile.
// Missing folders (an extension removed from disk) are skipped.
func extensionDirs(p *Profile) []string {
	var dirs []string
	for _, id := range p.ExtensionIDs {
		e, ok := extReg.get(id)
		if !ok || e.Dir == "" {
			continue
		}
		if fileExists(filepath.Join(e.Dir, "manifest.json")) {
			dirs = append(dirs, e.Dir)
		}
	}
	return dirs
}

// withExtensionArgs adds --load-extension=<dirs> and turns off Chrome 137+'s
// DisableLoadExtensionCommandLineSwitch, merging it into an existing
// --disable-features value instead of adding a second flag (Chromium only
// honours the last one). args is not modified in place.
func withExtensionArgs(args []string, dirs []string) []string {
	if len(dirs) == 0 {
		return args
	}
	out := make([]string, 0, len(args)+2)
	merged := false
	for _, a := range args {
		if strings.HasPrefix(a, "--load-extension=") {
			continue
		}
		if v, ok := strings.CutPrefix(a, "--disable-features="); ok && !merged {
			if !strings.Contains(","+v+",", ",DisableLoadExtensionCommandLineSwitch,") {
				if v != "" {
					v += ","
				}
				v += "DisableLoadExtensionCommandLineSwitch"
			}
			a, merged = "--disable-features="+v, true
		}
		out = append(out, a)
	}
	if !merged {
		out = append(out, "--disable-features=DisableLoadExtensionCommandLineSwitch")
	}
	return append(out, "--load-extension="+strings.Join(dirs, ","))
}
