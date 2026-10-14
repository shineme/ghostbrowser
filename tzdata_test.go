package main

import (
	"testing"
	"time"
)

// With ZONEINFO pointing nowhere (as on a Windows box without Go installed),
// the embedded tzdata must still resolve IANA names.
func TestLoadLocationWithoutSystemZoneinfo(t *testing.T) {
	t.Setenv("ZONEINFO", "/nonexistent/zoneinfo.zip")
	for _, name := range []string{"America/New_York", "Asia/Shanghai", "Europe/London"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatalf("LoadLocation(%q): %v", name, err)
		}
		if loc.String() != name {
			t.Fatalf("got %q", loc.String())
		}
	}
}
