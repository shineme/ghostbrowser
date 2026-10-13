package main

import (
	"fmt"
	"strings"
	"time"
)

// countryLangs: Accept-Language per exit country (same table as the editor's
// country presets in frontend/src/locales.tsx).
var countryLangs = map[string][]string{
	"US": {"en-US", "en"},
	"CA": {"en-CA", "en", "fr-CA"},
	"MX": {"es-MX", "es", "en"},
	"BR": {"pt-BR", "pt", "en"},
	"AR": {"es-AR", "es", "en"},
	"CL": {"es-CL", "es", "en"},
	"CO": {"es-CO", "es", "en"},
	"PE": {"es-PE", "es", "en"},
	"GB": {"en-GB", "en"},
	"IE": {"en-IE", "en"},
	"DE": {"de-DE", "de", "en"},
	"FR": {"fr-FR", "fr", "en"},
	"ES": {"es-ES", "es", "en"},
	"PT": {"pt-PT", "pt", "en"},
	"IT": {"it-IT", "it", "en"},
	"NL": {"nl-NL", "nl", "en"},
	"BE": {"nl-BE", "fr-BE", "en"},
	"CH": {"de-CH", "fr-CH", "en"},
	"AT": {"de-AT", "de", "en"},
	"SE": {"sv-SE", "sv", "en"},
	"NO": {"nb-NO", "no", "en"},
	"DK": {"da-DK", "da", "en"},
	"FI": {"fi-FI", "fi", "en"},
	"PL": {"pl-PL", "pl", "en"},
	"CZ": {"cs-CZ", "cs", "en"},
	"HU": {"hu-HU", "hu", "en"},
	"RO": {"ro-RO", "ro", "en"},
	"GR": {"el-GR", "el", "en"},
	"UA": {"uk-UA", "uk", "ru", "en"},
	"RU": {"ru-RU", "ru", "en"},
	"TR": {"tr-TR", "tr", "en"},
	"IL": {"he-IL", "he", "en"},
	"AE": {"ar-AE", "ar", "en"},
	"SA": {"ar-SA", "ar", "en"},
	"EG": {"ar-EG", "ar", "en"},
	"ZA": {"en-ZA", "en", "af"},
	"NG": {"en-NG", "en"},
	"KE": {"en-KE", "sw", "en"},
	"IN": {"en-IN", "hi", "en"},
	"PK": {"en-PK", "ur", "en"},
	"BD": {"bn-BD", "bn", "en"},
	"TH": {"th-TH", "th", "en"},
	"VN": {"vi-VN", "vi", "en"},
	"MY": {"ms-MY", "en-MY", "en"},
	"SG": {"en-SG", "en", "zh-SG"},
	"ID": {"id-ID", "id", "en"},
	"PH": {"en-PH", "fil", "en"},
	"CN": {"zh-CN", "zh", "en"},
	"HK": {"zh-HK", "zh", "en"},
	"TW": {"zh-TW", "zh", "en"},
	"JP": {"ja-JP", "ja", "en"},
	"KR": {"ko-KR", "ko", "en"},
	"AU": {"en-AU", "en"},
	"NZ": {"en-NZ", "en"},
}

// languagesFor picks Accept-Language for an exit country, refined by region
// where a country has more than one main language.
func languagesFor(cc, region string) []string {
	cc, r := strings.ToUpper(cc), strings.ToLower(region)
	switch cc {
	case "CA":
		if strings.Contains(r, "quebec") || strings.Contains(r, "québec") {
			return []string{"fr-CA", "fr", "en-CA", "en"}
		}
	case "CH":
		for _, k := range []string{"geneva", "genève", "vaud", "neuchâtel", "neuchatel", "jura", "fribourg", "valais"} {
			if strings.Contains(r, k) {
				return []string{"fr-CH", "fr", "en"}
			}
		}
		if strings.Contains(r, "ticino") {
			return []string{"it-CH", "it", "en"}
		}
		return []string{"de-CH", "de", "en"}
	case "BE":
		if strings.Contains(r, "wallon") || strings.Contains(r, "bruxelles") || strings.Contains(r, "brussels") {
			return []string{"fr-BE", "fr", "nl", "en"}
		}
		return []string{"nl-BE", "nl", "fr", "en"}
	}
	if v, ok := countryLangs[cc]; ok {
		return append([]string(nil), v...)
	}
	return languageForCountry(cc)
}

// GeoMatch is what a profile's region settings become for its current exit IP.
type GeoMatch struct {
	OK        bool     `json:"ok"`
	Error     string   `json:"error"`
	Stale     bool     `json:"stale"` // live check failed; values come from the last good check
	Direct    bool     `json:"direct"`
	IP        string   `json:"ip"`
	Country   string   `json:"country"`
	Region    string   `json:"region"`
	City      string   `json:"city"`
	ISP       string   `json:"isp"`
	IPType    string   `json:"ipType"`
	Timezone  string   `json:"timezone"`
	Languages []string `json:"languages"`
	Lat       float64  `json:"lat"`
	Lon       float64  `json:"lon"`
	Latency   int64    `json:"latency"`
}

func needsGeo(fp Fingerprint) bool { return fp.TimezoneFromIP || fp.LanguageFromIP || fp.GeoFromIP }

// liveExit checks the exit IP now (or reuses a check younger than maxAge) and
// stores the result on a saved proxy. A nil proxy means the direct connection.
func (s *Store) liveExit(px *Proxy, maxAge time.Duration) GeoMatch {
	target := Proxy{Type: "none"}
	if px != nil && px.Host != "" && px.Type != "none" {
		target = *px
	}
	m := GeoMatch{Direct: target.Type == "none"}
	cur := target
	if !(cur.Status == "ok" && time.Since(time.Unix(cur.Checked, 0)) < maxAge) {
		fresh := CheckProxy(target)
		if fresh.Status == "ok" {
			cur = fresh
			if target.ID != "" {
				if saved, ok := s.GetProxy(target.ID); ok {
					keep := *saved
					fresh.Name, fresh.Tag, fresh.ChangeIP, fresh.Speed, fresh.SpeedAt = keep.Name, keep.Tag, keep.ChangeIP, keep.Speed, keep.SpeedAt
					s.PutProxy(fresh)
				}
			}
		} else if cur.Status == "ok" && cur.Timezone != "" {
			m.Stale, m.Error = true, fresh.Error
		} else {
			m.Error = fresh.Error
			if m.Error == "" {
				m.Error = "无法连接"
			}
			return m
		}
	}
	m.OK = true
	m.IP, m.Country, m.Region, m.City, m.ISP, m.IPType = cur.IP, cur.Country, cur.Region, cur.City, cur.ISP, cur.IPType
	m.Timezone, m.Lat, m.Lon, m.Latency = cur.Timezone, cur.Lat, cur.Lon, cur.Latency
	m.Languages = languagesFor(cur.Country, cur.Region)
	return m
}

func applyGeo(fp Fingerprint, m GeoMatch) Fingerprint {
	if !m.OK {
		return fp
	}
	if fp.TimezoneFromIP && m.Timezone != "" {
		fp.Timezone = m.Timezone
	}
	if fp.LanguageFromIP && m.Country != "" {
		fp.Languages = m.Languages
	}
	if fp.GeoFromIP && (m.Lat != 0 || m.Lon != 0) {
		fp.Latitude, fp.Longitude = m.Lat, m.Lon
	}
	return fp
}

// matchFP resolves the "follow IP" toggles against the exit IP as it is right
// now, so rotating proxies get the timezone/language of today's exit. If the
// exit cannot be determined at all the launch is refused rather than leaking a
// timezone that does not match the IP.
func (s *Store) matchFP(fp Fingerprint, px *Proxy) (Fingerprint, error) {
	if !needsGeo(fp) {
		return fp, nil
	}
	m := s.liveExit(px, 60*time.Second)
	if !m.OK {
		what := "代理"
		if m.Direct {
			what = "本机网络"
		}
		return fp, fmt.Errorf("%s出口检测失败（%s），时区/语言/定位无法跟随 IP。请检查代理，或在环境的「高级」里关闭「跟随 IP」", what, m.Error)
	}
	return applyGeo(fp, m), nil
}
