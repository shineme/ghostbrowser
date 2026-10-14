package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
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
	"runtime"
	"strconv"
	"strings"
	"time"

	"ghostbrowser/internal/undetectable"
	"golang.org/x/crypto/pbkdf2"
)

const maxSyncBlob = 256 << 20
const maxUnpackedSync = 2 << 30
const profileArchiveMetadata = "ghost-profile.json"

var syncIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func deriveKey(pass, salt []byte) []byte { return pbkdf2.Key(pass, salt, 120000, 32, sha256.New) }

func encrypt(plain, pass []byte) ([]byte, error) {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return nil, e
	}
	block, e := aes.NewCipher(deriveKey(pass, salt))
	if e != nil {
		return nil, e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return nil, e
	}
	out := append([]byte("GBE1"), salt...)
	out = append(out, nonce...)
	return gcm.Seal(out, nonce, plain, nil), nil
}

func decrypt(blob, pass []byte) ([]byte, error) {
	if len(blob) < 48 || string(blob[:4]) != "GBE1" {
		return nil, fmt.Errorf("bad encrypted blob header")
	}
	block, e := aes.NewCipher(deriveKey(pass, blob[4:20]))
	if e != nil {
		return nil, e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	return gcm.Open(nil, blob[20:32], blob[32:], nil)
}

func zipDir(dir string, p *Profile) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	excluded := map[string]bool{"cache": true, "code cache": true, "gpucache": true, "shadercache": true, "graphitedawncache": true, "dawnwebgpucache": true, "component_crx_cache": true, "_fp_ext": true, "_runtime": true, "crashpad": true}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		low := strings.ToLower(info.Name())
		if info.IsDir() {
			if excluded[low] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if strings.EqualFold(rel, profileArchiveMetadata) || low == "launch.json" || low == "native-launch.lock" || strings.HasPrefix(low, "singleton") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			f.Close()
			return err
		}
		_, err = io.Copy(w, f)
		ce := f.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		if buf.Len() > maxSyncBlob {
			return fmt.Errorf("profile exceeds the 256 MiB sync limit")
		}
		return nil
	})
	if err != nil {
		zw.Close()
		return nil, err
	}
	w, err := zw.Create(profileArchiveMetadata)
	if err != nil {
		zw.Close()
		return nil, err
	}
	if err = json.NewEncoder(w).Encode(p); err != nil {
		zw.Close()
		return nil, err
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() > maxSyncBlob-48 {
		return nil, fmt.Errorf("profile exceeds the 256 MiB sync limit")
	}
	return buf.Bytes(), nil
}

func unzipDir(blob []byte, dir string) error {
	zr, err := zip.NewReader(bytes.NewReader(blob), int64(len(blob)))
	if err != nil {
		return err
	}
	if len(zr.File) > 100000 {
		return fmt.Errorf("too many archive entries")
	}
	var total uint64
	for _, f := range zr.File {
		name := f.Name
		if !filepath.IsLocal(name) || strings.Contains(name, "\\") || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe archive path: %q", name)
		}
		if f.UncompressedSize64 > maxUnpackedSync || total > maxUnpackedSync-f.UncompressedSize64 {
			return fmt.Errorf("archive exceeds the 2 GiB unpacked limit")
		}
		total += f.UncompressedSize64
		dst := filepath.Join(dir, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(dst, 0700); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			rc.Close()
			return err
		}
		n, err := io.Copy(out, io.LimitReader(rc, int64(f.UncompressedSize64)+1))
		ce := out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if ce != nil {
			return ce
		}
		if uint64(n) != f.UncompressedSize64 {
			return fmt.Errorf("archive entry size mismatch")
		}
	}
	return nil
}

type syncClient struct {
	base, token, device, lockToken, deviceName string
	hc                                         *http.Client
}

func (s *Store) sync() *syncClient {
	st := s.GetSettings()
	return &syncClient{base: strings.TrimRight(st.SyncServer, "/"), token: st.SyncToken, device: st.DeviceID, deviceName: st.DeviceName, hc: &http.Client{Timeout: 5 * time.Minute}}
}

func (c *syncClient) req(method, path string, body io.Reader, ct string) (*http.Response, error) {
	req, err := http.NewRequest(method, syncURL(c.base, path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Ghost-Token", c.token) // some PHP hosts strip Authorization
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if c.device != "" {
		req.Header.Set("X-Device-ID", c.device)
	}
	if c.deviceName != "" {
		req.Header.Set("X-Device-Name", url.QueryEscape(c.deviceName))
	}
	if c.lockToken != "" {
		req.Header.Set("X-Lock-Token", c.lockToken)
	}
	return c.hc.Do(req)
}

// syncURL joins the server address and an API path. A server address that
// points at a .php file (the single-file server, no URL rewrite needed) gets
// the path as ?r=/api/...
func syncURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Path), ".php") {
		return base + path
	}
	p, q, _ := strings.Cut(path, "?")
	v := u.Query()
	v.Set("r", p)
	if q != "" {
		extra, _ := url.ParseQuery(q)
		for k, vs := range extra {
			for _, x := range vs {
				v.Add(k, x)
			}
		}
	}
	u.RawQuery = v.Encode()
	return u.String()
}

// errBodyTooLarge is a 413. front=true means the rejection came from a web
// server or CDN in front of PHP (nginx client_max_body_size, Cloudflare),
// which ping's maxBody cannot see.
type errBodyTooLarge struct{ front bool }

func (e errBodyTooLarge) Error() string {
	if e.front {
		return "上传被服务器前端拒绝（HTTP 413，请求体过大）：这是 nginx/CDN 的上传上限，不是 PHP 的。请在站点 nginx 配置里加 client_max_body_size 64m;（宝塔：网站→设置→配置文件，或 软件商店→Nginx→设置→性能调整），然后重载 nginx"
	}
	return "上传超过 ghostsync.php 的单次上限（HTTP 413）：请调大 PHP post_max_size / upload_max_filesize，或 GHOST_SYNC_MAX_BLOB_MB"
}

func isTooLarge(e error) bool {
	var t errBodyTooLarge
	return errors.As(e, &t)
}

func syncStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode == 412 {
		return fmt.Errorf("conflict: a newer cloud version exists; pull first")
	}
	if resp.StatusCode == 413 {
		return errBodyTooLarge{front: !strings.Contains(string(b), `"error"`)}
	}
	if resp.StatusCode == 410 || resp.StatusCode == 423 {
		return cloudStatusErr(resp.StatusCode, b)
	}
	return fmt.Errorf("sync HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
}

func (c *syncClient) jsonRequest(method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	resp, e := c.req(method, path, body, "application/json")
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return e
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
	}
	return nil
}

type cloudMeta struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Version int64           `json:"version"`
	Sha256  string          `json:"sha256"`
	Meta    json.RawMessage `json:"meta"`
	Locked  string          `json:"lockedBy"`
	Updated int64           `json:"updatedAt"`
	Size    int64           `json:"size"`
	Device  string          `json:"deviceName"`
	// Tombstones (?tombstones=1): the profile was deleted from the cloud.
	Deleted       bool   `json:"deleted,omitempty"`
	DeletedAt     int64  `json:"deletedAt,omitempty"`
	DeletedByName string `json:"deletedByName,omitempty"`
	DeletedBy     string `json:"deletedBy,omitempty"`
}

func syncReady(st Settings, password bool) error {
	u, e := url.Parse(st.SyncServer)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("set a valid sync server base URL in Settings")
	}
	if st.SyncToken == "" {
		return fmt.Errorf("set the sync access token in Settings")
	}
	if password && st.SyncPassword == "" {
		return fmt.Errorf("set the sync encryption password in Settings")
	}
	return nil
}

// All staging and replacement targets must stay inside the resolved profile
// root. In particular an existing junction cannot redirect a restore elsewhere.
func (s *Store) syncDir(id string) (string, string, error) {
	if !syncIDPattern.MatchString(id) {
		return "", "", fmt.Errorf("invalid profile ID")
	}
	configured := s.GetSettings().ProfilesDir
	if configured == "" {
		return "", "", fmt.Errorf("profiles directory is empty")
	}
	root, e := filepath.Abs(configured)
	if e != nil {
		return "", "", e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return "", "", e
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		return "", "", e
	}
	dir := filepath.Join(root, id)
	if fi, e := os.Lstat(dir); e == nil {
		resolved, e := filepath.EvalSymlinks(dir)
		if e != nil {
			return "", "", e
		}
		rel, e := filepath.Rel(root, resolved)
		if e != nil || !strings.EqualFold(rel, id) || !fi.IsDir() {
			return "", "", fmt.Errorf("profile target does not stay inside its configured root")
		}
	} else if !os.IsNotExist(e) {
		return "", "", e
	}
	return root, dir, nil
}

func syncFileLock(dir string) (func(), error) {
	if runtime.GOOS != "windows" {
		return func() {}, nil
	}
	f, e := undetectable.AcquireProfileData(dir)
	if e != nil {
		return nil, e
	}
	return func() { f.Close() }, nil
}

func (s *Store) PushProfile(id string) error {
	t0 := time.Now()
	n, e := s.pushProfile(id)
	s.logSync(id, "push", n, t0, e)
	return e
}

func (s *Store) pushProfile(id string) (sent int64, err error) {
	t := s.newTransfer(id, "up")
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("sync crashed: %v", r)
		}
		t.finish(err)
	}()
	err = s.pushProfileInner(id, &sent, t)
	return sent, err
}

func (s *Store) pushProfileInner(id string, sent *int64, t *transfer) error {
	s.launchMu.Lock()
	defer s.launchMu.Unlock()
	if _, ok := runReg.get(id); ok {
		return fmt.Errorf("stop the profile before syncing its browser databases")
	}
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return e
	}
	p, ok := s.GetProfile(id)
	if !ok {
		return fmt.Errorf("profile not found")
	}
	root, dir, e := s.syncDir(id)
	if e != nil {
		return e
	}
	release, e := syncFileLock(dir)
	if e != nil {
		return e
	}
	defer release()
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	// Put all private metadata inside the encrypted archive. Proxy references on
	// another device would not resolve, so archive the resolved inline proxy.
	pb, _ := json.Marshal(p)
	var archived Profile
	_ = json.Unmarshal(pb, &archived)
	if archived.InlineProxy == nil && archived.ProxyID != "" {
		if px, ok := s.GetProxy(archived.ProxyID); ok {
			archived.InlineProxy = px
			archived.ProxyID = ""
		}
	}
	var enc []byte
	var refs []string
	var files []manifestFile
	if si, err := s.ServerInfo(false); err == nil && si.has("chunks") {
		enc, refs, *sent, files, e = s.buildChunked(dir, root, &archived, st.SyncPassword, t)
		if e != nil {
			return e
		}
		if refs == nil {
			refs = []string{}
		}
	} else {
		t.phase("pack", 0)
		raw, e := zipDir(dir, &archived)
		if e != nil {
			return e
		}
		enc, e = encrypt(raw, []byte(st.SyncPassword))
		if e != nil {
			return e
		}
		if err == nil && si.MaxBody > 0 && int64(len(enc)) > si.MaxBody {
			return fmt.Errorf("同步包 %d MB 超过服务器单次上传上限 %d MB：请调大 PHP post_max_size / upload_max_filesize，或升级到支持增量同步的 ghostsync.php", len(enc)>>20, si.MaxBody>>20)
		}
		*sent = int64(len(enc))
	}
	sum := sha256.Sum256(enc)
	c := s.sync()
	path := "/api/profiles/" + id
	var lease struct {
		Token   string `json:"lockToken"`
		Version int64  `json:"version"`
	}
	_, restore := restorePush.LoadAndDelete(id)
	if e = c.jsonRequest("POST", path+"/lock", map[string]any{"device": st.DeviceID, "restore": restore}, &lease); e != nil {
		return fmt.Errorf("lock: %w", e)
	}
	if lease.Token == "" {
		return fmt.Errorf("sync server did not issue a lock token")
	}
	c.lockToken = lease.Token
	committed := false
	defer func() {
		if !committed {
			_ = c.jsonRequest("POST", path+"/unlock", map[string]any{}, nil)
		}
	}()
	if restore && lease.Version > p.CloudVer {
		p.CloudVer = lease.Version // recreate over the tombstone
	}
	if lease.Version != p.CloudVer {
		return fmt.Errorf("conflict: cloud is v%d, local is v%d; pull first", lease.Version, p.CloudVer)
	}
	if refs != nil {
		rr, e := c.req("PUT", path+"/refs", strings.NewReader(strings.Join(refs, "\n")), "text/plain")
		if e != nil {
			return e
		}
		e = syncStatus(rr)
		rr.Body.Close()
		if e != nil {
			return e
		}
	}
	var blobBody io.Reader = bytes.NewReader(enc)
	if files == nil {
		t.phase("upload", int64(len(enc)))
		blobBody = &countingReader{r: blobBody, fn: t.add}
	}
	resp, e := c.req("PUT", path+"/blob?version="+strconv.FormatInt(p.CloudVer, 10), blobBody, "application/octet-stream")
	if e != nil {
		return e
	}
	e = syncStatus(resp)
	resp.Body.Close()
	if e != nil {
		return e
	}
	marker := json.RawMessage(`{"encrypted":true,"format":"GBE1","profileMetadata":"in-blob"}`)
	cm := cloudMeta{ID: id, Name: p.Name, Version: p.CloudVer + 1, Sha256: hex.EncodeToString(sum[:]), Meta: marker}
	var result struct {
		OK      bool  `json:"ok"`
		Version int64 `json:"version"`
	}
	t.phase("commit", 0)
	if e = c.jsonRequest("POST", path+"/commit", cm, &result); e != nil {
		return fmt.Errorf("commit: %w", e)
	}
	if !result.OK || result.Version != cm.Version {
		return fmt.Errorf("server returned an invalid commit acknowledgement")
	}
	committed = true
	if files != nil {
		saveSyncCache(root, id, files)
		p.ChunkKey = archived.ChunkKey
	} else {
		dropSyncCache(root, id)
	}
	p.Storage = "cloud"
	p.CloudVer = cm.Version
	p.SyncedAt = syncStamp(p.LastRun)
	s.PutProfile(*p)
	return nil
}

func (s *Store) PullProfile(id string) error {
	t0 := time.Now()
	n, e := s.pullProfile(id, 0)
	s.logSync(id, "pull", n, t0, e)
	return e
}

// pullProfile restores the cloud copy (version 0 = current, else a kept older
// version) and returns the bytes downloaded.
func (s *Store) pullProfile(id string, want int64) (got int64, err error) {
	t := s.newTransfer(id, "down")
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("sync crashed: %v", r)
		}
		t.finish(err)
	}()
	err = s.pullProfileInner(id, want, &got, t)
	return got, err
}

func (s *Store) pullProfileInner(id string, want int64, got *int64, t *transfer) error {
	s.launchMu.Lock()
	defer s.launchMu.Unlock()
	if _, ok := runReg.get(id); ok {
		return fmt.Errorf("stop the profile before restoring its browser data")
	}
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return e
	}
	root, dir, e := s.syncDir(id)
	if e != nil {
		return e
	}
	release, e := syncFileLock(dir)
	if e != nil {
		return e
	}
	defer release()
	c := s.sync()
	blobPath := "/api/profiles/" + id + "/blob"
	if want > 0 {
		blobPath += "?version=" + strconv.FormatInt(want, 10)
	}
	resp, e := c.req("GET", blobPath, nil, "")
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return e
	}
	version, e := strconv.ParseInt(resp.Header.Get("X-Profile-Version"), 10, 64)
	if e != nil || version < 1 {
		return fmt.Errorf("sync server returned no valid profile version")
	}
	wantHash := resp.Header.Get("X-Blob-SHA256")
	if len(wantHash) != 64 {
		return fmt.Errorf("sync server returned no valid blob digest")
	}
	if resp.ContentLength > 0 {
		t.phase("download", resp.ContentLength)
	}
	enc, e := io.ReadAll(io.LimitReader(&countingReader{r: resp.Body, fn: t.add}, maxSyncBlob+1))
	if e != nil {
		return e
	}
	if len(enc) > maxSyncBlob {
		return fmt.Errorf("cloud blob exceeds 256 MiB")
	}
	sum := sha256.Sum256(enc)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), wantHash) {
		return fmt.Errorf("download SHA-256 does not match")
	}
	raw, e := decrypt(enc, []byte(st.SyncPassword))
	if e != nil {
		return fmt.Errorf("decrypt failed; check the sync password: %w", e)
	}
	stage, e := os.MkdirTemp(root, ".pull-"+id+"-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	*got = int64(len(enc))
	var p Profile
	man, chunked, e := decodeManifest(raw)
	if e != nil {
		return e
	}
	if chunked {
		n, e := s.restoreChunked(man, dir, root, stage, t)
		*got += n
		if e != nil {
			return e
		}
		p = man.Profile
		p.ChunkKey = man.ChunkKey
	} else {
		t.phase("write", 0)
		if e = unzipDir(raw, stage); e != nil {
			return e
		}
		meta, e := os.ReadFile(filepath.Join(stage, profileArchiveMetadata))
		if e != nil {
			return fmt.Errorf("archive has no encrypted profile metadata; re-upload with the new client")
		}
		if e = json.Unmarshal(meta, &p); e != nil {
			return e
		}
		_ = os.Remove(filepath.Join(stage, profileArchiveMetadata))
	}
	if p.ID != id {
		return fmt.Errorf("archive profile ID mismatch")
	}
	p.Storage = "cloud"
	p.CloudVer = version
	p.KeepLocal = 0
	p.SyncedAt = syncStamp(p.LastRun)
	backup := ""
	if _, e = os.Stat(dir); e == nil {
		backup, e = os.MkdirTemp(root, ".before-pull-"+id+"-")
		if e != nil {
			return e
		}
		if e = os.Remove(backup); e != nil {
			return e
		}
		if e = os.Rename(dir, backup); e != nil {
			return fmt.Errorf("cannot stage current environment: %w", e)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if e = os.Rename(stage, dir); e != nil {
		if backup != "" {
			_ = os.Rename(backup, dir)
		}
		return fmt.Errorf("restore failed: %w", e)
	}
	s.PutProfile(p)
	s.clearLocalDeleted(id) // restored on purpose: auto sync may follow it again
	if chunked {
		saveSyncCache(root, id, man.Files)
	} else {
		dropSyncCache(root, id)
	}
	// Keep the previous environment as a named backup (only the newest one).
	if backup != "" {
		old, _ := filepath.Glob(filepath.Join(root, ".before-pull-"+id+"-*"))
		for _, o := range old {
			if o != backup {
				_ = os.RemoveAll(o)
			}
		}
	}
	return nil
}

func (s *Store) ListCloud() ([]cloudMeta, error) {
	if e := syncReady(s.GetSettings(), false); e != nil {
		return nil, e
	}
	all, e := s.listCloudAll()
	out := []cloudMeta{}
	for _, c := range all {
		if c.ID != globalID {
			out = append(out, c)
		}
	}
	return out, e
}

// listCloudAll includes the reserved global-data record (live entries only).
func (s *Store) listCloudAll() ([]cloudMeta, error) {
	live, _, e := s.listCloudFull()
	return live, e
}

func (s *Store) TestSync() error {
	if e := syncReady(s.GetSettings(), false); e != nil {
		return e
	}
	var out struct {
		OK      bool   `json:"ok"`
		Service string `json:"service"`
	}
	if e := s.sync().jsonRequest("GET", "/api/ping", nil, &out); e != nil {
		return e
	}
	if !out.OK || out.Service != "ghostbrowser-sync" {
		return fmt.Errorf("the URL is not a GhostBrowser sync server")
	}
	return nil
}
