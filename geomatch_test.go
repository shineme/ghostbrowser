package main

import "testing"

func TestLanguagesFor(t *testing.T) {
	if l := languagesFor("CA", "Quebec"); l[0] != "fr-CA" {
		t.Fatal(l)
	}
	if l := languagesFor("DE", ""); l[0] != "de-DE" {
		t.Fatal(l)
	}
	if l := languagesFor("CH", "Geneva"); l[0] != "fr-CH" {
		t.Fatal(l)
	}
	fp := applyGeo(Fingerprint{TimezoneFromIP: true, LanguageFromIP: false, GeoFromIP: true, Timezone: "X", Languages: []string{"keep"}},
		GeoMatch{OK: true, Timezone: "Europe/Berlin", Languages: []string{"de-DE"}, Lat: 52.5, Lon: 13.4})
	if fp.Timezone != "Europe/Berlin" || fp.Languages[0] != "keep" || fp.Latitude != 52.5 {
		t.Fatal(fp)
	}
}
