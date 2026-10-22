package main

// 修复云端: chunk_missing errors, front-end (nginx / CDN) error pages, and the
// forced re-upload that puts back the chunks a cloud manifest points at.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// errChunkMissing: the cloud copy references chunks the server no longer has
// (HTTP 409 code "chunk_missing"; an old ghostsync.php answers 404 on
// /api/chunks/get, which nginx may have replaced with its own HTML page).
type errChunkMissing struct {
	ids   []string
	count int
	path  string
}

func (e errChunkMissing) Error() string {
	n := e.count
	if n == 0 {
		n = len(e.ids)
	}
	what := "部分数据分块"
	if n > 0 {
		what = fmt.Sprintf("%d 个数据分块", n)
	}
	return "云端缺少该环境的" + what + "，请在上传它的设备上点「修复云端」/重新上传（更多 → 修复云端），然后再拉取"
}

func isChunkMissing(e error) bool { var x errChunkMissing; return errors.As(e, &x) }

// chunkMissingIDs returns the chunk IDs the server named (may be empty).
func chunkMissingIDs(e error) []string {
	var x errChunkMissing
	if errors.As(e, &x) {
		return x.ids
	}
	return nil
}

// reqLabel names the request for an error: "POST /api/chunks/get" (the r=
// parameter of the single-file server, else the URL path).
func reqLabel(resp *http.Response) string {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return "?"
	}
	u := resp.Request.URL
	p := u.Query().Get("r")
	if p == "" {
		p = u.Path
	}
	return resp.Request.Method + " " + p
}

func reqAPIPath(resp *http.Response) string {
	l := reqLabel(resp)
	if _, p, ok := strings.Cut(l, " "); ok {
		return p
	}
	return l
}

func isHTMLBody(resp *http.Response, b []byte) bool {
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/html") {
		return true
	}
	t := strings.ToLower(strings.TrimSpace(string(b)))
	return strings.HasPrefix(t, "<!doctype html") || strings.HasPrefix(t, "<html") || strings.HasPrefix(t, "<head") || strings.HasPrefix(t, "<center>") || strings.HasPrefix(t, "<body")
}

var htmlTitleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// frontPageErr explains an HTML error body: the web server or CDN in front of
// PHP replaced ghostsync.php's JSON error with its own page.
func frontPageErr(resp *http.Response, b []byte) error {
	title := ""
	if m := htmlTitleRE.FindSubmatch(b); m != nil {
		title = strings.Join(strings.Fields(string(m[1])), " ")
	}
	if title == "" {
		title = http.StatusText(resp.StatusCode)
	}
	if len(title) > 120 {
		title = title[:120]
	}
	hint := "服务器前端（nginx/CDN）替换了错误页，真实错误被隐藏；宝塔请在网站配置里关闭 fastcgi_intercept_errors（改为 off）或移除 error_page 404 等行后重载 nginx，Cloudflare 请检查自定义错误页"
	if resp.StatusCode == 404 && reqAPIPath(resp) != "/api/chunks/get" {
		hint += "；如果所有请求都 404，请确认服务器地址指向 ghostsync.php"
	}
	return fmt.Errorf("sync HTTP %d (%s): 返回了网页「%s」而不是 JSON。%s", resp.StatusCode, reqLabel(resp), title, hint)
}

// chunkMissingErr recognises a missing-chunk answer.
func chunkMissingErr(resp *http.Response, b []byte) error {
	if resp.StatusCode == 409 {
		var v struct {
			Code    string   `json:"code"`
			Missing []string `json:"missing"`
			Count   int      `json:"count"`
		}
		if json.Unmarshal(b, &v) == nil && v.Code == "chunk_missing" {
			return errChunkMissing{ids: v.Missing, count: v.Count, path: reqAPIPath(resp)}
		}
		return nil
	}
	// ghostsync.php ≤ 0.8.7 answers a missing chunk with 404, which a BaoTa
	// nginx (fastcgi_intercept_errors + error_page 404) turns into HTML.
	if resp.StatusCode == 404 && reqAPIPath(resp) == "/api/chunks/get" {
		var v struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &v) == nil && strings.Contains(v.Error, "is missing") {
			var ids []string
			if f := strings.Fields(v.Error); len(f) > 1 && len(f[1]) == 64 {
				ids = []string{f[1]}
			}
			return errChunkMissing{ids: ids, path: "/api/chunks/get"}
		}
		if isHTMLBody(resp, b) {
			return errChunkMissing{path: "/api/chunks/get"}
		}
	}
	return nil
}

// ---------------------------------------------------------------- repair

// CloudRepairResult is shown after 修复云端.
type CloudRepairResult struct {
	Version  int64  `json:"version"`
	Uploaded int    `json:"uploaded"` // chunks the server lacked and got again
	Bytes    int64  `json:"bytes"`
	Message  string `json:"message"`
}

// RepairCloud force-uploads a local profile: a fresh missing-chunk check over
// every chunk of the local copy, upload of all the server lacks, and a new
// committed version even when nothing changed locally. Run it on the device
// that holds the good copy.
func (s *Store) RepairCloud(id string) (res CloudRepairResult, err error) {
	t0 := time.Now()
	var pr pushResult
	defer func() { s.logSync(id, "repair", pr.Bytes, t0, err) }()
	p, ok := s.GetProfile(id)
	if !ok {
		return res, fmt.Errorf("profile not found")
	}
	if p.ShareID != "" {
		return res, fmt.Errorf("共享给你的环境只能由分享者修复")
	}
	if _, dir, e := s.syncDir(id); e != nil {
		return res, e
	} else if !dirHasFiles(dir) {
		return res, fmt.Errorf("本机没有这个环境的数据，无法修复云端：请在上传它的设备上点「修复云端」")
	}
	t := s.newTransfer(id, "up")
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("sync crashed: %v", r)
			}
			t.finish(err)
		}()
		var sent int64
		err = s.pushProfileInner(id, &sent, t, &pr)
	}()
	if err != nil {
		if strings.Contains(err.Error(), "pull first") {
			cloud := ""
			if l, e := s.ListCloud(); e == nil {
				for _, m := range l {
					if m.ID == id {
						cloud = fmt.Sprintf("云端 v%d，", m.Version)
					}
				}
			}
			err = fmt.Errorf("本机的副本不是云端最新版本（%s本机 v%d），不能用它修复：请在上传最新版本的设备上点「修复云端」；如果那台设备已不可用，可在「历史版本」里恢复一个完好的旧版本", cloud, p.CloudVer)
		}
		return res, err
	}
	res = CloudRepairResult{Version: pr.Version, Uploaded: pr.Uploaded, Bytes: pr.Bytes}
	switch {
	case !pr.Chunked:
		res.Message = fmt.Sprintf("已整包重新上传，云端版本 v%d", pr.Version)
	case pr.Uploaded > 0:
		res.Message = fmt.Sprintf("已修复：补传 %d 个缺失的数据分块，云端版本 v%d", pr.Uploaded, pr.Version)
	default:
		res.Message = fmt.Sprintf("云端数据完整，已重新提交为 v%d", pr.Version)
	}
	return res, nil
}

// RepairGlobal re-uploads the global record (settings, proxies, fingerprint
// library) even when nothing changed.
func (s *Store) RepairGlobal() (CloudRepairResult, error) {
	if e := s.PushGlobal(); e != nil {
		return CloudRepairResult{}, e
	}
	v := s.globalState().Version
	return CloudRepairResult{Version: v, Message: fmt.Sprintf("全局数据已重新上传，云端版本 v%d", v)}, nil
}

// CloudCheck is the chunk health of a profile's current cloud version.
type CloudCheck struct {
	Version int64  `json:"version"`
	Chunked bool   `json:"chunked"`
	Chunks  int    `json:"chunks"`
	Missing int    `json:"missing"`
	Message string `json:"message"`
}

// CheckCloud downloads and decrypts the current manifest and asks the server
// which of its chunks it lacks (works with every ghostsync.php version).
func (s *Store) CheckCloud(id string) (CloudCheck, error) {
	var out CloudCheck
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return out, e
	}
	c := s.sync()
	resp, e := c.req("GET", "/api/profiles/"+url.PathEscape(id)+"/blob", nil, "")
	if e != nil {
		return out, e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return out, e
	}
	fmt.Sscan(resp.Header.Get("X-Profile-Version"), &out.Version)
	enc, e := readAllLimit(resp.Body, maxSyncBlob)
	if e != nil {
		return out, e
	}
	secret, e := s.dataSecret()
	if e != nil {
		return out, e
	}
	raw, e := decrypt(enc, secret)
	if e != nil {
		return out, fmt.Errorf("decrypt failed; check the sync password: %w", e)
	}
	m, chunked, e := decodeManifest(raw)
	if e != nil {
		return out, e
	}
	if !chunked {
		out.Message = fmt.Sprintf("v%d 是整包格式，不依赖分块，数据完整", out.Version)
		return out, nil
	}
	out.Chunked = true
	seen := map[string]bool{}
	var ids []string
	for _, f := range m.Files {
		for _, x := range f.Chunks {
			if !seen[x] {
				seen[x] = true
				ids = append(ids, x)
			}
		}
	}
	out.Chunks = len(ids)
	miss, e := c.missingChunks(ids)
	if e != nil {
		return out, e
	}
	out.Missing = len(miss)
	if out.Missing == 0 {
		out.Message = fmt.Sprintf("云端 v%d 完整（%d 个分块都在）", out.Version, out.Chunks)
	} else {
		out.Message = fmt.Sprintf("云端 v%d 缺少 %d / %d 个分块：请在上传它的设备上点「修复云端」", out.Version, out.Missing, out.Chunks)
	}
	return out, nil
}

func readAllLimit(r io.Reader, max int64) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, max+1))
	if e == nil && int64(len(b)) > max {
		e = fmt.Errorf("cloud blob exceeds %d MiB", max>>20)
	}
	return b, e
}

func dirHasFiles(dir string) bool {
	ents, e := os.ReadDir(dir)
	return e == nil && len(ents) > 0
}
