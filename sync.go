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
	base, token, device, lockToken string
	hc                             *http.Client
}

func (s *Store) sync() *syncClient {
	st := s.GetSettings()
	return &syncClient{base: strings.TrimRight(st.SyncServer, "/"), token: st.SyncToken, device: st.DeviceID, hc: &http.Client{Timeout: 5 * time.Minute}}
}

func (c *syncClient) req(method, path string, body io.Reader, ct string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if c.device != "" {
		req.Header.Set("X-Device-ID", c.device)
	}
	if c.lockToken != "" {
		req.Header.Set("X-Lock-Token", c.lockToken)
	}
	return c.hc.Do(req)
}

func syncStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if resp.StatusCode == 412 {
		return fmt.Errorf("conflict: a newer cloud version exists; pull first")
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
	_, dir, e := s.syncDir(id)
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
	raw, e := zipDir(dir, &archived)
	if e != nil {
		return e
	}
	enc, e := encrypt(raw, []byte(st.SyncPassword))
	if e != nil {
		return e
	}
	sum := sha256.Sum256(enc)
	c := s.sync()
	path := "/api/profiles/" + id
	var lease struct {
		Token   string `json:"lockToken"`
		Version int64  `json:"version"`
	}
	if e = c.jsonRequest("POST", path+"/lock", map[string]string{"device": st.DeviceID}, &lease); e != nil {
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
	if lease.Version != p.CloudVer {
		return fmt.Errorf("conflict: cloud is v%d, local is v%d; pull first", lease.Version, p.CloudVer)
	}
	resp, e := c.req("PUT", path+"/blob?version="+strconv.FormatInt(p.CloudVer, 10), bytes.NewReader(enc), "application/octet-stream")
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
	if e = c.jsonRequest("POST", path+"/commit", cm, &result); e != nil {
		return fmt.Errorf("commit: %w", e)
	}
	if !result.OK || result.Version != cm.Version {
		return fmt.Errorf("server returned an invalid commit acknowledgement")
	}
	committed = true
	p.Storage = "cloud"
	p.CloudVer = cm.Version
	p.SyncedAt = time.Now().Unix()
	s.PutProfile(*p)
	return nil
}

func (s *Store) PullProfile(id string) error {
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
	resp, e := c.req("GET", "/api/profiles/"+id+"/blob", nil, "")
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
	enc, e := io.ReadAll(io.LimitReader(resp.Body, maxSyncBlob+1))
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
	if e = unzipDir(raw, stage); e != nil {
		return e
	}
	meta, e := os.ReadFile(filepath.Join(stage, profileArchiveMetadata))
	if e != nil {
		return fmt.Errorf("archive has no encrypted profile metadata; re-upload with the new client")
	}
	var p Profile
	if e = json.Unmarshal(meta, &p); e != nil {
		return e
	}
	if p.ID != id {
		return fmt.Errorf("archive profile ID mismatch")
	}
	p.Storage = "cloud"
	p.CloudVer = version
	p.SyncedAt = time.Now().Unix()
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
	// Keep the previous environment as a named backup, rather than erasing it.
	return nil
}

func (s *Store) ListCloud() ([]cloudMeta, error) {
	if e := syncReady(s.GetSettings(), false); e != nil {
		return nil, e
	}
	var out []cloudMeta
	e := s.sync().jsonRequest("GET", "/api/profiles", nil, &out)
	if out == nil {
		out = []cloudMeta{}
	}
	return out, e
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
