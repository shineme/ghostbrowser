package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pbBytes(field int, v []byte) []byte {
	b := binary.AppendUvarint(nil, uint64(field)<<3|2)
	b = binary.AppendUvarint(b, uint64(len(v)))
	return append(b, v...)
}

func testZip(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// testCRX3 wraps a zip in a CRX3 header carrying key and crx_id = sha256(key)[:16].
func testCRX3(key, zipBytes []byte) []byte {
	sum := sha256.Sum256(key)
	hdr := pbBytes(2, pbBytes(1, key))                                // sha256_with_rsa { public_key }
	hdr = append(hdr, pbBytes(2, pbBytes(1, []byte("other-key")))...) // a second, unrelated proof
	hdr = append(hdr, pbBytes(10000, pbBytes(1, sum[:16]))...)        // signed_header_data { crx_id }
	out := []byte("Cr24")
	out = binary.LittleEndian.AppendUint32(out, 3)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(hdr)))
	out = append(out, hdr...)
	return append(out, zipBytes...)
}

func TestExtCRX3InstallParse(t *testing.T) {
	old := extReg.root
	extReg.root = t.TempDir()
	defer func() { extReg.root = old }()

	key := []byte("fake-public-key-der-bytes-0123456789")
	z := testZip(t, map[string]string{
		"manifest.json":                    `{"manifest_version":3,"name":"__MSG_appName__","version":"1.2.3","default_locale":"en","icons":{"16":"i16.png","128":"i128.png"}}`,
		"_locales/en/messages.json":        `{"appName":{"message":"Ghost Helper"}}`,
		"_locales/zh_CN/messages.json":     `{"appName":{"message":"幽灵助手"}}`,
		"i128.png":                         "PNGDATA",
		"js/bg.js":                         "console.log(1)",
		"_metadata/verified_contents.json": "{}",
	})
	crx := testCRX3(key, z)
	info, err := parseCRX(crx)
	if err != nil {
		t.Fatal(err)
	}
	wantID := extIDFromKey(key)
	if info.Version != 3 || info.CrxID != wantID || !bytes.Equal(info.PubKey, key) {
		t.Fatalf("parse: ver=%d id=%s want %s key=%q", info.Version, info.CrxID, wantID, info.PubKey)
	}
	if !extIDRe.MatchString(wantID) {
		t.Fatalf("bad id %s", wantID)
	}
	e, err := extReg.installPackage(crx, "", "file", "x.crx")
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != wantID || e.Name != "幽灵助手" || e.Version != "1.2.3" || e.MV != 3 {
		t.Fatalf("ext = %+v", e)
	}
	if !strings.HasPrefix(e.Icon, "data:image/png;base64,") {
		t.Fatalf("icon %q", e.Icon)
	}
	if _, err := os.Stat(filepath.Join(e.Dir, "_metadata")); !os.IsNotExist(err) {
		t.Fatal("_metadata not stripped")
	}
	if !fileExists(filepath.Join(e.Dir, "js", "bg.js")) {
		t.Fatal("js/bg.js missing")
	}
	var man map[string]any
	b, _ := os.ReadFile(filepath.Join(e.Dir, "manifest.json"))
	json.Unmarshal(b, &man)
	if man["key"] != base64.StdEncoding.EncodeToString(key) {
		t.Fatalf("manifest key = %v", man["key"])
	}
	if l := extReg.list(); len(l) != 1 || l[0].ID != wantID {
		t.Fatalf("registry %+v", l)
	}
	if d := extensionDirs(&Profile{ExtensionIDs: []string{wantID, "missingmissingmissingmissingmiss"}}); len(d) != 1 || d[0] != e.Dir {
		t.Fatalf("dirs %v", d)
	}
	// Store ID mismatch is rejected.
	if _, err := extReg.installPackage(crx, strings.Repeat("a", 32), "store", "x"); err == nil {
		t.Fatal("expected ID mismatch error")
	}
	if err := extReg.remove(wantID); err != nil || fileExists(e.Dir) {
		t.Fatalf("remove: %v", err)
	}
}

func TestExtCRX2AndZip(t *testing.T) {
	old := extReg.root
	extReg.root = t.TempDir()
	defer func() { extReg.root = old }()
	key := []byte("crx2-key")
	z := testZip(t, map[string]string{"manifest.json": `{"manifest_version":2,"name":"Two","version":"2.0"}`})
	crx := []byte("Cr24")
	crx = binary.LittleEndian.AppendUint32(crx, 2)
	crx = binary.LittleEndian.AppendUint32(crx, uint32(len(key)))
	crx = binary.LittleEndian.AppendUint32(crx, 3)
	crx = append(append(append(crx, key...), "sig"...), z...)
	e, err := extReg.installPackage(crx, "", "url", "http://x/two.crx")
	if err != nil || e.ID != extIDFromKey(key) || e.Name != "Two" {
		t.Fatalf("crx2: %+v %v", e, err)
	}
	e2, err := extReg.installPackage(z, "", "file", "/tmp/two.zip")
	if err != nil || !extIDRe.MatchString(e2.ID) || e2.ID == e.ID {
		t.Fatalf("zip: %+v %v", e2, err)
	}
	if _, err := parseCRX([]byte("nope nope nope nope")); err == nil {
		t.Fatal("expected Cr24 error")
	}
	if _, err := extReg.installPackage(testZip(t, map[string]string{"a.txt": "x"}), "", "file", "y"); err == nil {
		t.Fatal("expected missing manifest error")
	}
}

func TestExtZipSlipRejected(t *testing.T) {
	root := t.TempDir()
	old := extReg.root
	extReg.root = filepath.Join(root, "ext")
	os.MkdirAll(extReg.root, 0o755)
	defer func() { extReg.root = old }()
	z := testZip(t, map[string]string{"manifest.json": `{"name":"Evil","version":"1"}`, "../../evil.txt": "pwned"})
	if _, err := extReg.installPackage(testCRX3([]byte("k"), z), "", "file", "evil.crx"); err == nil {
		t.Fatal("zip slip accepted")
	}
	if fileExists(filepath.Join(root, "evil.txt")) || fileExists(filepath.Join(filepath.Dir(root), "evil.txt")) {
		t.Fatal("file written outside the extension folder")
	}
	if len(extReg.list()) != 0 {
		t.Fatal("evil extension registered")
	}
}

func TestExtNormalizeIDAndArgs(t *testing.T) {
	id := strings.Repeat("b", 32)
	for _, in := range []string{id, "https://chromewebstore.google.com/detail/some-name/" + id, "https://chrome.google.com/webstore/detail/x/" + id + "?hl=zh"} {
		if got, err := normalizeExtID(in); err != nil || got != id {
			t.Fatalf("%s -> %s %v", in, got, err)
		}
	}
	if _, err := normalizeExtID("xyz"); err == nil {
		t.Fatal("expected error")
	}
	args := withExtensionArgs([]string{"--a", "--disable-features=Translate,MediaRouter", "--b"}, []string{"/e/1", "/e/2"})
	j := strings.Join(args, " ")
	if strings.Count(j, "--disable-features=") != 1 || !strings.Contains(j, "--disable-features=Translate,MediaRouter,DisableLoadExtensionCommandLineSwitch") || !strings.HasSuffix(j, "--load-extension=/e/1,/e/2") {
		t.Fatalf("args %v", args)
	}
	if a := withExtensionArgs([]string{"--x"}, []string{"/e"}); strings.Join(a, " ") != "--x --disable-features=DisableLoadExtensionCommandLineSwitch --load-extension=/e" {
		t.Fatalf("args %v", a)
	}
	if a := withExtensionArgs([]string{"--x"}, nil); len(a) != 1 {
		t.Fatalf("args %v", a)
	}
}
