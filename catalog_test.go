package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const sampleCatalog = `{"revision":3,"groups":[
 {"id":"ai","name":"AI","sites":[{"id":"c","name":"ChatGPT","url":"https://chatgpt.com/","icon":"https://chatgpt.com/favicon.ico"},
  {"id":"evil","name":"x","url":"javascript:alert(1)"},{"id":"f","name":"file","url":"file:///C:/x"}]},
 {"id":"empty","name":"空","sites":[{"url":"ftp://x"}]},
 {"name":"电商","sites":[{"url":"https://www.amazon.com/","icon":"javascript:x"}]}],
 "announcements":[
  {"id":"old","title":"旧","body":"b","level":"info","start":"2020-01-01","end":"2020-02-01"},
  {"id":"now","title":"现在","body":"**粗体**","level":"warn","start":1700000000,"end":""},
  {"id":"future","title":"将来","body":"b","start":"2999-01-01T00:00:00+08:00"},
  {"id":"lvl","title":"级别","body":"b","level":"<script>"}]}`

func TestParseCatalog(t *testing.T) {
	c, err := parseCatalog([]byte(sampleCatalog))
	if err != nil {
		t.Fatal(err)
	}
	if c.Revision != 3 || len(c.Groups) != 2 {
		t.Fatalf("groups: %+v", c.Groups)
	}
	if len(c.Groups[0].Sites) != 1 || c.Groups[0].Sites[0].ID != "c" {
		t.Fatalf("unsafe URLs must be dropped: %+v", c.Groups[0].Sites)
	}
	g := c.Groups[1]
	if g.ID == "" || g.Sites[0].Name != "www.amazon.com" || g.Sites[0].Icon != "" || g.Sites[0].ID == "" {
		t.Fatalf("defaults: %+v", g)
	}
	act := activeAnnouncements(c.Announcements, time.Now().Unix())
	ids := []string{}
	for _, a := range act {
		ids = append(ids, a.ID)
		if a.ID == "lvl" && a.Level != "info" {
			t.Fatal("level not normalised")
		}
	}
	if strings.Join(ids, ",") != "now,lvl" && strings.Join(ids, ",") != "lvl,now" {
		t.Fatalf("active: %v", ids)
	}
	if _, err := parseCatalog([]byte("not json")); err == nil {
		t.Fatal("bad json accepted")
	}
}

func TestCatalogCacheOffline(t *testing.T) {
	var up atomic.Bool
	up.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() {
			w.WriteHeader(502)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok-tok-tok" || r.URL.Query().Get("r") != "/api/catalog" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(sampleCatalog))
	}))
	defer srv.Close()
	dir := t.TempDir()
	st := Settings{SyncServer: srv.URL + "/ghostsync.php", SyncToken: "tok-tok-tok"}
	svc := &catalogSvc{}
	s1 := svc.Refresh(dir, st)
	if s1.Offline || s1.Error != "" || len(s1.Catalog.Groups) != 2 || s1.FetchedAt == 0 || s1.Source != "server" {
		t.Fatalf("refresh: %+v", s1)
	}
	svc.MarkRead(dir, []string{"now"})
	// new process, server down: cache + read marks survive
	up.Store(false)
	svc2 := &catalogSvc{}
	if g := svc2.Get(dir); len(g.Catalog.Groups) != 2 || len(g.Read) != 1 {
		t.Fatalf("cache not loaded: %+v", g)
	}
	s2 := svc2.Refresh(dir, st)
	if !s2.Offline || s2.Error == "" || len(s2.Catalog.Groups) != 2 || s2.FetchedAt != s1.FetchedAt {
		t.Fatalf("offline fallback: %+v", s2)
	}
	b, _ := json.Marshal(s2)
	if !strings.Contains(string(b), `"read":["now"]`) {
		t.Fatalf("read marks: %s", b)
	}
	// no server configured at all
	s3 := (&catalogSvc{}).Refresh(t.TempDir(), Settings{})
	if !s3.Offline || len(s3.Catalog.Groups) != 0 || s3.Catalog.Groups == nil {
		t.Fatalf("no source: %+v", s3)
	}
}

func TestDocSchemaRejectsNewer(t *testing.T) {
	if err := checkDocSchema([]byte(`{"schema":1,"x":1}`), "环境 JSON"); err != nil {
		t.Fatal(err)
	}
	if err := checkDocSchema([]byte(`[{"name":"a"}]`), "Cookie"); err != nil {
		t.Fatal(err)
	}
	err := checkDocSchema([]byte("\ufeff{\"schema\": 2, \"newField\": true}"), "环境 JSON")
	if err == nil || !strings.Contains(err.Error(), "请先升级 GhostBrowser") || !strings.Contains(err.Error(), "schema 2") {
		t.Fatalf("newer schema: %v", err)
	}
	if _, _, err := parseCookies("auto", `{"schema":5,"cookies":[{"name":"a","value":"b","domain":".x.test"}]}`); err == nil || !strings.Contains(err.Error(), "升级") {
		t.Fatalf("cookie doc: %v", err)
	}
	if list, _, err := parseCookies("auto", `{"schema":1,"cookies":[{"name":"a","value":"b","domain":".x.test"}]}`); err != nil || len(list) != 1 {
		t.Fatalf("cookie schema 1: %v %v", list, err)
	}
	a := &App{store: rfStore(t)}
	if _, err := a.ImportNativeJSON(`{"schema":2,"schemaVersion":1,"name":"x"}`); err == nil || !strings.Contains(err.Error(), "升级") {
		t.Fatalf("native import: %v", err)
	}
	out := string(withDocSchema([]byte("{\n  \"a\": 1\n}")))
	var m map[string]any
	if json.Unmarshal([]byte(out), &m) != nil || m["schema"] != float64(1) || m["a"] != float64(1) {
		t.Fatalf("withDocSchema: %s", out)
	}
}
