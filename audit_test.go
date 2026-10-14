package main

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	d := t.TempDir()
	return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: d + "/profiles", APIKey: "gb_testkey"}}
}

func TestAuditAppendCapFilter(t *testing.T) {
	s := testStore(t)
	s.auditNamed("ui", "profile.start", "p1", "Alpha", nil, "")
	s.auditNamed("api", "profile.stop", "p2", "Beta", errors.New("boom"), "")
	s.auditNamed("sync", "sync.push", "p1", "Alpha", nil, "12 B")
	r := s.AuditQuery(AuditFilter{})
	if r.All != 3 || r.Total != 3 || r.Failed != 1 {
		t.Fatalf("counts: %+v", r)
	}
	if r.Items[0].Action != "sync.push" { // newest first
		t.Fatalf("order: %+v", r.Items[0])
	}
	if _, err := os.Stat(s.auditFile()); err != nil {
		t.Fatalf("not persisted: %v", err)
	}
	if n := s.AuditQuery(AuditFilter{Category: "profile"}).Total; n != 2 {
		t.Fatalf("category filter = %d", n)
	}
	if n := s.AuditQuery(AuditFilter{OnlyFail: true}).Total; n != 1 {
		t.Fatalf("onlyFail = %d", n)
	}
	if n := s.AuditQuery(AuditFilter{Target: "p1", Source: "sync"}).Total; n != 1 {
		t.Fatalf("target+source = %d", n)
	}
	if n := s.AuditQuery(AuditFilter{Query: "beta"}).Total; n != 1 {
		t.Fatalf("query = %d", n)
	}
	if r := s.AuditQuery(AuditFilter{Limit: 1}); len(r.Items) != 1 || r.Total != 3 {
		t.Fatalf("limit: %d items total %d", len(r.Items), r.Total)
	}
	for i := 0; i < auditMax+50; i++ {
		s.auditNamed("ui", "profile.edit", fmt.Sprint("x", i), "", nil, "")
	}
	if r := s.AuditQuery(AuditFilter{}); r.All != auditMax || r.Items[0].Target != fmt.Sprint("x", auditMax+49) {
		t.Fatalf("cap: all=%d first=%s", r.All, r.Items[0].Target)
	}
	// reload from disk
	audit.mu.Lock()
	audit.list, audit.dir = nil, ""
	audit.mu.Unlock()
	if r := s.AuditQuery(AuditFilter{}); r.All != auditMax {
		t.Fatalf("reload all=%d", r.All)
	}
	if b := auditExport(s.AuditQuery(AuditFilter{Limit: 2}).Items, "csv"); len(b) < 10 {
		t.Fatal("empty csv")
	}
	s.ClearAudit()
	if r := s.AuditQuery(AuditFilter{}); r.All != 0 {
		t.Fatalf("clear: %d", r.All)
	}
}
