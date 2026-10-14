// sign-release creates the Ed25519 key pair for GhostBrowser's in-app
// updater and signs release EXEs into a latest.json.
//
//	go run ./cmd/sign-release keygen -out keys
//	    writes keys/ghostbrowser-update.key (PRIVATE: keep it offline, never commit it)
//	    and keys/ghostbrowser-update.pub (paste into updater.go updatePubKey, or
//	    build with -ldflags "-X main.updatePubKey=<pub>")
//
//	go run ./cmd/sign-release sign -key keys/ghostbrowser-update.key \
//	    -file build/bin/GhostBrowser.exe -version 0.7.3 -notes "更新说明" -out update/latest.json
//	    then upload latest.json and the EXE into update/ next to ghostsync.php.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"ghostbrowser/internal/updsig"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sign-release keygen -out DIR | sign -key FILE -file EXE -version X.Y.Z [-url NAME] [-notes TEXT|-notes-file F] [-out latest.json]")
	os.Exit(2)
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", ".", "output directory")
	force := fs.Bool("force", false, "overwrite an existing key")
	_ = fs.Parse(args)
	keyPath := filepath.Join(*out, "ghostbrowser-update.key")
	if _, err := os.Stat(keyPath); err == nil && !*force {
		return fmt.Errorf("%s already exists (use -force to replace; old clients will then reject new releases)", keyPath)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600); err != nil {
		return err
	}
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	if err := os.WriteFile(filepath.Join(*out, "ghostbrowser-update.pub"), []byte(pubB64+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Println("public key:", pubB64)
	fmt.Println("private key written to", keyPath, "- keep it secret and backed up; it is the only way to sign updates.")
	return nil
}

func sign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	keyPath := fs.String("key", "ghostbrowser-update.key", "private key file")
	file := fs.String("file", "", "release EXE")
	version := fs.String("version", "", "release version, e.g. 0.7.3")
	urlName := fs.String("url", "", "download URL or file name in update/ (default: the EXE's base name)")
	notes := fs.String("notes", "", "release notes (markdown)")
	notesFile := fs.String("notes-file", "", "read release notes from a file")
	out := fs.String("out", "latest.json", "output latest.json")
	_ = fs.Parse(args)
	if *file == "" || *version == "" {
		return fmt.Errorf("-file and -version are required")
	}
	raw, err := os.ReadFile(*keyPath)
	if err != nil {
		return err
	}
	priv, err := updsig.ParsePrivate(string(raw))
	if err != nil {
		return err
	}
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		f.Close()
		return err
	}
	f.Close()
	sum := hex.EncodeToString(h.Sum(nil))
	if *notesFile != "" {
		b, err := os.ReadFile(*notesFile)
		if err != nil {
			return err
		}
		*notes = string(b)
	}
	if *urlName == "" {
		*urlName = filepath.Base(*file)
	}
	doc := map[string]string{"version": *version, "notes": *notes, "url": *urlName, "sha256": sum, "sig": updsig.Sign(priv, *version, sum)}
	b, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", *out, "sha256", sum)
	return nil
}
