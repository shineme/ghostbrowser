package main

// Device management on the sync server: every device registers itself with
// its X-Device-ID on each request; the server can require approval for new
// devices and refuses revoked ones with HTTP 403.

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// errSecForbidden: HTTP 403 from the device gate (revoked / waiting for approval).
type errSecForbidden struct{ msg string }

func (e errSecForbidden) Error() string { return e.msg }

func isSecForbidden(e error) bool { var x errSecForbidden; return errors.As(e, &x) }

// secStatusErr turns the server's Chinese 403/404/409/410/429 messages of the
// account & security routes into plain errors.
func secStatusErr(code int, body []byte) error {
	if code != 403 && code != 404 && code != 409 && code != 410 && code != 429 {
		return nil
	}
	var v struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &v)
	msg := strings.TrimSpace(v.Error)
	if code == 403 {
		if msg == "" {
			msg = "同步服务器拒绝了此设备（HTTP 403）"
		}
		return errSecForbidden{msg}
	}
	if hasCJK(msg) {
		return errors.New(msg)
	}
	return nil
}

type CloudDevice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	FirstSeen int64  `json:"firstSeen"`
	LastSeen  int64  `json:"lastSeen"`
	Approved  bool   `json:"approved"`
	Revoked   bool   `json:"revoked"`
	Status    string `json:"status"` // approved | pending | revoked
	Current   bool   `json:"current"`
}

type DeviceList struct {
	Supported       bool          `json:"supported"`
	Self            string        `json:"self"`
	SelfStatus      string        `json:"selfStatus"` // approved | pending | revoked | unknown
	RequireApproval bool          `json:"requireApproval"`
	Devices         []CloudDevice `json:"devices"`
	Error           string        `json:"error,omitempty"`
}

// ListDevices returns the account's device registry. A pending or revoked
// device only learns its own status.
func (s *Store) ListDevices() DeviceList {
	out := DeviceList{Devices: []CloudDevice{}, Self: s.GetSettings().DeviceID}
	if e := syncReady(s.GetSettings(), false); e != nil {
		out.Error = e.Error()
		return out
	}
	si, e := s.ServerInfo(false)
	if e != nil {
		out.Error = e.Error()
		if isSecForbidden(e) {
			out.Supported, out.SelfStatus = true, "revoked"
		}
		return out
	}
	if out.Supported = si.has("devices"); !out.Supported {
		return out
	}
	var self struct {
		Status          string `json:"status"`
		RequireApproval bool   `json:"requireApproval"`
	}
	c := s.sync()
	if e = c.jsonRequest("GET", "/api/devices/self", nil, &self); e != nil {
		out.Error = e.Error()
		if isSecForbidden(e) {
			out.SelfStatus = "revoked"
		}
		return out
	}
	out.SelfStatus, out.RequireApproval = self.Status, self.RequireApproval
	if self.Status != "approved" {
		return out
	}
	var list struct {
		RequireApproval bool          `json:"requireApproval"`
		Devices         []CloudDevice `json:"devices"`
	}
	if e = c.jsonRequest("GET", "/api/devices", nil, &list); e != nil {
		out.Error = e.Error()
		return out
	}
	out.RequireApproval = list.RequireApproval
	for _, d := range list.Devices {
		d.Current = d.ID == out.Self
		out.Devices = append(out.Devices, d)
	}
	return out
}

func (s *Store) deviceAction(id, act string) error {
	if !syncIDPatternDevice(id) {
		return errors.New("invalid device ID")
	}
	return s.sync().jsonRequest("POST", "/api/devices/"+url.PathEscape(id)+"/"+act, map[string]any{}, nil)
}

func syncIDPatternDevice(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

func (s *Store) ApproveDevice(id string) error { return s.deviceAction(id, "approve") }
func (s *Store) RevokeDevice(id string) error  { return s.deviceAction(id, "revoke") }
func (s *Store) ForgetDevice(id string) error  { return s.deviceAction(id, "forget") }

func (s *Store) SetRequireApproval(on bool) error {
	return s.sync().jsonRequest("POST", "/api/devices/settings", map[string]any{"requireApproval": on}, nil)
}
