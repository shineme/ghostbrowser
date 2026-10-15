package main

// Registry layout for "GhostBrowser as a Windows browser" (HKCU only, no
// admin). Pure data so it can be tested on any OS; defaultbrowser_windows.go
// writes it.

const (
	dbAppName      = "GhostBrowser"
	dbClientKey    = `Software\Clients\StartMenuInternet\GhostBrowser`
	dbCapsKey      = dbClientKey + `\Capabilities`
	dbRegAppsKey   = `Software\RegisteredApplications`
	dbURLProgID    = "GhostBrowserURL"
	dbHTMLProgID   = "GhostBrowserHTML"
	dbClassesKey   = `Software\Classes`
	dbUserChoiceFm = `Software\Microsoft\Windows\Shell\Associations\UrlAssociations\%s\UserChoice`
	dbDescription  = "GhostBrowser 指纹浏览器：外部链接在默认环境中打开"
)

var dbFileExts = []string{".htm", ".html", ".shtml", ".xht", ".xhtml", ".svg"}

// regKey is one key and its string values ("" = the key's default value).
type regKey struct {
	Path   string
	Values map[string]string
}

// dbOpenCommand is the shell command for links and files.
func dbOpenCommand(exe string) string { return `"` + exe + `" --open-url "%1"` }

// defaultBrowserKeys is everything RegisterDefaultBrowser writes under HKCU.
func defaultBrowserKeys(exe string) []regKey {
	icon := exe + ",0"
	cmd := dbOpenCommand(exe)
	urlAssoc := map[string]string{"http": dbURLProgID, "https": dbURLProgID}
	fileAssoc := map[string]string{}
	for _, e := range dbFileExts {
		fileAssoc[e] = dbHTMLProgID
	}
	keys := []regKey{
		{dbClientKey, map[string]string{"": dbAppName}},
		{dbClientKey + `\DefaultIcon`, map[string]string{"": icon}},
		{dbClientKey + `\shell\open\command`, map[string]string{"": `"` + exe + `"`}},
		{dbCapsKey, map[string]string{"ApplicationName": dbAppName, "ApplicationDescription": dbDescription, "ApplicationIcon": icon}},
		{dbCapsKey + `\URLAssociations`, urlAssoc},
		{dbCapsKey + `\FileAssociations`, fileAssoc},
		{dbCapsKey + `\StartMenu`, map[string]string{"StartMenuInternet": dbAppName}},
		{dbRegAppsKey, map[string]string{dbAppName: dbCapsKey}},
	}
	for _, pg := range []struct{ id, name string }{{dbURLProgID, "GhostBrowser URL"}, {dbHTMLProgID, "GhostBrowser HTML Document"}} {
		base := dbClassesKey + `\` + pg.id
		vals := map[string]string{"": pg.name, "FriendlyTypeName": pg.name}
		if pg.id == dbURLProgID {
			vals["URL Protocol"] = ""
		}
		keys = append(keys,
			regKey{base, vals},
			regKey{base + `\DefaultIcon`, map[string]string{"": icon}},
			regKey{base + `\Application`, map[string]string{"ApplicationName": dbAppName, "ApplicationIcon": icon, "ApplicationDescription": dbDescription}},
			regKey{base + `\shell\open\command`, map[string]string{"": cmd}},
		)
	}
	for _, e := range dbFileExts { // "Open with" lists GhostBrowser for these files
		keys = append(keys, regKey{dbClassesKey + `\` + e + `\OpenWithProgids`, map[string]string{dbHTMLProgID: ""}})
	}
	return keys
}

// defaultBrowserTrees are the keys UnregisterDefaultBrowser deletes outright;
// the shared keys (RegisteredApplications, .html\OpenWithProgids) only lose
// GhostBrowser's value.
func defaultBrowserTrees() []string {
	return []string{dbClientKey, dbClassesKey + `\` + dbURLProgID, dbClassesKey + `\` + dbHTMLProgID}
}

// defaultBrowserSharedValues: key -> value name removed on unregister.
func defaultBrowserSharedValues() map[string]string {
	m := map[string]string{dbRegAppsKey: dbAppName}
	for _, e := range dbFileExts {
		m[dbClassesKey+`\`+e+`\OpenWithProgids`] = dbHTMLProgID
	}
	return m
}

// needsReRegister: registered, but the command points at another EXE path
// (the portable EXE was moved or renamed).
func needsReRegister(registered bool, currentCmd, exe string) bool {
	return registered && currentCmd != dbOpenCommand(exe)
}

// isSystemDefaultChoice: both http and https UserChoice point at our ProgId.
func isSystemDefaultChoice(httpProgID, httpsProgID string) bool {
	return httpsProgID == dbURLProgID && httpProgID == dbURLProgID
}
