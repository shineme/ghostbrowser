package undetectable

// Embed the IANA time zone database so time.LoadLocation works on Windows,
// which ships no zoneinfo (fixes "tzId: unknown time zone America/New_York").
import _ "time/tzdata"
