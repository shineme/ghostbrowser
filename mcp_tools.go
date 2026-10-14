package main

import (
	"fmt"
	"math"
	"strings"
)

// MCP tools. Every tool is backed by existing Store / App methods; none of
// them deletes anything.

type mcpTool struct {
	Name        string
	Title       string
	Description string
	Props       map[string]any // JSON-schema properties
	Required    []string
	ReadOnly    bool
	Idempotent  bool
	Run         func(c *mcpCore, args map[string]any) (any, error)
}

func (t *mcpTool) describe() map[string]any {
	props := t.Props
	if props == nil {
		props = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(t.Required) > 0 {
		schema["required"] = t.Required
	}
	return map[string]any{
		"name":        t.Name,
		"title":       t.Title,
		"description": t.Description,
		"inputSchema": schema,
		"annotations": map[string]any{
			"title":           t.Title,
			"readOnlyHint":    t.ReadOnly,
			"destructiveHint": false,
			"idempotentHint":  t.Idempotent,
			"openWorldHint":   false,
		},
	}
}

func findMcpTool(name string) *mcpTool {
	for i := range mcpTools {
		if mcpTools[i].Name == name {
			return &mcpTools[i]
		}
	}
	return nil
}

// ---- argument helpers

func mArgStr(a map[string]any, k string) string {
	if v, ok := a[k]; ok && v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

func mArgInt(a map[string]any, k string, def int) (int, error) {
	v, ok := a[k]
	if !ok || v == nil {
		return def, nil
	}
	switch n := v.(type) {
	case float64:
		if n != math.Trunc(n) {
			return 0, fmt.Errorf("%s must be an integer", k)
		}
		return int(n), nil
	case int:
		return n, nil
	}
	return 0, fmt.Errorf("%s must be an integer", k)
}

func mArgStrs(a map[string]any, k string) ([]string, error) {
	v, ok := a[k]
	if !ok || v == nil {
		return nil, nil
	}
	switch x := v.(type) {
	case string:
		if strings.TrimSpace(x) == "" {
			return nil, nil
		}
		return []string{strings.TrimSpace(x)}, nil
	case []any:
		out := []string{}
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", k)
			}
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s must be an array of strings", k)
}

func schemaStr(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func jsStrArr(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}
func jsInt(desc string, min, max int) map[string]any {
	return map[string]any{"type": "integer", "description": desc, "minimum": min, "maximum": max}
}
func jsEnum(desc string, vals ...string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": vals}
}

// ---- views (never include proxy passwords)

type mcpEnv struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Group      string   `json:"group"`
	Tags       []string `json:"tags"`
	Notes      string   `json:"notes,omitempty"`
	OS         string   `json:"os"`
	DeviceType string   `json:"deviceType"`
	Kernel     string   `json:"kernel"` // custom | chromium
	ProxyID    string   `json:"proxyId"`
	ProxyName  string   `json:"proxyName"`
	Status     string   `json:"status"` // running | stopped
	Storage    string   `json:"storage"`
	CreatedAt  int64    `json:"createdAt"`
	LastRun    int64    `json:"lastRun"`
}

func (c *mcpCore) envView(p Profile) mcpEnv {
	e := mcpEnv{ID: p.ID, Name: p.Name, Group: p.Group, Tags: p.Tags, Notes: p.Notes, OS: p.Fingerprint.OS,
		DeviceType: p.Fingerprint.DeviceType, Kernel: "custom", ProxyID: p.ProxyID, Status: "stopped",
		Storage: p.Storage, CreatedAt: p.CreatedAt, LastRun: p.LastRun}
	if e.Tags == nil {
		e.Tags = []string{}
	}
	if e.DeviceType == "" {
		e.DeviceType = "desktop"
	}
	if p.useChromium() {
		e.Kernel = "chromium"
	}
	if p.InlineProxy != nil && p.InlineProxy.Host != "" {
		e.ProxyName = fmt.Sprintf("inline %s:%d", p.InlineProxy.Host, p.InlineProxy.Port)
	} else if p.ProxyID != "" {
		if px, ok := c.store.GetProxy(p.ProxyID); ok {
			e.ProxyName = px.Name
		}
	}
	if _, ok := runReg.get(p.ID); ok {
		e.Status = "running"
	}
	return e
}

type mcpProxy struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	HasAuth   bool   `json:"hasAuth"`
	Tag       string `json:"tag"`
	Status    string `json:"status"`
	IP        string `json:"ip"`
	Country   string `json:"country"`
	City      string `json:"city"`
	Timezone  string `json:"timezone"`
	IPType    string `json:"ipType"`
	LatencyMs int64  `json:"latencyMs"`
	CheckedAt int64  `json:"checkedAt"`
	UsedBy    int    `json:"usedBy"`
}

func (c *mcpCore) requireProfile(id string) (*Profile, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	p, ok := c.store.GetProfile(id)
	if !ok {
		// allow an exact (unique) name as well, models often pass names
		var hit *Profile
		for _, q := range c.store.ListProfiles() {
			if q.Name == id {
				if hit != nil {
					return nil, fmt.Errorf("more than one environment is named %q; pass the id", id)
				}
				qq := q
				hit = &qq
			}
		}
		if hit == nil {
			return nil, fmt.Errorf("environment %q not found", id)
		}
		return hit, nil
	}
	return p, nil
}

var mcpTools = []mcpTool{
	{
		Name: "list_environments", Title: "List environments", ReadOnly: true, Idempotent: true,
		Description: "List browser environments (profiles) with their group, tags, OS, device type, kernel, proxy and running status. Optional filters.",
		Props: map[string]any{
			"group":  schemaStr("Only this group (exact name); omit to list all."),
			"tag":    schemaStr("Only environments carrying this tag."),
			"query":  schemaStr("Case-insensitive substring of the name or notes."),
			"status": jsEnum("Only running or stopped environments.", "running", "stopped"),
			"limit":  jsInt("Maximum rows to return (default 200).", 1, 1000),
		},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			limit, err := mArgInt(a, "limit", 200)
			if err != nil {
				return nil, err
			}
			group, tag, q, status := mArgStr(a, "group"), mArgStr(a, "tag"), strings.ToLower(mArgStr(a, "query")), mArgStr(a, "status")
			out := []mcpEnv{}
			total := 0
			for _, p := range c.store.ListProfiles() {
				if group != "" && p.Group != group {
					continue
				}
				if tag != "" {
					has := false
					for _, t := range p.Tags {
						has = has || t == tag
					}
					if !has {
						continue
					}
				}
				if q != "" && !strings.Contains(strings.ToLower(p.Name+"\x00"+p.Notes), q) {
					continue
				}
				e := c.envView(p)
				if status != "" && e.Status != status {
					continue
				}
				total++
				if len(out) < limit {
					out = append(out, e)
				}
			}
			return map[string]any{"environments": out, "total": total}, nil
		},
	},
	{
		Name: "create_environment", Title: "Create environments",
		Description: "Create one or more new environments with freshly generated fingerprints. With count > 1 they are named name-1, name-2, ...",
		Props: map[string]any{
			"name":       schemaStr("Environment name (or prefix when count > 1)."),
			"count":      jsInt("How many to create (default 1, max 50).", 1, 50),
			"deviceType": jsEnum("Device type (default desktop).", "desktop", "laptop", "phone", "tablet", "foldable", "chromebook", "tv"),
			"os":         jsEnum("Operating system (default windows for desktop/laptop, android for phones/tablets).", "windows", "macos", "linux", "android", "chromeos"),
			"group":      schemaStr("Group to put them in (created implicitly)."),
			"proxyId":    schemaStr("Saved proxy ID to assign (see list_proxies)."),
			"tags":       jsStrArr("Tags to add."),
			"notes":      schemaStr("Notes."),
		},
		Required: []string{"name"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			name := mArgStr(a, "name")
			if name == "" {
				return nil, fmt.Errorf("name is required")
			}
			count, err := mArgInt(a, "count", 1)
			if err != nil {
				return nil, err
			}
			if count < 1 || count > 50 {
				return nil, fmt.Errorf("count must be between 1 and 50")
			}
			tags, err := mArgStrs(a, "tags")
			if err != nil {
				return nil, err
			}
			proxyID := mArgStr(a, "proxyId")
			if proxyID != "" {
				if _, ok := c.store.GetProxy(proxyID); !ok {
					return nil, fmt.Errorf("proxy %q not found", proxyID)
				}
			}
			group := mArgStr(a, "group")
			if group != "" {
				if group, err = validGroupName(group); err != nil {
					return nil, err
				}
			}
			dev, osName := mArgStr(a, "deviceType"), mArgStr(a, "os")
			app := &App{store: c.store}
			created := app.CreateTypedProfiles(name, dev, osName, count, proxyID, group)
			out := []mcpEnv{}
			notes := mArgStr(a, "notes")
			for _, p := range created {
				if len(tags) > 0 || notes != "" {
					p.Tags = append(p.Tags, tags...)
					p.Notes = notes
					p = c.store.PutProfile(p)
				}
				out = append(out, c.envView(p))
			}
			c.store.auditNamed("mcp", "profile.create", mFirstID(created), name, nil, fmt.Sprintf("创建 %d 个环境", len(created)))
			return map[string]any{"created": out}, nil
		},
	},
	{
		Name: "start_environment", Title: "Start environment", Idempotent: true,
		Description: "Launch an environment's browser. Returns the CDP debug port and websocket URL for Puppeteer / Playwright connectOverCDP. Starting a running environment returns its existing session.",
		Props:       map[string]any{"id": schemaStr("Environment ID (or exact unique name).")},
		Required:    []string{"id"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			p, err := c.requireProfile(mArgStr(a, "id"))
			if err != nil {
				return nil, err
			}
			if rs, ok := runReg.get(p.ID); ok {
				return map[string]any{"id": p.ID, "name": p.Name, "status": "running", "alreadyRunning": true, "debugPort": rs.DebugPort, "websocket": rs.WSURL, "pid": rs.PID}, nil
			}
			if err := globalAuto.BeforeLaunch(p.ID); err != nil {
				return nil, err
			}
			rs, err := c.store.Launch(p.ID)
			c.store.auditProfile("mcp", "profile.start", p.ID, err, "")
			if err != nil {
				return nil, err
			}
			return map[string]any{"id": p.ID, "name": p.Name, "status": "running", "debugPort": rs.DebugPort, "websocket": rs.WSURL, "pid": rs.PID}, nil
		},
	},
	{
		Name: "stop_environment", Title: "Stop environment", Idempotent: true,
		Description: "Close a running environment's browser gracefully (cookies and storage are flushed; auto sync uploads afterwards when enabled).",
		Props:       map[string]any{"id": schemaStr("Environment ID (or exact unique name).")},
		Required:    []string{"id"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			p, err := c.requireProfile(mArgStr(a, "id"))
			if err != nil {
				return nil, err
			}
			if _, ok := runReg.get(p.ID); !ok {
				return map[string]any{"id": p.ID, "name": p.Name, "status": "stopped", "wasRunning": false}, nil
			}
			(&App{store: c.store}).StopProfile(p.ID)
			c.store.auditProfile("mcp", "profile.stop", p.ID, nil, "")
			return map[string]any{"id": p.ID, "name": p.Name, "status": "stopping", "wasRunning": true}, nil
		},
	},
	{
		Name: "list_proxies", Title: "List proxies", ReadOnly: true, Idempotent: true,
		Description: "List saved proxies with their last check result (exit IP, country, latency) and how many environments use each. Passwords are never returned.",
		Props: map[string]any{
			"tag":    schemaStr("Only proxies imported with this tag."),
			"status": jsEnum("Only proxies whose last check was ok / failed / never run.", "ok", "fail", "unknown"),
		},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			tag, status := mArgStr(a, "tag"), mArgStr(a, "status")
			used := map[string]int{}
			for _, p := range c.store.ListProfiles() {
				if p.ProxyID != "" {
					used[p.ProxyID]++
				}
			}
			out := []mcpProxy{}
			for _, p := range c.store.ListProxies() {
				if tag != "" && p.Tag != tag {
					continue
				}
				st := p.Status
				if st == "" {
					st = "unknown"
				}
				if status != "" && st != status {
					continue
				}
				out = append(out, mcpProxy{ID: p.ID, Name: p.Name, Type: p.Type, Host: p.Host, Port: p.Port, HasAuth: p.User != "",
					Tag: p.Tag, Status: st, IP: p.IP, Country: p.Country, City: p.City, Timezone: p.Timezone, IPType: p.IPType,
					LatencyMs: p.Latency, CheckedAt: p.Checked, UsedBy: used[p.ID]})
			}
			return map[string]any{"proxies": out, "total": len(out)}, nil
		},
	},
	{
		Name: "assign_proxy", Title: "Assign proxy", Idempotent: true,
		Description: "Assign one saved proxy to one or more environments. Pass proxyId \"\" to switch them to a direct connection.",
		Props: map[string]any{
			"environmentIds": jsStrArr("Environment IDs."),
			"proxyId":        schemaStr("Saved proxy ID from list_proxies, or \"\" for direct connection."),
		},
		Required: []string{"environmentIds", "proxyId"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			ids, err := mArgStrs(a, "environmentIds")
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return nil, fmt.Errorf("environmentIds is required")
			}
			for i, id := range ids {
				p, err := c.requireProfile(id)
				if err != nil {
					return nil, err
				}
				ids[i] = p.ID
			}
			if _, ok := a["proxyId"]; !ok {
				return nil, fmt.Errorf("proxyId is required (use \"\" for direct)")
			}
			proxyID := mArgStr(a, "proxyId")
			var px []string
			pname := "直连"
			if proxyID != "" {
				pr, ok := c.store.GetProxy(proxyID)
				if !ok {
					return nil, fmt.Errorf("proxy %q not found", proxyID)
				}
				px, pname = []string{proxyID}, pr.Name
			}
			n, err := (&App{store: c.store}).AssignProxies(ids, px, "one")
			c.store.auditNamed("mcp", "profile.edit", mFirstStr(ids), pname, err, fmt.Sprintf("分配代理「%s」给 %d 个环境", pname, n))
			if err != nil {
				return nil, err
			}
			return map[string]any{"updated": n, "proxyId": proxyID, "proxyName": pname}, nil
		},
	},
	{
		Name: "list_groups", Title: "List groups", ReadOnly: true, Idempotent: true,
		Description: "List environment groups with how many environments each holds and how many are running.",
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			g := c.store.ListGroups()
			ungrouped := 0
			for _, p := range c.store.ListProfiles() {
				if strings.TrimSpace(p.Group) == "" {
					ungrouped++
				}
			}
			return map[string]any{"groups": g, "ungrouped": ungrouped}, nil
		},
	},
	{
		Name: "create_group", Title: "Create group",
		Description: "Create an empty group. Move environments into it with set_environment_group or create_environment.group.",
		Props:       map[string]any{"name": schemaStr("Group name (max 64 characters).")},
		Required:    []string{"name"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			g, err := c.store.CreateGroup(mArgStr(a, "name"))
			c.store.auditNamed("mcp", "group.create", "", mArgStr(a, "name"), err, "")
			if err != nil {
				return nil, err
			}
			return map[string]any{"group": g}, nil
		},
	},
	{
		Name: "set_environment_group", Title: "Move environments to a group", Idempotent: true,
		Description: "Move environments into a group (\"\" = ungrouped) and optionally add a tag to them.",
		Props: map[string]any{
			"environmentIds": jsStrArr("Environment IDs."),
			"group":          schemaStr("Target group name, \"\" to remove from any group."),
			"addTag":         schemaStr("Optional tag to add."),
		},
		Required: []string{"environmentIds", "group"},
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			ids, err := mArgStrs(a, "environmentIds")
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return nil, fmt.Errorf("environmentIds is required")
			}
			for i, id := range ids {
				p, err := c.requireProfile(id)
				if err != nil {
					return nil, err
				}
				ids[i] = p.ID
			}
			group := mArgStr(a, "group")
			if group != "" {
				if group, err = validGroupName(group); err != nil {
					return nil, err
				}
			}
			n := (&App{store: c.store}).SetProfilesGroup(ids, group, mArgStr(a, "addTag"))
			label := group
			if label == "" {
				label = "未分组"
			}
			c.store.auditNamed("mcp", "profile.edit", mFirstStr(ids), label, nil, fmt.Sprintf("移动 %d 个环境到分组「%s」", n, label))
			return map[string]any{"updated": n, "group": group}, nil
		},
	},
	{
		Name: "list_tags", Title: "List tags", ReadOnly: true, Idempotent: true,
		Description: "List tags used on environments, most used first.",
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			return map[string]any{"tags": c.store.ListTags()}, nil
		},
	},
	{
		Name: "list_kernels", Title: "List browser kernels", ReadOnly: true, Idempotent: true,
		Description: "Show the two browser kernels (custom Chromium with native fingerprints, open-source Chromium via CDP), their paths, whether they were found and their Chrome version.",
		Run: func(c *mcpCore, a map[string]any) (any, error) {
			return map[string]any{"kernels": c.store.ListKernels()}, nil
		},
	},
}

func mFirstID(ps []Profile) string {
	if len(ps) > 0 {
		return ps[0].ID
	}
	return ""
}

func mFirstStr(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
