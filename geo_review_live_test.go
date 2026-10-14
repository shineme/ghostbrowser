package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGeoSourcesRealLive(t *testing.T) {
	if os.Getenv("GHOST_NET") == "" {
		t.Skip("set GHOST_NET=1 to query the real IP services")
	}
	for _, src := range geoSources {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		g, err := fetchGeo(ctx, http.DefaultClient, src)
		cancel()
		t.Logf("%-10s err=%v ip?=%v cc=%s tz=%s isp?=%v", src.Name, err, g.IP != "", g.CC, g.TZ, g.ISP != "")
	}
}
