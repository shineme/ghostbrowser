package main

// Incremental sync. A profile is stored on the server as
//   - an encrypted manifest (the usual GBE1 blob, password-derived key) that
//     lists every file with the IDs of its chunks, plus the profile metadata
//     and a random per-profile chunk key;
//   - content-addressed chunks: each file is cut into pieces of up to 4 MiB,
//     the piece's ID is HMAC-SHA256(chunk key, plaintext) and its body is
//     "GBC1" + nonce + AES-256-GCM(deflate(plaintext)).
// Only chunks the server does not have yet are uploaded, and a pull reuses
// every piece already present in the local profile, so after the first sync
// closing a browser usually moves a few MB instead of the whole profile.

import (
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const chunkSize = 4 << 20
const chunkBatchBytes = 12 << 20 // per upload request
const chunkGetBatch = 64         // chunks per download request
const maxManifestFiles = 200000

type manifestFile struct {
	Path   string   `json:"p"`
	Size   int64    `json:"s"`
	MTime  int64    `json:"m"` // unix nanoseconds
	Chunks []string `json:"c"`
}

type syncManifest struct {
	Format   string         `json:"format"` // GBM1
	Profile  Profile        `json:"profile"`
	ChunkKey string         `json:"chunkKey"` // base64, 32 bytes
	Files    []manifestFile `json:"files"`
	Total    int64          `json:"total"`
}

// Local cache of what the last sync saw, so unchanged files are not re-hashed.
type syncCache struct {
	Files map[string]manifestFile `json:"files"`
}

// ---------------------------------------------------------------- features

type serverInfo struct {
	Features     []string `json:"features"`
	KeepVersions int      `json:"keepVersions"`
	MaxBody      int64    `json:"maxBody"`
	UsedBytes    int64    `json:"usedBytes"`
	FreeBytes    int64    `json:"freeBytes"`
	User         string   `json:"user"`
	Protocol     int      `json:"protocol"`
}

func (si serverInfo) has(f string) bool {
	for _, x := range si.Features {
		if x == f {
			return true
		}
	}
	return false
}

var infoCache struct {
	sync.Mutex
	key string
	at  time.Time
	si  serverInfo
}

// ServerInfo asks the server what it supports (cached for 5 minutes).
func (s *Store) ServerInfo(fresh bool) (serverInfo, error) {
	st := s.GetSettings()
	if e := syncReady(st, false); e != nil {
		return serverInfo{}, e
	}
	key := st.SyncServer + "\x00" + st.SyncToken
	infoCache.Lock()
	if !fresh && infoCache.key == key && time.Since(infoCache.at) < 5*time.Minute {
		si := infoCache.si
		infoCache.Unlock()
		return si, nil
	}
	infoCache.Unlock()
	var si serverInfo
	if e := s.sync().jsonRequest("GET", "/api/ping", nil, &si); e != nil {
		return si, e
	}
	infoCache.Lock()
	infoCache.key, infoCache.at, infoCache.si = key, time.Now(), si
	infoCache.Unlock()
	return si, nil
}

// ---------------------------------------------------------------- walking

var syncExcludedDirs = map[string]bool{"cache": true, "code cache": true, "gpucache": true, "shadercache": true, "graphitedawncache": true, "dawnwebgpucache": true, "component_crx_cache": true, "_fp_ext": true, "_runtime": true, "crashpad": true}

func syncSkipFile(rel, low string) bool {
	return strings.EqualFold(rel, profileArchiveMetadata) || low == "launch.json" || low == "native-launch.lock" || strings.HasPrefix(low, "singleton")
}

// walkSyncFiles calls fn for every regular file that belongs in a sync.
func walkSyncFiles(dir string, fn func(rel string, info os.FileInfo) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path != dir {
				return nil // vanished while walking
			}
			return err
		}
		if path == dir || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		low := strings.ToLower(info.Name())
		if info.IsDir() {
			if syncExcludedDirs[low] {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if syncSkipFile(rel, low) {
			return nil
		}
		return fn(filepath.ToSlash(rel), info)
	})
}

// ---------------------------------------------------------------- crypto

func chunkID(key, data []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return hex.EncodeToString(m.Sum(nil))
}

func sealChunk(key, data []byte) ([]byte, error) {
	var z bytes.Buffer
	zw, _ := flate.NewWriter(&z, flate.BestSpeed)
	if _, e := zw.Write(data); e != nil {
		return nil, e
	}
	if e := zw.Close(); e != nil {
		return nil, e
	}
	block, e := aes.NewCipher(key)
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
	out := append([]byte("GBC1"), nonce...)
	return gcm.Seal(out, nonce, z.Bytes(), nil), nil
}

func openChunk(key []byte, id string, blob []byte) ([]byte, error) {
	if len(blob) < 32 || string(blob[:4]) != "GBC1" {
		return nil, fmt.Errorf("bad chunk header")
	}
	block, e := aes.NewCipher(key)
	if e != nil {
		return nil, e
	}
	gcm, e := cipher.NewGCM(block)
	if e != nil {
		return nil, e
	}
	z, e := gcm.Open(nil, blob[4:16], blob[16:], nil)
	if e != nil {
		return nil, fmt.Errorf("chunk %s failed to decrypt: %w", id[:12], e)
	}
	data, e := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(z)), chunkSize+1))
	if e != nil {
		return nil, e
	}
	if len(data) > chunkSize || chunkID(key, data) != id {
		return nil, fmt.Errorf("chunk %s failed its integrity check", id[:12])
	}
	return data, nil
}

func profileChunkKey(p *Profile) ([]byte, error) {
	if p.ChunkKey != "" {
		if k, e := base64.StdEncoding.DecodeString(p.ChunkKey); e == nil && len(k) == 32 {
			return k, nil
		}
	}
	k := make([]byte, 32)
	if _, e := rand.Read(k); e != nil {
		return nil, e
	}
	p.ChunkKey = base64.StdEncoding.EncodeToString(k)
	return k, nil
}

// ---------------------------------------------------------------- cache

func syncCachePath(root, id string) string { return filepath.Join(root, ".syncmeta", id+".json") }

func loadSyncCache(root, id string) syncCache {
	c := syncCache{Files: map[string]manifestFile{}}
	if b, e := os.ReadFile(syncCachePath(root, id)); e == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.Files == nil {
		c.Files = map[string]manifestFile{}
	}
	return c
}

func saveSyncCache(root, id string, files []manifestFile) {
	c := syncCache{Files: map[string]manifestFile{}}
	for _, f := range files {
		c.Files[f.Path] = f
	}
	b, _ := json.Marshal(c)
	p := syncCachePath(root, id)
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	_ = os.WriteFile(p+".tmp", b, 0o600)
	_ = os.Rename(p+".tmp", p)
}

func dropSyncCache(root, id string) { _ = os.Remove(syncCachePath(root, id)) }

// ---------------------------------------------------------------- scanning

type chunkLoc struct {
	path string // absolute
	off  int64
	n    int
}

// scanProfile lists the profile's files with chunk IDs, reusing the cache for
// files whose size and modification time did not change.
func scanProfile(dir string, key []byte, cache syncCache) ([]manifestFile, map[string]chunkLoc, int64, error) {
	var files []manifestFile
	locs := map[string]chunkLoc{}
	var total int64
	buf := make([]byte, chunkSize)
	err := walkSyncFiles(dir, func(rel string, info os.FileInfo) error {
		if len(files) >= maxManifestFiles {
			return fmt.Errorf("profile has more than %d files", maxManifestFiles)
		}
		total += info.Size()
		if total > maxUnpackedSync {
			return fmt.Errorf("profile exceeds the 2 GiB sync limit")
		}
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		mf := manifestFile{Path: rel, Size: info.Size(), MTime: info.ModTime().UnixNano()}
		if c, ok := cache.Files[rel]; ok && c.Size == mf.Size && c.MTime == mf.MTime && len(c.Chunks) == int((mf.Size+chunkSize-1)/chunkSize) {
			mf.Chunks = c.Chunks
			for i, id := range mf.Chunks {
				off := int64(i) * chunkSize
				n := chunkSize
				if rem := mf.Size - off; rem < int64(n) {
					n = int(rem)
				}
				locs[id] = chunkLoc{abs, off, n}
			}
			files = append(files, mf)
			return nil
		}
		f, err := os.Open(abs)
		if err != nil {
			if os.IsNotExist(err) {
				total -= info.Size()
				return nil
			}
			return err
		}
		defer f.Close()
		var off int64
		for {
			n, err := io.ReadFull(f, buf)
			if n > 0 {
				id := chunkID(key, buf[:n])
				mf.Chunks = append(mf.Chunks, id)
				locs[id] = chunkLoc{abs, off, n}
				off += int64(n)
			}
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				return err
			}
		}
		mf.Size = off
		files = append(files, mf)
		return nil
	})
	return files, locs, total, err
}

func readLoc(l chunkLoc) ([]byte, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b := make([]byte, l.n)
	if _, err = f.ReadAt(b, l.off); err != nil && err != io.EOF {
		return nil, err
	}
	return b, nil
}

// ---------------------------------------------------------------- transport

func (c *syncClient) missingChunks(ids []string) ([]string, error) {
	var miss []string
	for i := 0; i < len(ids); i += 1000 {
		j := i + 1000
		if j > len(ids) {
			j = len(ids)
		}
		var out struct {
			Missing []string `json:"missing"`
		}
		if e := c.jsonRequest("POST", "/api/chunks/missing", map[string]any{"ids": ids[i:j]}, &out); e != nil {
			return nil, e
		}
		miss = append(miss, out.Missing...)
	}
	return miss, nil
}

func (c *syncClient) putChunks(frame []byte, onRead func(int64)) error {
	var body io.Reader = bytes.NewReader(frame)
	if onRead != nil {
		body = &countingReader{r: body, fn: onRead}
	}
	resp, e := c.req("PUT", "/api/chunks", body, "application/octet-stream")
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	return syncStatus(resp)
}

// getChunks downloads ids and hands each decrypted chunk to fn.
func (c *syncClient) getChunks(key []byte, ids []string, fn func(id string, data []byte) error, onRead func(int64)) error {
	b, _ := json.Marshal(map[string]any{"ids": ids})
	resp, e := c.req("POST", "/api/chunks/get", bytes.NewReader(b), "application/json")
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return e
	}
	var body io.Reader = resp.Body
	if onRead != nil {
		body = &countingReader{r: body, fn: onRead}
	}
	r := bufio.NewReaderSize(body, 256<<10)
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	for range ids {
		head := make([]byte, 68)
		if _, e = io.ReadFull(r, head); e != nil {
			return fmt.Errorf("chunk download truncated: %w", e)
		}
		id := string(head[:64])
		n := binary.BigEndian.Uint32(head[64:])
		if !want[id] || n > 8<<20 {
			return fmt.Errorf("unexpected chunk in download")
		}
		blob := make([]byte, n)
		if _, e = io.ReadFull(r, blob); e != nil {
			return fmt.Errorf("chunk download truncated: %w", e)
		}
		data, e := openChunk(key, id, blob)
		if e != nil {
			return e
		}
		if e = fn(id, data); e != nil {
			return e
		}
	}
	return nil
}

// ---------------------------------------------------------------- push

// chunkedUpload is what buildChunked produced: the encrypted manifest, the
// chunk list for the server's garbage collector, and a way to send again the
// chunks a commit reports missing.
type chunkedUpload struct {
	enc      []byte
	refs     []string
	sent     int64
	files    []manifestFile
	uploaded int // chunks uploaded (the server lacked them)
	// reupload asks the server again which chunks it lacks and uploads them.
	reupload func() (int, error)
}

// testHookAfterMissing runs between the missing check and the chunk upload
// (tests use it to reproduce the server's garbage collector racing a push).
var testHookAfterMissing func()

// testHookCommitRetry runs when a commit answered chunk_missing and the push
// uploads the missing chunks again.
var testHookCommitRetry func()

// buildChunked uploads the chunks the server is missing and returns the
// encrypted manifest plus the chunk list for the server's garbage collector.
func (s *Store) buildChunked(dir, root string, archived *Profile, password string, t *transfer) (*chunkedUpload, error) {
	key, e := profileChunkKey(archived)
	if e != nil {
		return nil, e
	}
	cache := loadSyncCache(root, archived.ID)
	files, locs, total, e := scanProfile(dir, key, cache)
	if e != nil {
		return nil, e
	}
	ids := make([]string, 0, len(locs))
	for id := range locs {
		ids = append(ids, id)
	}
	c := s.sync()
	miss, e := c.missingChunks(ids)
	if e != nil {
		return nil, e
	}
	if testHookAfterMissing != nil {
		testHookAfterMissing()
	}
	var upPlain int64
	for _, id := range miss {
		if l, ok := locs[id]; ok {
			upPlain += int64(l.n)
		}
	}
	t.saved(total - upPlain)
	t.phase("upload", upPlain)
	var donePlain int64
	limit := int64(chunkBatchBytes)
	if si, err := s.ServerInfo(false); err == nil && si.MaxBody > 0 && si.MaxBody-(1<<20) < limit {
		limit = si.MaxBody - (1 << 20)
		if limit < chunkSize+(256<<10) {
			return nil, fmt.Errorf("服务器单次上传上限只有 %d MB，至少需要 8 MB：请调大 PHP post_max_size / upload_max_filesize", si.MaxBody>>20)
		}
	}
	if learned := s.bodyCap.Load(); learned > 0 && learned < limit {
		limit = learned
	}
	var sent int64
	type piece struct {
		id    string
		blob  []byte
		plain int64
	}
	// send uploads a batch; on a 413 it halves the batch and remembers the
	// smaller size, so a front-end limit we cannot query is found by itself.
	var send func(ps []piece) error
	send = func(ps []piece) error {
		var frame []byte
		var plain int64
		for _, p := range ps {
			plain += p.plain
			var hdr [4]byte
			binary.BigEndian.PutUint32(hdr[:], uint32(len(p.blob)))
			frame = append(frame, p.id...)
			frame = append(frame, hdr[:]...)
			frame = append(frame, p.blob...)
		}
		var read int64
		e := c.putChunks(frame, func(n int64) {
			read += n
			t.set(donePlain + plain*read/int64(len(frame)))
		})
		if e != nil && isTooLarge(e) && len(ps) > 1 {
			if l := int64(len(frame)) / 2; l < limit {
				limit = l
				s.bodyCap.Store(l)
			}
			h := len(ps) / 2
			if e = send(ps[:h]); e != nil {
				return e
			}
			return send(ps[h:])
		}
		if e != nil {
			return e
		}
		sent += int64(len(frame))
		donePlain += plain
		t.set(donePlain)
		return nil
	}
	upload := func(miss []string) error {
		var batch []piece
		var batchLen int64
		flush := func() error {
			if len(batch) == 0 {
				return nil
			}
			e := send(batch)
			batch, batchLen = nil, 0
			return e
		}
		for _, id := range miss {
			l, ok := locs[id]
			if !ok {
				return fmt.Errorf("server asked for an unknown chunk")
			}
			data, e := readLoc(l)
			if e != nil {
				return e
			}
			if chunkID(key, data) != id {
				return fmt.Errorf("%s changed during sync; close the browser and try again", filepath.Base(l.path))
			}
			blob, e := sealChunk(key, data)
			if e != nil {
				return e
			}
			if batchLen+int64(68+len(blob)) > limit {
				if e = flush(); e != nil {
					return e
				}
			}
			batch = append(batch, piece{id, blob, int64(len(data))})
			batchLen += int64(68 + len(blob))
		}
		return flush()
	}
	if e = upload(miss); e != nil {
		return nil, e
	}
	m := syncManifest{Format: "GBM1", Profile: *archived, ChunkKey: archived.ChunkKey, Files: files, Total: total}
	raw, e := json.Marshal(m)
	if e != nil {
		return nil, e
	}
	var z bytes.Buffer
	zw, _ := flate.NewWriter(&z, flate.DefaultCompression)
	_, _ = zw.Write(raw)
	_ = zw.Close()
	enc, e := encrypt(append([]byte("GBMZ"), z.Bytes()...), []byte(password))
	if e != nil {
		return nil, e
	}
	out := &chunkedUpload{enc: enc, refs: ids, files: files, uploaded: len(miss)}
	out.reupload = func() (int, error) {
		miss, e := c.missingChunks(ids)
		if e != nil {
			return 0, e
		}
		if len(miss) == 0 {
			return 0, nil
		}
		t.phase("upload", 0)
		if e = upload(miss); e != nil {
			return 0, e
		}
		out.uploaded += len(miss)
		out.sent = sent
		return len(miss), nil
	}
	out.sent = sent
	return out, nil
}

// ---------------------------------------------------------------- pull

// decodeManifest recognises a chunked manifest; ok=false means a legacy zip.
func decodeManifest(raw []byte) (*syncManifest, bool, error) {
	if len(raw) < 4 || string(raw[:4]) != "GBMZ" {
		return nil, false, nil
	}
	js, e := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(raw[4:])), 256<<20))
	if e != nil {
		return nil, true, e
	}
	var m syncManifest
	if e = json.Unmarshal(js, &m); e != nil || m.Format != "GBM1" {
		return nil, true, fmt.Errorf("unrecognised sync manifest")
	}
	if len(m.Files) > maxManifestFiles || m.Total > maxUnpackedSync {
		return nil, true, fmt.Errorf("manifest exceeds the sync limits")
	}
	return &m, true, nil
}

// restoreChunked writes the manifest's files into stage, copying pieces that
// already exist in the current local profile and downloading the rest.
func (s *Store) restoreChunked(m *syncManifest, dir, root, stage string, t *transfer) (int64, error) {
	key, e := base64.StdEncoding.DecodeString(m.ChunkKey)
	if e != nil || len(key) != 32 {
		return 0, fmt.Errorf("manifest has no valid chunk key")
	}
	// What we already have locally.
	have := map[string]chunkLoc{}
	if _, err := os.Stat(dir); err == nil {
		_, locs, _, err := scanProfile(dir, key, loadSyncCache(root, m.Profile.ID))
		if err == nil {
			have = locs
		}
	}
	var need []string
	seen := map[string]bool{}
	size := map[string]int64{}
	var needPlain, writeTotal, savedPlain int64
	for _, f := range m.Files {
		writeTotal += f.Size
		for i, id := range f.Chunks {
			n := f.Size - int64(i)*chunkSize
			if n > chunkSize {
				n = chunkSize
			}
			if n > 0 {
				size[id] = n
			}
		}
	}
	for _, f := range m.Files {
		if !filepath.IsLocal(filepath.FromSlash(f.Path)) || strings.Contains(f.Path, "\\") || f.Path == "" {
			return 0, fmt.Errorf("unsafe path in manifest: %q", f.Path)
		}
		for _, id := range f.Chunks {
			if len(id) != 64 {
				return 0, fmt.Errorf("bad chunk ID in manifest")
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			if _, ok := have[id]; ok {
				savedPlain += size[id]
			} else {
				need = append(need, id)
				needPlain += size[id]
			}
		}
	}
	// Downloaded chunks wait in a scratch folder next to the stage.
	scratch := filepath.Join(stage, ".gb-chunks")
	if e = os.MkdirAll(scratch, 0o700); e != nil {
		return 0, e
	}
	defer os.RemoveAll(scratch)
	c := s.sync()
	t.saved(savedPlain)
	t.phase("download", needPlain)
	var got, plainDone, wire int64
	for i := 0; i < len(need); i += chunkGetBatch {
		j := i + chunkGetBatch
		if j > len(need) {
			j = len(need)
		}
		var cur int64 = chunkSize
		e = c.getChunks(key, need[i:j], func(id string, data []byte) error {
			got += int64(len(data))
			plainDone += size[id]
			wire = 0
			t.set(plainDone)
			return os.WriteFile(filepath.Join(scratch, id), data, 0o600)
		}, func(n int64) {
			wire += n
			if wire < cur {
				t.set(plainDone + wire)
			}
		})
		if e != nil {
			return got, e
		}
	}
	t.phase("write", writeTotal)
	for _, f := range m.Files {
		dst := filepath.Join(stage, filepath.FromSlash(f.Path))
		if e = os.MkdirAll(filepath.Dir(dst), 0o700); e != nil {
			return got, e
		}
		out, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if e != nil {
			return got, e
		}
		var written int64
		for _, id := range f.Chunks {
			var data []byte
			if l, ok := have[id]; ok {
				data, e = readLoc(l)
				if e == nil && chunkID(key, data) != id {
					e = fmt.Errorf("local file changed during restore")
				}
			} else {
				data, e = os.ReadFile(filepath.Join(scratch, id))
			}
			if e != nil {
				out.Close()
				return got, e
			}
			if _, e = out.Write(data); e != nil {
				out.Close()
				return got, e
			}
			written += int64(len(data))
			t.add(int64(len(data)))
		}
		if e = out.Close(); e != nil {
			return got, e
		}
		if written != f.Size {
			return got, fmt.Errorf("%s: size mismatch after restore", f.Path)
		}
		if f.MTime > 0 {
			t := time.Unix(0, f.MTime)
			_ = os.Chtimes(dst, t, t)
		}
	}
	return got, nil
}
