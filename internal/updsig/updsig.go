// Package updsig defines what an update signature covers, shared by the
// client (verify) and cmd/sign-release (sign).
//
// The Ed25519 signature is over the message
//
//	"ghostbrowser-update\n" + version + "\n" + lowercase hex sha256 of the file
//
// so a signature binds one exact file to one version: a valid old release
// cannot be served under a newer version number.
package updsig

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Message returns the bytes that are signed for a release.
func Message(version, sha256hex string) []byte {
	return []byte("ghostbrowser-update\n" + strings.TrimPrefix(strings.TrimSpace(version), "v") + "\n" + strings.ToLower(strings.TrimSpace(sha256hex)))
}

// Sign signs a release with a private key.
func Sign(priv ed25519.PrivateKey, version, sha256hex string) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, Message(version, sha256hex)))
}

// ParsePublic decodes a base64 Ed25519 public key.
func ParsePublic(b64 string) (ed25519.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("更新公钥无效")
	}
	return ed25519.PublicKey(b), nil
}

// ParsePrivate decodes a base64 Ed25519 private key (64 bytes) or seed (32 bytes).
func ParsePrivate(b64 string) (ed25519.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("私钥不是 base64：%w", err)
	}
	switch len(b) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(b), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(b), nil
	}
	return nil, errors.New("私钥长度不对")
}

// Verify checks a base64 signature for (version, sha256).
func Verify(pub ed25519.PublicKey, version, sha256hex, sigB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("更新包签名格式无效")
	}
	if !ed25519.Verify(pub, Message(version, sha256hex), sig) {
		return errors.New("更新包签名校验失败：文件可能被篡改或不是官方发布，已拒绝安装")
	}
	return nil
}
