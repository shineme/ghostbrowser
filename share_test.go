package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// GHOST_PHP_TEST=php go test -run 'TestSec' .
type secEnv struct {
	t      *testing.T
	root   string
	base   string
	server string
	tokens map[string]string // user -> token
	admin  string
}

func startSecServer(t *testing.T) *secEnv {
	php := os.Getenv("GHOST_PHP_TEST")
	if php == "" {
		t.Skip("set GHOST_PHP_TEST to php")
	}
	project, _ := os.Getwd()
	root := t.TempDir()
	port := freePort()
	env := &secEnv{t: t, root: root, base: "http://127.0.0.1:" + itoa(port) + "/ghostsync.php", server: filepath.Join(root, "server"),
		tokens: map[string]string{"owner": strings.Repeat("owner-private-token-", 3), "bob": strings.Repeat("bob-private-token-xx", 3), "carol": strings.Repeat("carol-private-token-", 3)},
		admin:  strings.Repeat("admin-token-", 4)}
	cmd := exec.Command(php, "-S", "127.0.0.1:"+itoa(port), "-t", filepath.Join(project, "cloud-server"))
	cmd.Env = append(os.Environ(), "GHOST_SYNC_TOKEN="+env.tokens["owner"], "GHOST_SYNC_DATA_DIR="+env.server,
		"GHOST_SYNC_USERS="+env.tokens["bob"]+"=bob;"+env.tokens["carol"]+"=carol", "GHOST_SYNC_ADMIN_TOKEN="+env.admin)
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	deadline := time.Now().Add(10 * time.Second)
	for {
		if r, e := http.Get(env.base); e == nil {
			r.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("PHP did not start")
		}
		time.Sleep(100 * time.Millisecond)
	}
	return env
}

func (env *secEnv) dev(user, name, password string) *Store {
	d := filepath.Join(env.root, user+"-"+name)
	_ = os.MkdirAll(d, 0700)
	return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"),
		SyncServer: env.base, SyncToken: env.tokens[user], SyncPassword: password, DeviceID: "dev-" + user + "-" + name, DeviceName: name}}
}

func (env *secEnv) raw(user, method, path, device, body string, hdr map[string]string) (int, string) {
	req, _ := http.NewRequest(method, syncURL(env.base, path), strings.NewReader(body))
	if user != "" {
		req.Header.Set("Authorization", "Bearer "+env.tokens[user])
	}
	req.Header.Set("Content-Type", "application/json")
	if device != "" {
		req.Header.Set("X-Device-ID", device)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, e := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if e != nil {
		env.t.Fatal(e)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func secProfile(t *testing.T, s *Store, name, marker string) Profile {
	p := s.PutProfile(Profile{Name: name, Cookies: `[{"name":"sid","value":"` + marker + `"}]`, Fingerprint: GenerateFingerprint("windows", "141")})
	_ = os.MkdirAll(s.ProfileDir(p.ID), 0700)
	if e := os.WriteFile(filepath.Join(s.ProfileDir(p.ID), "marker.txt"), []byte(marker), 0600); e != nil {
		t.Fatal(e)
	}
	return *s.mustProfile(t, p.ID)
}

func (s *Store) mustProfile(t *testing.T, id string) *Profile {
	p, ok := s.GetProfile(id)
	if !ok {
		t.Fatalf("profile %s missing", id)
	}
	return p
}

func readMarker(s *Store, id string) string {
	b, _ := os.ReadFile(filepath.Join(s.ProfileDir(id), "marker.txt"))
	return string(b)
}

func flushCaches() {
	infoCache.Lock()
	infoCache.at = time.Time{}
	infoCache.Unlock()
	envCache.Lock()
	envCache.at = time.Time{}
	envCache.Unlock()
}

// ------------------------------------------------------------------ devices

func TestSecDevices(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "pw-12345678")
	b := env.dev("owner", "B", "pw-12345678")
	if e := a.TestSync(); e != nil {
		t.Fatal(e)
	}
	if l := a.ListDevices(); !l.Supported || l.SelfStatus != "approved" || len(l.Devices) != 1 || !l.Devices[0].Current {
		t.Fatalf("first device should be auto-approved: %+v", l)
	}
	if e := a.SetRequireApproval(true); e != nil {
		t.Fatal(e)
	}
	if e := b.TestSync(); e != nil { // ping stays open for a pending device
		t.Fatal(e)
	}
	if _, e := b.ListCloud(); !isSecForbidden(e) || !strings.Contains(e.Error(), "等待批准") {
		t.Fatalf("pending device must get 403: %v", e)
	}
	if l := b.ListDevices(); l.SelfStatus != "pending" || len(l.Devices) != 0 {
		t.Fatalf("pending device sees only its status: %+v", l)
	}
	l := a.ListDevices()
	if len(l.Devices) != 2 || !l.RequireApproval {
		t.Fatalf("registry: %+v", l)
	}
	for _, d := range l.Devices {
		if d.ID == b.Settings.DeviceID && (d.Status != "pending" || d.Name != "B" || d.OS == "") {
			t.Fatalf("B entry: %+v", d)
		}
	}
	if e := a.ApproveDevice(b.Settings.DeviceID); e != nil {
		t.Fatal(e)
	}
	if _, e := b.ListCloud(); e != nil {
		t.Fatalf("approved device: %v", e)
	}
	p := secProfile(t, b, "B 的环境", "b-data")
	if e := b.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	if e := a.RevokeDevice(b.Settings.DeviceID); e != nil {
		t.Fatal(e)
	}
	if _, e := b.ListCloud(); !isSecForbidden(e) || !strings.Contains(e.Error(), "吊销") {
		t.Fatalf("revoked device must get 403: %v", e)
	}
	if e := b.PushProfile(p.ID); e == nil || !strings.Contains(e.Error(), "吊销") {
		t.Fatalf("revoked push: %v", e)
	}
	if e := b.PullProfile(p.ID); e == nil {
		t.Fatal("revoked pull must fail")
	}
	if code, _ := env.raw("owner", "GET", "/api/profiles", b.Settings.DeviceID, "", nil); code != 403 {
		t.Fatalf("raw revoked: %d", code)
	}
	if code, _ := env.raw("owner", "GET", "/api/profiles", "", "", nil); code != 403 {
		t.Fatalf("no device id with approval required: %d", code)
	}
	if e := a.RevokeDevice(a.Settings.DeviceID); e == nil || !strings.Contains(e.Error(), "当前") {
		t.Fatalf("self revoke must be refused: %v", e)
	}
	if l := b.ListDevices(); l.SelfStatus != "revoked" {
		t.Fatalf("revoked self status: %+v", l)
	}
	// re-approve restores access; other user's registry is separate
	if e := a.ApproveDevice(b.Settings.DeviceID); e != nil {
		t.Fatal(e)
	}
	if _, e := b.ListCloud(); e != nil {
		t.Fatal(e)
	}
	bob := env.dev("bob", "X", "pw-bob-123456")
	if l := bob.ListDevices(); l.SelfStatus != "approved" || len(l.Devices) != 1 {
		t.Fatalf("bob registry should be separate: %+v", l)
	}
}

// ------------------------------------------------------------------ recovery

func TestSecRecoveryNewAccount(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "first-password")
	if st := a.RecoveryStatus(); !st.Supported || st.Configured {
		t.Fatalf("status before: %+v", st)
	}
	rk, e := a.SetupRecovery()
	if e != nil {
		t.Fatal(e)
	}
	if len(rk) != 39 || strings.Count(rk, "-") != 7 {
		t.Fatalf("recovery key format: %q", rk)
	}
	if st := a.RecoveryStatus(); !st.Configured || !st.Unlocked || st.Legacy || !st.ShareReady {
		t.Fatalf("status after: %+v", st)
	}
	if _, e := a.SetupRecovery(); e == nil {
		t.Fatal("second setup must be refused")
	}
	p := secProfile(t, a, "环境一", "v1")
	if e := a.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	// data key is the random master, not the password
	mk := a.loadKeyring().master()
	if string(mk) == "first-password" || len(mk) != 32 {
		t.Fatal("new account must use a random master key")
	}
	// new device, same password: unlocks itself
	b := env.dev("owner", "B", "first-password")
	if e := b.PullProfile(p.ID); e != nil || readMarker(b, p.ID) != "v1" {
		t.Fatalf("unlock with password: %v", e)
	}
	// new device with the recovery key (password forgotten)
	c := env.dev("owner", "C", "")
	if e := c.UnlockWithRecoveryKey("AAAA-BBBB-CCCC-DDDD-EEEE-FFFF-GGGG-HHHH", "second-password"); e == nil || !strings.Contains(e.Error(), "恢复密钥不正确") {
		t.Fatalf("wrong recovery key: %v", e)
	}
	if e := c.UnlockWithRecoveryKey(strings.ToLower(rk), "second-password"); e != nil {
		t.Fatal(e)
	}
	if c.GetSettings().SyncPassword != "second-password" {
		t.Fatal("recovery unlock must set the new password")
	}
	if e := c.PullProfile(p.ID); e != nil || readMarker(c, p.ID) != "v1" {
		t.Fatalf("pull after recovery: %v", e)
	}
	// A keeps working (its master key did not change) and pushes v2
	_ = os.WriteFile(filepath.Join(a.ProfileDir(p.ID), "marker.txt"), []byte("v2"), 0600)
	pp := a.mustProfile(t, p.ID)
	pp.LastRun = time.Now().Unix() + 5
	a.PutProfile(*pp)
	if e := a.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	if e := c.PullProfile(p.ID); e != nil || readMarker(c, p.ID) != "v2" {
		t.Fatalf("pull v2: %v", e)
	}
	// password change: old password stops unlocking new devices, data stays readable
	if e := c.ChangeSyncPassword("wrong", "third-password"); e == nil {
		t.Fatal("wrong old password must be refused")
	}
	if e := c.ChangeSyncPassword("second-password", "third-password"); e != nil {
		t.Fatal(e)
	}
	flushCaches()
	d := env.dev("owner", "D", "second-password")
	if e := d.PullProfile(p.ID); e == nil || !strings.Contains(e.Error(), "无法解锁") {
		t.Fatalf("old password must not unlock: %v", e)
	}
	if e := d.UnlockWithPassword("third-password"); e != nil {
		t.Fatal(e)
	}
	if e := d.PullProfile(p.ID); e != nil || readMarker(d, p.ID) != "v2" {
		t.Fatalf("pull after password change: %v", e)
	}
	// rotate recovery key
	rk2, e := d.RotateRecoveryKey("third-password")
	if e != nil || rk2 == rk {
		t.Fatalf("rotate: %v", e)
	}
	e1 := env.dev("owner", "E1", "").UnlockWithRecoveryKey(rk, "fourth-password")
	if e1 == nil {
		t.Fatal("old recovery key must stop working")
	}
	if e := env.dev("owner", "E2", "").UnlockWithRecoveryKey(rk2, "fourth-password"); e != nil {
		t.Fatal(e)
	}
	// server never sees passwords or the master key
	secAssertNoSecrets(t, env.server, []string{"first-password", "second-password", "third-password", "fourth-password",
		b64.EncodeToString(mk), strings.ReplaceAll(rk, "-", ""), strings.ReplaceAll(rk2, "-", "")})
}

func secAssertNoSecrets(t *testing.T, dir string, secrets []string) {
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		b, _ := os.ReadFile(p)
		for _, s := range secrets {
			if s != "" && strings.Contains(string(b), s) {
				t.Errorf("server file %s contains a secret", p)
			}
		}
		return nil
	})
}

func TestSecRecoveryLegacyMigration(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "legacy-password")
	p := secProfile(t, a, "老环境", "legacy-v1")
	if e := a.PushProfile(p.ID); e != nil { // pre-master-key upload: password-derived key
		t.Fatal(e)
	}
	if e := a.PushGlobal(); e != nil {
		t.Fatal(e)
	}
	flushCaches()
	w := env.dev("owner", "W", "typo-password")
	if _, e := w.SetupRecovery(); e == nil || !strings.Contains(e.Error(), "无法解密") {
		t.Fatalf("a wrong password must not become the master key: %v", e)
	}
	rk, e := a.SetupRecovery()
	if e != nil {
		t.Fatal(e)
	}
	if st := a.RecoveryStatus(); !st.Legacy || !st.Unlocked {
		t.Fatalf("legacy status: %+v", st)
	}
	// existing cloud data is still readable on a new device
	b := env.dev("owner", "B", "legacy-password")
	if e := b.PullProfile(p.ID); e != nil || readMarker(b, p.ID) != "legacy-v1" {
		t.Fatalf("legacy pull: %v", e)
	}
	// the pre-migration global record decrypts with the unlocked master key
	// (PullGlobal itself writes the process-wide fingerprint library, so check the blob directly)
	resp, e := b.sync().req("GET", "/api/profiles/"+globalID+"/blob", nil, "")
	if e != nil {
		t.Fatal(e)
	}
	enc, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if _, e := decrypt(enc, b.loadKeyring().master()); e != nil {
		t.Fatalf("legacy global record: %v", e)
	}
	// change the password: data uploaded before migration stays readable
	if e := a.ChangeSyncPassword("legacy-password", "new-password-1"); e != nil {
		t.Fatal(e)
	}
	flushCaches()
	c := env.dev("owner", "C", "new-password-1")
	if e := c.PullProfile(p.ID); e != nil || readMarker(c, p.ID) != "legacy-v1" {
		t.Fatalf("pull pre-migration data with the new password: %v", e)
	}
	// and the recovery key alone also opens it
	r := env.dev("owner", "R", "")
	if e := r.UnlockWithRecoveryKey(rk, "new-password-2"); e != nil {
		t.Fatal(e)
	}
	if e := r.PullProfile(p.ID); e != nil || readMarker(r, p.ID) != "legacy-v1" {
		t.Fatalf("recovery-key pull of legacy data: %v", e)
	}
	// uploads after migration are readable everywhere
	_ = os.WriteFile(filepath.Join(c.ProfileDir(p.ID), "marker.txt"), []byte("legacy-v2"), 0600)
	pp := c.mustProfile(t, p.ID)
	pp.LastRun = time.Now().Unix() + 5
	c.PutProfile(*pp)
	if e := c.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	if e := r.PullProfile(p.ID); e != nil || readMarker(r, p.ID) != "legacy-v2" {
		t.Fatalf("post-migration pull: %v", e)
	}
}

// ------------------------------------------------------------------ sharing

func TestSecSharing(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "owner-password")
	b := env.dev("bob", "B", "bob-password")
	c := env.dev("carol", "C", "carol-password")
	if l := a.ListShares(); !l.Supported || l.Ready {
		t.Fatalf("not ready before recovery setup: %+v", l)
	}
	if _, e := b.SetupRecovery(); e != nil {
		t.Fatal(e)
	}
	p1 := secProfile(t, a, "共享环境一", "session-secret-1")
	p2 := secProfile(t, a, "私有环境二", "private-2")
	if e := a.PushProfile(p2.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := a.CreateShare(ShareRequest{ProfileID: p1.ID, To: "bob", Mode: "session", Days: 7}); e == nil || !strings.Contains(e.Error(), "恢复密钥") {
		t.Fatalf("sharing needs the identity: %v", e)
	}
	if _, e := a.SetupRecovery(); e != nil {
		t.Fatal(e)
	}
	if _, e := a.CreateShare(ShareRequest{ProfileID: p1.ID, To: "carol", Mode: "session", Days: 7}); e == nil {
		t.Fatal("carol has not published a key yet")
	}
	if _, e := a.CreateShare(ShareRequest{ProfileID: p1.ID, To: "bob", Mode: "session", Days: 400}); e == nil {
		t.Fatal("expiry > 366 days must be refused")
	}
	users, e := a.ShareUsers()
	if e != nil || len(users) != 1 || users[0] != "bob" {
		t.Fatalf("directory: %v %v", users, e)
	}
	s1, e := a.CreateShare(ShareRequest{ProfileID: p1.ID, To: "bob", Mode: "session", Days: 7, Note: "给你用一周"})
	if e != nil {
		t.Fatal(e)
	}
	s2, e := a.CreateShare(ShareRequest{ProfileID: p1.ID, To: "bob", Mode: "config", Days: 1})
	if e != nil {
		t.Fatal(e)
	}
	l := b.ListShares()
	if l.Error != "" || len(l.In) != 2 || l.In[0].Status != "pending" || l.In[0].Wrap != nil {
		t.Fatalf("bob incoming: %+v", l)
	}
	if lo := a.ListShares(); len(lo.Out) != 2 || (lo.Out[0].Note+lo.Out[1].Note) != "给你用一周" {
		t.Fatalf("owner outgoing: %+v", lo)
	}
	// accept the session share: launch-able profile with the browser data
	got, e := b.AcceptShare(s1.ID)
	if e != nil {
		t.Fatal(e)
	}
	if got.ID != "sh-"+s1.ID || got.ShareID != s1.ID || got.SharedFrom != "owner" || got.ShareMode != "session" {
		t.Fatalf("shared profile: %+v", got)
	}
	if readMarker(b, got.ID) != "session-secret-1" || got.Fingerprint.UserAgent != p1.Fingerprint.UserAgent || !strings.Contains(got.Cookies, "session-secret-1") {
		t.Fatal("session share must carry the browser data and the fingerprint")
	}
	if e := b.PushProfile(got.ID); e == nil {
		t.Fatal("a shared profile must never be pushed to the recipient's cloud")
	}
	// config-only share
	g2, e := b.AcceptShare(s2.ID)
	if e != nil {
		t.Fatal(e)
	}
	if readMarker(b, g2.ID) != "" || g2.Cookies != "" || g2.Name != p1.Name {
		t.Fatal("config share must not carry session data")
	}
	// bob cannot read other owner profiles or other shares
	if cl, _ := b.ListCloud(); len(cl) != 0 {
		t.Fatalf("bob sees owner's cloud: %+v", cl)
	}
	if code, _ := env.raw("bob", "GET", "/api/profiles/"+p2.ID+"/blob", "dev-bob-B", "", nil); code == 200 {
		t.Fatal("bob read an owner profile blob")
	}
	if code, _ := env.raw("carol", "GET", "/api/shares/"+s1.ID+"/blob", "dev-carol-C", "", nil); code != 404 {
		t.Fatalf("carol read bob's share: %d", code)
	}
	if code, _ := env.raw("carol", "POST", "/api/shares/"+s1.ID+"/revoke", "dev-carol-C", "{}", nil); code != 404 {
		t.Fatalf("carol revoked someone else's share: %d", code)
	}
	if code, _ := env.raw("bob", "POST", "/api/shares/"+s1.ID+"/extend", "dev-bob-B", `{"days":30}`, nil); code != 404 {
		t.Fatalf("recipient extended: %d", code)
	}
	_ = c
	// extend, then revoke -> bob's copy is removed on the next share sync
	if e := a.ExtendShare(s1.ID, 30); e != nil {
		t.Fatal(e)
	}
	if n, e := b.SyncShares(); e != nil || n != 0 {
		t.Fatalf("sync after extend: %d %v", n, e)
	}
	if p := b.mustProfile(t, got.ID); p.ShareExpires < time.Now().Unix()+29*86400 {
		t.Fatalf("extension not applied locally: %d", p.ShareExpires)
	}
	if e := a.RevokeShare(s1.ID); e != nil {
		t.Fatal(e)
	}
	if n, e := b.SyncShares(); e != nil || n != 1 {
		t.Fatalf("revoke sync: %d %v", n, e)
	}
	if _, ok := b.GetProfile(got.ID); ok || !dirGone(b, got.ID) {
		t.Fatal("revoked share must be removed locally")
	}
	if e := b.shareLaunchGuard(g2.ID); e != nil {
		t.Fatalf("live share must launch: %v", e)
	}
	// expiry: push the server's expiry into the past
	shf := filepath.Join(env.server, "_shared", "shares.json")
	var all map[string]map[string]any
	raw, _ := os.ReadFile(shf)
	_ = json.Unmarshal(raw, &all)
	all[s2.ID]["expires_at"] = time.Now().Unix() - 10
	raw, _ = json.Marshal(all)
	_ = os.WriteFile(shf, raw, 0600)
	if e := b.shareLaunchGuard(g2.ID); e == nil || !strings.Contains(e.Error(), "已结束") {
		t.Fatalf("expired share must not launch: %v", e)
	}
	if _, ok := b.GetProfile(g2.ID); ok {
		t.Fatal("expired share must be removed locally")
	}
	if code, _ := env.raw("bob", "GET", "/api/shares/"+s2.ID+"/blob", "dev-bob-B", "", nil); code != 410 {
		t.Fatalf("expired blob: %d", code)
	}
	if e := a.ExtendShare(s2.ID, 7); e == nil {
		t.Fatal("an expired share cannot be extended")
	}
	// leave / decline
	s3, _ := a.CreateShare(ShareRequest{ProfileID: p2.ID, To: "bob", Mode: "config", Days: 30})
	s4, _ := a.CreateShare(ShareRequest{ProfileID: p2.ID, To: "bob", Mode: "config", Days: 30})
	g3, e := b.AcceptShare(s3.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e := b.LeaveShare(s3.ID); e != nil {
		t.Fatal(e)
	}
	if _, ok := b.GetProfile(g3.ID); ok {
		t.Fatal("leave must remove the local copy")
	}
	if e := b.DeclineShare(s4.ID); e != nil {
		t.Fatal(e)
	}
	st := map[string]string{}
	for _, x := range a.ListShares().Out {
		st[x.ID] = x.Status
	}
	if st[s1.ID] != "revoked" || st[s2.ID] != "expired" || st[s3.ID] != "left" || st[s4.ID] != "declined" {
		t.Fatalf("owner statuses: %v", st)
	}
	// blobs of ended shares are gone; nothing in the share store is plaintext
	if fs, _ := filepath.Glob(filepath.Join(env.server, "_shared", "shares", "*.gbe")); len(fs) != 0 {
		t.Fatalf("ended share blobs kept: %v", fs)
	}
	secAssertNoSecrets(t, filepath.Join(env.server, "_shared"), []string{"session-secret-1", "private-2", "owner-password", "bob-password"})
}

// ------------------------------------------------------------------ admin

func TestSecAdminPage(t *testing.T) {
	env := startSecServer(t)
	a := env.dev("owner", "A", "owner-password")
	p := secProfile(t, a, "非常机密的环境名", "admin-secret-data")
	if e := a.PushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_, _ = a.SetupRecovery()
	_, _ = env.dev("bob", "B", "bob-password").ListCloud()
	env.raw("bob", "PUT", "/api/keys", "dev-bob-B", `{"envelope":{"v":1},"epoch":5}`, nil) // 409, shows under 最近错误
	env.raw("owner", "POST", "/api/shares", "dev-owner-A", `{"to":"bob","days":400}`, nil)
	code, body := env.raw("", "GET", "/admin", "", "", nil)
	if code != 200 || !strings.Contains(body, "管理令牌") || strings.Contains(body, "bob") {
		t.Fatalf("login page: %d", code)
	}
	code, body = env.raw("", "POST", "/admin", "", "", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if code != 401 || strings.Contains(body, "bob") {
		t.Fatalf("empty token: %d", code)
	}
	req, _ := http.NewRequest("POST", syncURL(env.base, "/admin"), strings.NewReader(url.Values{"token": {"wrong"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("wrong token: %d", resp.StatusCode)
	}
	code, body = env.raw("", "GET", "/admin", "", "", map[string]string{"X-Admin-Token": env.admin})
	if code != 200 || !strings.Contains(body, "bob") || !strings.Contains(body, "carol") || !strings.Contains(body, "最近错误") {
		t.Fatalf("admin page: %d %s", code, body)
	}
	for _, s := range []string{"非常机密的环境名", "admin-secret-data", "owner-password", env.tokens["owner"], env.tokens["bob"], env.admin, p.ID} {
		if strings.Contains(body, s) {
			t.Fatalf("admin page leaks %q", s)
		}
	}
	// a real form login sets a session cookie
	req, _ = http.NewRequest("POST", syncURL(env.base, "/admin"), strings.NewReader(url.Values{"token": {env.admin}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, e := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if e != nil {
		t.Fatal(e)
	}
	resp.Body.Close()
	if resp.StatusCode != 303 || len(resp.Cookies()) == 0 {
		t.Fatalf("form login: %d", resp.StatusCode)
	}
	if os.Getenv("GHOST_ADMIN_HTML") != "" {
		_ = os.WriteFile(os.Getenv("GHOST_ADMIN_HTML"), []byte(body), 0644)
	}
}
