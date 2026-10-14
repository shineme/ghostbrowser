package main

// Account master key + recovery key.
//
// Cloud data has always been encrypted with encrypt(plain, secret), which runs
// PBKDF2-SHA256 (per-blob salt) over `secret` and seals with AES-256-GCM. Up to
// v0.7 the secret was the sync password itself. This file introduces an
// account master secret M that replaces the password as that `secret`:
//
//   - new accounts (empty cloud): M = 32 random bytes;
//   - existing accounts (cloud already holds data): M = the current password
//     bytes, so every blob already on the server stays decryptable without
//     re-encryption ("legacy" envelope).
//
// M is stored on the server only wrapped, in an opaque JSON envelope:
//
//   pw slot: AES-256-GCM(PBKDF2-SHA256(password, salt, 200000), M)
//   rk slot: AES-256-GCM(HKDF-SHA256(recovery key 160 bit, salt), M)
//   id slot: the account's X25519 sharing identity, sealed with HKDF(M)
//
// Changing the password or rotating the recovery key re-wraps M (data is not
// re-encrypted). Each device keeps M in keyring.json next to data.json, bound
// to the server URL + token it was unlocked for.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const kdfIterPW = 200000

var b64 = base64.StdEncoding

type keySlot struct {
	Salt    string `json:"salt"`
	Nonce   string `json:"nonce"`
	CT      string `json:"ct"`
	Iter    int    `json:"iter,omitempty"`
	Created int64  `json:"created"`
}

type idSlot struct {
	Pub   string `json:"pub"`
	Nonce string `json:"nonce"`
	CT    string `json:"ct"`
}

type keyEnvelope struct {
	V       int     `json:"v"`
	Kid     string  `json:"kid"`
	Legacy  bool    `json:"legacy"`
	Created int64   `json:"created"`
	PW      keySlot `json:"pw"`
	RK      keySlot `json:"rk"`
	ID      *idSlot `json:"id,omitempty"`
}

// keyring is this device's unlocked copy (keyring.json, 0600).
type keyring struct {
	Scope  string `json:"scope"`
	Master string `json:"master"`
	Kid    string `json:"kid"`
	Legacy bool   `json:"legacy"`
	IDPriv string `json:"idPriv,omitempty"`
	IDPub  string `json:"idPub,omitempty"`
}

// RecoveryState is what the Settings card shows.
type RecoveryState struct {
	Supported  bool   `json:"supported"`  // server has the "keys" feature
	Configured bool   `json:"configured"` // an envelope exists on the server
	Unlocked   bool   `json:"unlocked"`   // this device holds the master key
	Legacy     bool   `json:"legacy"`     // master key = the original sync password
	Kid        string `json:"kid"`
	RKCreated  int64  `json:"rkCreated"`
	PWChanged  int64  `json:"pwChanged"`
	ShareReady bool   `json:"shareReady"`
	User       string `json:"user"`
	Error      string `json:"error,omitempty"`
}

func kidOf(m []byte) string {
	h := sha256.Sum256(append([]byte("GBK1 kid\x00"), m...))
	return hex.EncodeToString(h[:8])
}

func syncScope(st Settings) string {
	h := sha256.Sum256([]byte(strings.TrimRight(st.SyncServer, "/") + "\x00" + st.SyncToken))
	return hex.EncodeToString(h[:16])
}

func gcmSeal(key, plain []byte, aad string) (nonce, ct []byte, err error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, g.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, g.Seal(nil, nonce, plain, []byte(aad)), nil
}

func gcmOpen(key []byte, nonceB64, ctB64, aad string) ([]byte, error) {
	nonce, e1 := b64.DecodeString(nonceB64)
	ct, e2 := b64.DecodeString(ctB64)
	if e1 != nil || e2 != nil {
		return nil, fmt.Errorf("malformed key envelope")
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		return nil, err
	}
	if len(nonce) != g.NonceSize() {
		return nil, fmt.Errorf("malformed key envelope")
	}
	return g.Open(nil, nonce, ct, []byte(aad))
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return b
}

func hkdfKey(secret, salt []byte, info string) []byte {
	k, e := hkdf.Key(sha256.New, secret, salt, info, 32)
	if e != nil {
		panic(e)
	}
	return k
}

func wrapPW(m []byte, pw, kid string) (keySlot, error) {
	salt := randBytes(16)
	n, ct, e := gcmSeal(pbkdf2.Key([]byte(pw), salt, kdfIterPW, 32, sha256.New), m, "GBK1|pw|"+kid)
	return keySlot{Salt: b64.EncodeToString(salt), Nonce: b64.EncodeToString(n), CT: b64.EncodeToString(ct), Iter: kdfIterPW, Created: time.Now().Unix()}, e
}

func unwrapPW(s keySlot, pw, kid string) ([]byte, error) {
	salt, e := b64.DecodeString(s.Salt)
	if e != nil || s.Iter < 10000 || s.Iter > 10000000 {
		return nil, fmt.Errorf("malformed key envelope")
	}
	m, e := gcmOpen(pbkdf2.Key([]byte(pw), salt, s.Iter, 32, sha256.New), s.Nonce, s.CT, "GBK1|pw|"+kid)
	if e != nil {
		return nil, errors.New("同步密码不正确，无法解锁账号主密钥")
	}
	return m, nil
}

func wrapRK(m, rk []byte, kid string) (keySlot, error) {
	salt := randBytes(16)
	n, ct, e := gcmSeal(hkdfKey(rk, salt, "GBK1 recovery"), m, "GBK1|rk|"+kid)
	return keySlot{Salt: b64.EncodeToString(salt), Nonce: b64.EncodeToString(n), CT: b64.EncodeToString(ct), Created: time.Now().Unix()}, e
}

func unwrapRK(s keySlot, rk []byte, kid string) ([]byte, error) {
	salt, e := b64.DecodeString(s.Salt)
	if e != nil {
		return nil, fmt.Errorf("malformed key envelope")
	}
	m, e := gcmOpen(hkdfKey(rk, salt, "GBK1 recovery"), s.Nonce, s.CT, "GBK1|rk|"+kid)
	if e != nil {
		return nil, errors.New("恢复密钥不正确")
	}
	return m, nil
}

func sealIdentity(m []byte, kid string, priv *ecdh.PrivateKey) (*idSlot, error) {
	n, ct, e := gcmSeal(hkdfKey(m, nil, "GBK1 identity"), priv.Bytes(), "GBK1|id|"+kid)
	if e != nil {
		return nil, e
	}
	return &idSlot{Pub: b64.EncodeToString(priv.PublicKey().Bytes()), Nonce: b64.EncodeToString(n), CT: b64.EncodeToString(ct)}, nil
}

func openIdentity(m []byte, kid string, s *idSlot) (*ecdh.PrivateKey, error) {
	if s == nil {
		return nil, errors.New("no sharing identity")
	}
	raw, e := gcmOpen(hkdfKey(m, nil, "GBK1 identity"), s.Nonce, s.CT, "GBK1|id|"+kid)
	if e != nil {
		return nil, e
	}
	return ecdh.X25519().NewPrivateKey(raw)
}

// ---------------------------------------------------------------- recovery key text

// newRecoveryKey: 160 random bits as 8 groups of 4 base32 characters.
func newRecoveryKey() ([]byte, string) {
	raw := randBytes(20)
	return raw, formatRecoveryKey(raw)
}

func formatRecoveryKey(raw []byte) string {
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	parts := []string{}
	for i := 0; i < len(s); i += 4 {
		parts = append(parts, s[i:min(i+4, len(s))])
	}
	return strings.Join(parts, "-")
}

func parseRecoveryKey(s string) ([]byte, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '2' && r <= '7':
			b.WriteRune(r)
		case r == '0':
			b.WriteRune('O') // common misreads
		case r == '1':
			b.WriteRune('I')
		}
	}
	raw, e := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b.String())
	if e != nil || len(raw) != 20 {
		return nil, errors.New("恢复密钥格式不正确：应为 32 位字母数字（8 组，每组 4 位）")
	}
	return raw, nil
}

// ---------------------------------------------------------------- local keyring

var keyringMu sync.Mutex

func (s *Store) keyringPath() string { return filepath.Join(s.dir, "keyring.json") }

func (s *Store) loadKeyring() *keyring {
	keyringMu.Lock()
	defer keyringMu.Unlock()
	b, e := os.ReadFile(s.keyringPath())
	if e != nil {
		return nil
	}
	var k keyring
	if json.Unmarshal(b, &k) != nil || k.Scope != syncScope(s.GetSettings()) || k.Master == "" {
		return nil
	}
	return &k
}

func (s *Store) saveKeyring(k *keyring) error {
	keyringMu.Lock()
	defer keyringMu.Unlock()
	b, _ := json.MarshalIndent(k, "", "  ")
	tmp := s.keyringPath() + ".tmp"
	if e := os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, s.keyringPath())
}

func (k *keyring) master() []byte { m, _ := b64.DecodeString(k.Master); return m }

func (k *keyring) identity() (*ecdh.PrivateKey, error) {
	raw, e := b64.DecodeString(k.IDPriv)
	if e != nil || len(raw) != 32 {
		return nil, errors.New("请先在「设置 → 云同步」设置恢复密钥，以启用环境分享")
	}
	return ecdh.X25519().NewPrivateKey(raw)
}

func newKeyring(st Settings, m []byte, env *keyEnvelope) *keyring {
	k := &keyring{Scope: syncScope(st), Master: b64.EncodeToString(m), Kid: env.Kid, Legacy: env.Legacy}
	if priv, e := openIdentity(m, env.Kid, env.ID); e == nil {
		k.IDPriv = b64.EncodeToString(priv.Bytes())
		k.IDPub = b64.EncodeToString(priv.PublicKey().Bytes())
	}
	return k
}

// ---------------------------------------------------------------- server envelope

var envCache struct {
	sync.Mutex
	scope string
	at    time.Time
	env   *keyEnvelope
}

func (s *Store) keysSupported() bool {
	si, e := s.ServerInfo(false)
	return e == nil && si.has("keys")
}

func (s *Store) fetchKeys() (*keyEnvelope, int64, error) {
	var out struct {
		Epoch    int64           `json:"epoch"`
		Envelope json.RawMessage `json:"envelope"`
	}
	if e := s.sync().jsonRequest("GET", "/api/keys", nil, &out); e != nil {
		return nil, 0, e
	}
	var env *keyEnvelope
	if len(out.Envelope) > 0 && !bytes.Equal(out.Envelope, []byte("null")) {
		env = &keyEnvelope{}
		if e := json.Unmarshal(out.Envelope, env); e != nil || env.V != 1 || env.Kid == "" {
			return nil, 0, fmt.Errorf("unrecognised key envelope on the server")
		}
	}
	envCache.Lock()
	envCache.scope, envCache.at, envCache.env = syncScope(s.GetSettings()), time.Now(), env
	envCache.Unlock()
	return env, out.Epoch, nil
}

func (s *Store) putKeys(env *keyEnvelope, epoch int64) error {
	var out struct {
		Epoch int64 `json:"epoch"`
	}
	e := s.sync().jsonRequest("PUT", "/api/keys", map[string]any{"envelope": env, "epoch": epoch}, &out)
	envCache.Lock()
	envCache.at = time.Time{}
	envCache.Unlock()
	return e
}

// dataSecret is the `secret` handed to encrypt/decrypt for cloud data: the
// unlocked master key, or (accounts without an envelope) the sync password. A
// device that has no keyring yet unlocks itself with the stored password.
func (s *Store) dataSecret() ([]byte, error) {
	st := s.GetSettings()
	if k := s.loadKeyring(); k != nil {
		return k.master(), nil
	}
	if !s.keysSupported() {
		return []byte(st.SyncPassword), nil
	}
	envCache.Lock()
	env, fresh := envCache.env, envCache.scope == syncScope(st) && time.Since(envCache.at) < 2*time.Minute
	envCache.Unlock()
	if !fresh {
		var e error
		if env, _, e = s.fetchKeys(); e != nil {
			return nil, e
		}
	}
	if env == nil {
		return []byte(st.SyncPassword), nil // no master key yet: legacy password key
	}
	m, e := unwrapPW(env.PW, st.SyncPassword, env.Kid)
	if e != nil || kidOf(m) != env.Kid {
		return nil, errors.New("云端账号已启用主密钥，但本机的同步密码无法解锁（可能已在其他设备修改）。请在「设置 → 云同步 → 账号密钥」用新密码或恢复密钥解锁")
	}
	_ = s.saveKeyring(newKeyring(st, m, env))
	return m, nil
}

// ---------------------------------------------------------------- operations

func (s *Store) RecoveryStatus() RecoveryState {
	st := s.GetSettings()
	var r RecoveryState
	if e := syncReady(st, false); e != nil {
		r.Error = e.Error()
		return r
	}
	si, e := s.ServerInfo(true)
	if e != nil {
		r.Error = e.Error()
		return r
	}
	r.User = si.User
	r.Supported = si.has("keys")
	if !r.Supported {
		return r
	}
	env, _, e := s.fetchKeys()
	if e != nil {
		r.Error = e.Error()
		return r
	}
	k := s.loadKeyring()
	if env != nil {
		r.Configured, r.Legacy, r.Kid, r.RKCreated, r.PWChanged = true, env.Legacy, env.Kid, env.RK.Created, env.PW.Created
		if k == nil && st.SyncPassword != "" { // try the stored password silently
			if _, e := s.dataSecret(); e == nil {
				k = s.loadKeyring()
			}
		}
		r.Unlocked = k != nil && k.Kid == env.Kid
		r.ShareReady = r.Unlocked && k.IDPriv != "" && env.ID != nil
	}
	return r
}

// SetupRecovery creates the master key envelope and returns the recovery key
// (shown once). Existing cloud data keeps working: when the account already
// has cloud records the master key is the current password (verified against
// the encrypted global record when there is one).
func (s *Store) SetupRecovery() (string, error) {
	st := s.GetSettings()
	if e := syncReady(st, true); e != nil {
		return "", e
	}
	if !s.keysSupported() {
		return "", errors.New("服务器版本过旧，不支持恢复密钥：请更新 ghostsync.php")
	}
	env, epoch, e := s.fetchKeys()
	if e != nil {
		return "", e
	}
	if env != nil {
		return "", errors.New("此同步账号已设置过恢复密钥：如需新密钥请使用「轮换恢复密钥」")
	}
	live, tombs, e := s.listCloudFull()
	if e != nil {
		return "", e
	}
	var m []byte
	legacy := len(live) > 0 || len(tombs) > 0
	if legacy {
		m = []byte(st.SyncPassword)
		if e := s.verifyLegacyPassword(live, m); e != nil {
			return "", e
		}
	} else {
		m = randBytes(32)
	}
	rkRaw, rkText := newRecoveryKey()
	env, e = buildEnvelope(m, st.SyncPassword, rkRaw, legacy)
	if e != nil {
		return "", e
	}
	if e = s.putKeys(env, epoch); e != nil {
		return "", e
	}
	k := newKeyring(st, m, env)
	if e = s.saveKeyring(k); e != nil {
		return "", e
	}
	_ = s.publishIdentity(k)
	return rkText, nil
}

func buildEnvelope(m []byte, pw string, rk []byte, legacy bool) (*keyEnvelope, error) {
	kid := kidOf(m)
	env := &keyEnvelope{V: 1, Kid: kid, Legacy: legacy, Created: time.Now().Unix()}
	var e error
	if env.PW, e = wrapPW(m, pw, kid); e != nil {
		return nil, e
	}
	if env.RK, e = wrapRK(m, rk, kid); e != nil {
		return nil, e
	}
	priv, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		return nil, e
	}
	if env.ID, e = sealIdentity(m, kid, priv); e != nil {
		return nil, e
	}
	return env, nil
}

// verifyLegacyPassword decrypts the small global record (when present) so a
// wrong local password is never sealed as the account master key.
func (s *Store) verifyLegacyPassword(live []cloudMeta, m []byte) error {
	for _, c := range live {
		if c.ID != globalID {
			continue
		}
		resp, e := s.sync().req("GET", "/api/profiles/"+globalID+"/blob", nil, "")
		if e != nil {
			return e
		}
		defer resp.Body.Close()
		if e = syncStatus(resp); e != nil {
			return e
		}
		enc, e := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		if e != nil {
			return e
		}
		if _, e = decrypt(enc, m); e != nil {
			return errors.New("当前同步密码无法解密云端已有数据：请先在设置中填写正确的同步密码，再设置恢复密钥")
		}
	}
	return nil
}

// UnlockWithPassword unlocks this device (e.g. a new computer) with the sync
// password and stores it as the local sync password.
func (s *Store) UnlockWithPassword(pw string) error {
	env, _, e := s.fetchKeys()
	if e != nil {
		return e
	}
	if env == nil {
		return errors.New("此同步账号还没有设置账号密钥")
	}
	m, e := unwrapPW(env.PW, pw, env.Kid)
	if e != nil {
		return e
	}
	st := s.GetSettings()
	st.SyncPassword = pw
	s.PutSettings(st)
	k := newKeyring(s.GetSettings(), m, env)
	if e = s.saveKeyring(k); e != nil {
		return e
	}
	_ = s.publishIdentity(k)
	return nil
}

// UnlockWithRecoveryKey unlocks with the recovery key and sets a new sync
// password (the old one is presumably forgotten).
func (s *Store) UnlockWithRecoveryKey(rkText, newPW string) error {
	if len(newPW) < 8 {
		return errors.New("新同步密码至少 8 位")
	}
	rk, e := parseRecoveryKey(rkText)
	if e != nil {
		return e
	}
	env, epoch, e := s.fetchKeys()
	if e != nil {
		return e
	}
	if env == nil {
		return errors.New("此同步账号还没有设置账号密钥")
	}
	m, e := unwrapRK(env.RK, rk, env.Kid)
	if e != nil {
		return e
	}
	if env.PW, e = wrapPW(m, newPW, env.Kid); e != nil {
		return e
	}
	if e = s.putKeys(env, epoch); e != nil {
		return e
	}
	st := s.GetSettings()
	st.SyncPassword = newPW
	s.PutSettings(st)
	k := newKeyring(s.GetSettings(), m, env)
	if e = s.saveKeyring(k); e != nil {
		return e
	}
	_ = s.publishIdentity(k)
	return nil
}

// ChangeSyncPassword re-wraps the master key; cloud data is not re-encrypted.
func (s *Store) ChangeSyncPassword(oldPW, newPW string) error {
	if len(newPW) < 8 {
		return errors.New("新同步密码至少 8 位")
	}
	if oldPW == newPW {
		return errors.New("新密码与当前密码相同")
	}
	env, epoch, e := s.fetchKeys()
	if e != nil {
		return e
	}
	if env == nil {
		return errors.New("请先设置恢复密钥（会启用账号主密钥），之后修改密码不需要重新加密云端数据")
	}
	m, e := unwrapPW(env.PW, oldPW, env.Kid)
	if e != nil {
		return errors.New("当前同步密码不正确")
	}
	if env.PW, e = wrapPW(m, newPW, env.Kid); e != nil {
		return e
	}
	if e = s.putKeys(env, epoch); e != nil {
		return e
	}
	st := s.GetSettings()
	st.SyncPassword = newPW
	s.PutSettings(st)
	return s.saveKeyring(newKeyring(s.GetSettings(), m, env))
}

// RotateRecoveryKey replaces the recovery key (the old one stops working).
func (s *Store) RotateRecoveryKey(pw string) (string, error) {
	env, epoch, e := s.fetchKeys()
	if e != nil {
		return "", e
	}
	if env == nil {
		return "", errors.New("此同步账号还没有设置账号密钥")
	}
	m, e := unwrapPW(env.PW, pw, env.Kid)
	if e != nil {
		return "", e
	}
	rkRaw, rkText := newRecoveryKey()
	if env.RK, e = wrapRK(m, rkRaw, env.Kid); e != nil {
		return "", e
	}
	if e = s.putKeys(env, epoch); e != nil {
		return "", e
	}
	return rkText, s.saveKeyring(newKeyring(s.GetSettings(), m, env))
}
