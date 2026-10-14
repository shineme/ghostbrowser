package main

// Profile sharing between users of the same sync server.
//
// Every account has an X25519 identity (created with the recovery key, its
// private half sealed under the master key in the key envelope) whose public
// half is published in the server's directory. To share, the sender
// packs the profile (config only, or config + browser data/session) into a
// zip, encrypts it with a random 32-byte share key K (GBE1 format) and wraps
// K for the recipient: ephemeral X25519 -> HKDF-SHA256 -> AES-256-GCM. The
// server stores the encrypted blob and the wrapped key only.
//
// An accepted share becomes a local profile "sh-<share id>" marked with
// ShareID; it is never pushed to the recipient's own cloud and is removed on
// the next share sync once the share is revoked, expired or left.

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var shareIDPattern = regexp.MustCompile(`^[a-f0-9]{24}$`)

type shareWrap struct {
	Eph   string `json:"eph"`
	Nonce string `json:"nonce"`
	CT    string `json:"ct"`
	Pub   string `json:"pub,omitempty"` // recipient key the share was wrapped for
}

type ShareInfo struct {
	ID        string     `json:"id"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	Name      string     `json:"name"`
	Mode      string     `json:"mode"` // session | config
	Note      string     `json:"note"`
	OS        string     `json:"os"`
	CreatedAt int64      `json:"createdAt"`
	ExpiresAt int64      `json:"expiresAt"`
	UpdatedAt int64      `json:"updatedAt"`
	Size      int64      `json:"size"`
	Status    string     `json:"status"` // pending | accepted | declined | revoked | left | expired
	Wrap      *shareWrap `json:"wrap,omitempty"`
	LocalID   string     `json:"localId,omitempty"` // recipient: the local profile of an accepted share
}

type ShareList struct {
	Supported bool        `json:"supported"`
	Ready     bool        `json:"ready"` // this device holds the sharing identity
	Self      string      `json:"self"`
	Out       []ShareInfo `json:"out"`
	In        []ShareInfo `json:"in"`
	Removed   int         `json:"removed"` // local copies removed by this sync
	Error     string      `json:"error,omitempty"`
}

type ShareRequest struct {
	ProfileID string `json:"profileId"`
	To        string `json:"to"`
	Mode      string `json:"mode"`
	Days      int    `json:"days"`
	Note      string `json:"note"`
}

func shareLocalID(sid string) string { return "sh-" + sid }

func (s *Store) publishIdentity(k *keyring) error {
	if k == nil || k.IDPub == "" {
		return errors.New("no sharing identity")
	}
	return s.sync().jsonRequest("POST", "/api/directory", map[string]string{"pub": k.IDPub}, nil)
}

func (s *Store) shareSupported() (serverInfo, error) {
	if e := syncReady(s.GetSettings(), false); e != nil {
		return serverInfo{}, e
	}
	si, e := s.ServerInfo(false)
	if e != nil {
		return si, e
	}
	if !si.has("shares") {
		return si, errors.New("服务器版本过旧，不支持环境分享：请更新 ghostsync.php")
	}
	return si, nil
}

// shareKeyring returns the unlocked keyring with a sharing identity.
func (s *Store) shareKeyring() (*keyring, error) {
	if _, e := s.dataSecret(); e != nil { // unlocks with the stored password when possible
		return nil, e
	}
	k := s.loadKeyring()
	if k == nil || k.IDPriv == "" {
		return nil, errors.New("请先在「设置 → 云同步 → 账号密钥」设置恢复密钥，以启用环境分享")
	}
	return k, nil
}

// ShareUsers lists the other users on the server who can receive shares.
func (s *Store) ShareUsers() ([]string, error) {
	if _, e := s.shareSupported(); e != nil {
		return nil, e
	}
	if k, e := s.shareKeyring(); e == nil {
		_ = s.publishIdentity(k)
	}
	var out struct {
		Users []struct {
			User string `json:"user"`
		} `json:"users"`
	}
	if e := s.sync().jsonRequest("GET", "/api/directory", nil, &out); e != nil {
		return nil, e
	}
	names := []string{}
	for _, u := range out.Users {
		names = append(names, u.User)
	}
	return names, nil
}

func wrapShareKey(key []byte, recipientPub []byte) (*shareWrap, error) {
	pub, e := ecdh.X25519().NewPublicKey(recipientPub)
	if e != nil {
		return nil, e
	}
	eph, e := ecdh.X25519().GenerateKey(rand.Reader)
	if e != nil {
		return nil, e
	}
	ss, e := eph.ECDH(pub)
	if e != nil {
		return nil, e
	}
	salt := append(append([]byte{}, eph.PublicKey().Bytes()...), recipientPub...)
	n, ct, e := gcmSeal(hkdfKey(ss, salt, "GBS1 share key"), key, "GBS1")
	if e != nil {
		return nil, e
	}
	return &shareWrap{Eph: b64.EncodeToString(eph.PublicKey().Bytes()), Nonce: b64.EncodeToString(n), CT: b64.EncodeToString(ct)}, nil
}

func unwrapShareKey(w *shareWrap, priv *ecdh.PrivateKey) ([]byte, error) {
	if w == nil {
		return nil, errors.New("分享缺少密钥信息")
	}
	ephRaw, e := b64.DecodeString(w.Eph)
	if e != nil {
		return nil, e
	}
	eph, e := ecdh.X25519().NewPublicKey(ephRaw)
	if e != nil {
		return nil, e
	}
	ss, e := priv.ECDH(eph)
	if e != nil {
		return nil, e
	}
	mine := priv.PublicKey().Bytes()
	salt := append(append([]byte{}, ephRaw...), mine...)
	k, e := gcmOpen(hkdfKey(ss, salt, "GBS1 share key"), w.Nonce, w.CT, "GBS1")
	if e != nil {
		return nil, errors.New("无法解开分享密钥：此分享是用你的旧分享身份加密的，请让对方重新分享")
	}
	return k, nil
}

// CreateShare packs and encrypts a profile for another user.
func (s *Store) CreateShare(r ShareRequest) (ShareInfo, error) {
	var out ShareInfo
	si, e := s.shareSupported()
	if e != nil {
		return out, e
	}
	if r.Mode != "session" && r.Mode != "config" {
		return out, errors.New("请选择分享内容")
	}
	if r.Days < 1 || r.Days > 366 {
		return out, errors.New("有效期必须在 1–366 天之间")
	}
	r.To = strings.TrimSpace(r.To)
	if r.To == "" {
		return out, errors.New("请选择接收者")
	}
	if r.To == si.User {
		return out, errors.New("不能分享给自己")
	}
	k, e := s.shareKeyring()
	if e != nil {
		return out, e
	}
	_ = s.publishIdentity(k)
	p, ok := s.GetProfile(r.ProfileID)
	if !ok {
		return out, errors.New("profile not found")
	}
	if p.ShareID != "" {
		return out, errors.New("共享给你的环境不能再转分享")
	}
	if _, running := runReg.get(p.ID); running && r.Mode == "session" {
		return out, errors.New("请先关闭该环境的浏览器，再分享登录状态")
	}
	var dir struct {
		Pub string `json:"pub"`
	}
	c := s.sync()
	if e = c.jsonRequest("GET", "/api/directory/"+url.PathEscape(r.To), nil, &dir); e != nil {
		return out, e
	}
	pub, e := b64.DecodeString(dir.Pub)
	if e != nil || len(pub) != 32 {
		return out, errors.New("对方的分享公钥无效")
	}
	// Archive: resolved inline proxy, no sync bookkeeping of the sender.
	pb, _ := json.Marshal(p)
	var a Profile
	_ = json.Unmarshal(pb, &a)
	if a.InlineProxy == nil && a.ProxyID != "" {
		if px, ok := s.GetProxy(a.ProxyID); ok {
			a.InlineProxy = px
		}
	}
	a.ProxyID, a.Storage, a.CloudVer, a.SyncedAt, a.KeepLocal, a.ChunkKey = "", "local", 0, 0, 0, ""
	src := ""
	if r.Mode == "session" {
		if _, src, e = s.syncDir(p.ID); e != nil {
			return out, e
		}
		if _, e := os.Stat(src); e != nil {
			src = ""
		}
	} else {
		a.Cookies = ""
	}
	if src == "" {
		if src, e = os.MkdirTemp("", "gbshare-"); e != nil {
			return out, e
		}
		defer os.RemoveAll(src)
	}
	raw, e := zipDir(src, &a)
	if e != nil {
		return out, e
	}
	key := randBytes(32)
	enc, e := encrypt(raw, key)
	if e != nil {
		return out, e
	}
	if si.MaxBody > 0 && int64(len(enc)) > si.MaxBody {
		return out, fmt.Errorf("分享包 %d MB 超过服务器单次上传上限 %d MB：可改为「仅配置」分享，或调大 PHP post_max_size", len(enc)>>20, si.MaxBody>>20)
	}
	w, e := wrapShareKey(key, pub)
	if e != nil {
		return out, e
	}
	var created struct {
		ID        string `json:"id"`
		ExpiresAt int64  `json:"expiresAt"`
	}
	if e = c.jsonRequest("POST", "/api/shares", map[string]any{"to": r.To, "name": p.Name, "mode": r.Mode, "note": r.Note,
		"days": r.Days, "os": p.Fingerprint.OS, "wrap": w}, &created); e != nil {
		return out, e
	}
	if !shareIDPattern.MatchString(created.ID) {
		return out, errors.New("sync server returned an invalid share ID")
	}
	resp, e := c.req("PUT", "/api/shares/"+created.ID+"/blob", bytes.NewReader(enc), "application/octet-stream")
	if e == nil {
		e = syncStatus(resp)
		resp.Body.Close()
	}
	if e != nil {
		_ = c.jsonRequest("POST", "/api/shares/"+created.ID+"/revoke", map[string]any{}, nil)
		return out, e
	}
	s.auditNamed("ui", "share.create", p.ID, p.Name, nil, "分享给 "+r.To)
	return ShareInfo{ID: created.ID, From: si.User, To: r.To, Name: p.Name, Mode: r.Mode, Note: r.Note, OS: p.Fingerprint.OS,
		CreatedAt: time.Now().Unix(), ExpiresAt: created.ExpiresAt, Size: int64(len(enc)), Status: "pending"}, nil
}

func (s *Store) fetchShares() (ShareList, error) {
	out := ShareList{Out: []ShareInfo{}, In: []ShareInfo{}}
	var raw struct {
		Self string      `json:"self"`
		Out  []ShareInfo `json:"out"`
		In   []ShareInfo `json:"in"`
	}
	if e := s.sync().jsonRequest("GET", "/api/shares", nil, &raw); e != nil {
		return out, e
	}
	out.Self = raw.Self
	if raw.Out != nil {
		out.Out = raw.Out
	}
	if raw.In != nil {
		out.In = raw.In
	}
	return out, nil
}

// ListShares fetches both directions and reconciles local copies of received
// shares (revoked / expired / left ones are removed).
func (s *Store) ListShares() ShareList {
	out := ShareList{Out: []ShareInfo{}, In: []ShareInfo{}}
	if _, e := s.shareSupported(); e != nil {
		out.Error = e.Error()
		return out
	}
	out.Supported = true
	if k, e := s.shareKeyring(); e == nil {
		out.Ready = true
		_ = s.publishIdentity(k)
	}
	l, e := s.fetchShares()
	if e != nil {
		out.Error = e.Error()
		return out
	}
	l.Supported, l.Ready = true, out.Ready
	l.Removed = s.applyShares(l.In, true)
	for i := range l.In {
		l.In[i].Wrap = nil // never hand key material to the UI
		if p, ok := s.GetProfile(shareLocalID(l.In[i].ID)); ok && p.ShareID == l.In[i].ID {
			l.In[i].LocalID = p.ID
		}
	}
	return l
}

// applyShares removes local copies whose share ended and updates expiry
// dates. online=false only applies the local expiry dates.
func (s *Store) applyShares(in []ShareInfo, online bool) (removed int) {
	byID := map[string]ShareInfo{}
	for _, sh := range in {
		byID[sh.ID] = sh
	}
	now := time.Now().Unix()
	for _, p := range s.ListProfiles() {
		if p.ShareID == "" {
			continue
		}
		sh, ok := byID[p.ShareID]
		gone := p.ShareExpires > 0 && p.ShareExpires <= now
		if online {
			gone = gone || !ok || sh.Status != "accepted"
		}
		if !gone {
			if online && sh.ExpiresAt != p.ShareExpires {
				p.ShareExpires = sh.ExpiresAt
				s.PutProfile(p)
			}
			continue
		}
		if _, running := runReg.get(p.ID); running {
			continue // removed on a later sync once the browser is closed
		}
		s.DeleteProfile(p.ID)
		reason := "分享已结束"
		switch sh.Status {
		case "revoked":
			reason = "分享已被 " + p.SharedFrom + " 撤销"
		case "expired":
			reason = "分享已过期"
		case "left":
			reason = "已退出分享"
		}
		if p.ShareExpires > 0 && p.ShareExpires <= now {
			reason = "分享已过期"
		}
		s.auditNamed("sync", "share.remove-local", p.ID, p.Name, nil, reason+"，已删除本机副本")
		removed++
	}
	return removed
}

// SyncShares is the periodic reconcile (also used before launching a shared profile).
func (s *Store) SyncShares() (int, error) {
	if _, e := s.shareSupported(); e != nil {
		return s.applyShares(nil, false), e
	}
	l, e := s.fetchShares()
	if e != nil {
		return s.applyShares(nil, false), e
	}
	return s.applyShares(l.In, true), nil
}

// shareLaunchGuard refuses to start a received profile whose share ended.
func (s *Store) shareLaunchGuard(id string) error {
	p, ok := s.GetProfile(id)
	if !ok || p.ShareID == "" {
		return nil
	}
	_, _ = s.SyncShares()
	if _, ok := s.GetProfile(id); !ok {
		return fmt.Errorf("环境「%s」的分享已结束（撤销或过期），本机副本已删除", p.Name)
	}
	return nil
}

// AcceptShare downloads, decrypts and installs a received share as a local profile.
func (s *Store) AcceptShare(sid string) (Profile, error) {
	var p Profile
	if !shareIDPattern.MatchString(sid) {
		return p, errors.New("invalid share ID")
	}
	if _, e := s.shareSupported(); e != nil {
		return p, e
	}
	k, e := s.shareKeyring()
	if e != nil {
		return p, e
	}
	priv, e := k.identity()
	if e != nil {
		return p, e
	}
	l, e := s.fetchShares()
	if e != nil {
		return p, e
	}
	var sh *ShareInfo
	for i := range l.In {
		if l.In[i].ID == sid {
			sh = &l.In[i]
		}
	}
	if sh == nil {
		return p, errors.New("找不到该分享")
	}
	key, e := unwrapShareKey(sh.Wrap, priv)
	if e != nil {
		return p, e
	}
	c := s.sync()
	resp, e := c.req("GET", "/api/shares/"+sid+"/blob", nil, "")
	if e != nil {
		return p, e
	}
	defer resp.Body.Close()
	if e = syncStatus(resp); e != nil {
		return p, e
	}
	enc, e := io.ReadAll(io.LimitReader(resp.Body, maxSyncBlob+1))
	if e != nil {
		return p, e
	}
	if len(enc) > maxSyncBlob {
		return p, errors.New("分享包过大")
	}
	sum := sha256.Sum256(enc)
	if want := resp.Header.Get("X-Blob-SHA256"); want != "" && !strings.EqualFold(want, hex.EncodeToString(sum[:])) {
		return p, errors.New("分享包校验失败，请重试")
	}
	raw, e := decrypt(enc, key)
	if e != nil {
		return p, errors.New("分享包解密失败")
	}
	id := shareLocalID(sid)
	root, dir, e := s.syncDir(id)
	if e != nil {
		return p, e
	}
	stage, e := os.MkdirTemp(root, ".share-"+sid+"-")
	if e != nil {
		return p, e
	}
	defer os.RemoveAll(stage)
	if e = unzipDir(raw, stage); e != nil {
		return p, e
	}
	meta, e := os.ReadFile(filepath.Join(stage, profileArchiveMetadata))
	if e != nil {
		return p, errors.New("分享包缺少环境配置")
	}
	if e = json.Unmarshal(meta, &p); e != nil {
		return p, e
	}
	_ = os.Remove(filepath.Join(stage, profileArchiveMetadata))
	p.ID, p.Storage, p.CloudVer, p.SyncedAt, p.KeepLocal, p.ChunkKey, p.ProxyID = id, "local", 0, 0, 0, "", ""
	p.ShareID, p.SharedFrom, p.ShareMode, p.ShareExpires = sid, sh.From, sh.Mode, sh.ExpiresAt
	p.CreatedAt = time.Now().Unix()
	if e = c.jsonRequest("POST", "/api/shares/"+sid+"/accept", map[string]any{}, nil); e != nil {
		return p, e
	}
	_ = os.RemoveAll(dir)
	if e = os.Rename(stage, dir); e != nil {
		return p, fmt.Errorf("安装分享环境失败: %w", e)
	}
	s.PutProfile(p)
	s.auditNamed("ui", "share.accept", p.ID, p.Name, nil, "来自 "+sh.From)
	return p, nil
}

func (s *Store) shareAction(sid, act string, body map[string]any) error {
	if !shareIDPattern.MatchString(sid) {
		return errors.New("invalid share ID")
	}
	if body == nil {
		body = map[string]any{}
	}
	return s.sync().jsonRequest("POST", "/api/shares/"+sid+"/"+act, body, nil)
}

func (s *Store) DeclineShare(sid string) error { return s.shareAction(sid, "decline", nil) }
func (s *Store) RevokeShare(sid string) error  { return s.shareAction(sid, "revoke", nil) }

func (s *Store) ExtendShare(sid string, days int) error {
	if days < 1 || days > 366 {
		return errors.New("续期天数必须在 1–366 天之间")
	}
	return s.shareAction(sid, "extend", map[string]any{"days": days})
}

// LeaveShare: the recipient stops using a share; the local copy is removed.
func (s *Store) LeaveShare(sid string) error {
	id := shareLocalID(sid)
	if _, running := runReg.get(id); running {
		return errors.New("请先关闭该环境的浏览器")
	}
	if e := s.shareAction(sid, "leave", nil); e != nil {
		return e
	}
	if p, ok := s.GetProfile(id); ok && p.ShareID == sid {
		s.DeleteProfile(id)
	}
	return nil
}
