package main

// Portable document schema versioning. Profile documents exported by
// GhostBrowser (profile JSON, wrapped cookie files) carry a top-level
// "schema" int. A client refuses a document whose schema is newer than it
// understands instead of silently dropping the fields it does not know.

import (
	"encoding/json"
	"fmt"
	"strings"
)

// profileDocSchema is the current portable profile document schema.
// Bump it when an exported document gains fields older clients would lose.
const profileDocSchema = 1

// checkDocSchema reads the top-level "schema" of a JSON object document and
// rejects one newer than this client. Documents without it (older exports,
// third-party files) are schema 1. Non-object documents pass.
func checkDocSchema(raw []byte, kind string) error {
	t := strings.TrimSpace(strings.TrimPrefix(string(raw), "\ufeff"))
	if !strings.HasPrefix(t, "{") {
		return nil
	}
	var head struct {
		Schema json.RawMessage `json:"schema"`
	}
	if json.Unmarshal([]byte(t), &head) != nil || len(head.Schema) == 0 || string(head.Schema) == "null" {
		return nil
	}
	var n float64
	if err := json.Unmarshal(head.Schema, &n); err != nil {
		return fmt.Errorf("%s的 schema 字段无效", kind)
	}
	if int(n) > profileDocSchema {
		return fmt.Errorf("这个%s来自更新版本的 GhostBrowser（文档格式 schema %d，当前版本最高支持 %d），请先升级 GhostBrowser 再导入，以免丢失字段", kind, int(n), profileDocSchema)
	}
	return nil
}

// withDocSchema adds "schema": N as the first key of an exported JSON object.
func withDocSchema(b []byte) []byte {
	t := strings.TrimLeft(string(b), " \t\r\n")
	if !strings.HasPrefix(t, "{") {
		return b
	}
	rest := strings.TrimLeft(t[1:], " \t\r\n")
	if strings.HasPrefix(rest, "}") {
		return []byte(fmt.Sprintf("{\n  \"schema\": %d\n}", profileDocSchema))
	}
	return []byte(fmt.Sprintf("{\n  \"schema\": %d,\n  %s", profileDocSchema, rest))
}
