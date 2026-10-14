package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectGeoFallsBack(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", 500)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"ip":"203.0.113.7","country_code":"jp","region":"Tokyo","city":"Tokyo","latitude":35.6,"longitude":139.7,"timezone":{"id":"Asia/Tokyo"},"connection":{"isp":"TestNet"}}`))
	}))
	defer good.Close()
	srcs := []geoSource{{"bad", bad.URL, parseIPAPI}, {"good", good.URL, parseIPWho}}
	g, name, err := detectGeo(http.DefaultClient, srcs)
	if err != nil {
		t.Fatal(err)
	}
	if name != "good" || g.IP != "203.0.113.7" || g.CC != "JP" || g.TZ != "Asia/Tokyo" || g.ISP != "TestNet" {
		t.Fatalf("got %s %+v", name, g)
	}
	_, _, err = detectGeo(http.DefaultClient, srcs[:1])
	if err == nil {
		t.Fatal("expected error when all sources fail")
	}
}
