package tinkalet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/nats-io/nats.go"
)

type Config struct {
	Args    []string
	Stdout  io.Writer
	Stderr  io.Writer
	Env     []string
	Version string
}

type cli struct {
	out     io.Writer
	errOut  io.Writer
	env     map[string]string
	version string
}

type Profile struct {
	Name               string `json:"name"`
	Server             string `json:"server"`
	Shell              string `json:"shell"`
	Role               string `json:"role"`
	Trust              string `json:"trust"`
	Source             string `json:"source"`
	CredentialRef      string `json:"credentialRef"`
	CredentialRedacted bool   `json:"credentialRedacted"`
}

type profilesFile struct {
	Profiles []Profile `json:"profiles"`
}

type listProfile struct {
	Name               string `json:"name"`
	Default            bool   `json:"default"`
	Server             string `json:"server"`
	Shell              string `json:"shell"`
	Role               string `json:"role"`
	Trust              string `json:"trust"`
	Source             string `json:"source"`
	CredentialRef      string `json:"credentialRef"`
	CredentialRedacted bool   `json:"credentialRedacted"`
}

type descriptor struct {
	Kind       string `json:"kind"`
	Server     string `json:"server"`
	Shell      string `json:"shell"`
	Credential string `json:"credential"`
	Role       string `json:"role"`
	Trust      string `json:"trust"`
	Source     string `json:"source"`
}

func Run(cfg Config) int {
	out := cfg.Stdout
	if out == nil {
		out = io.Discard
	}
	errOut := cfg.Stderr
	if errOut == nil {
		errOut = io.Discard
	}
	env := cfg.Env
	if env == nil {
		env = os.Environ()
	}
	version := cfg.Version
	if version == "" {
		version = "tinkalet dev"
	}
	c := cli{out: out, errOut: errOut, env: envMap(env), version: version}
	return c.run(cfg.Args)
}

func (c cli) run(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "--help", "-h", "help":
		c.help()
		return 0
	case "--version", "version":
		fmt.Fprintln(c.out, c.version)
		return 0
	case "profile":
		return c.profile(args[1:])
	case "trigger":
		return c.trigger(args[1:])
	case "item":
		return c.item(args[1:])
	default:
		return c.usage()
	}
}

func (c cli) help() {
	fmt.Fprint(c.out, `usage: tinkalet <command> [options]

commands:
  profile import local --store <dir> --name <name>
  profile list [--json]
  profile use <name>
  trigger <intent> [--profile <name>] [--request-id <id>] [--json]
  item create <key> [--status pending] [--value <json>] [--json]
  item get <key> [--json]
  item resolve <key> [--value <json>] [--revision <n>] [--json]
  item wait <key> --for resolved [--timeout <duration>] [--json]
`)
}

func (c cli) usage() int {
	fmt.Fprintln(c.errOut, "usage: tinkalet <command> [options]")
	return 2
}

func (c cli) profile(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "import":
		if len(args) < 2 || args[1] != "local" {
			return c.usage()
		}
		store, name, ok := parseImport(args[2:])
		if !ok || !validName(name) {
			return c.usage()
		}
		return c.importLocal(store, name)
	case "list":
		jsonOut, ok := parseList(args[1:])
		if !ok {
			return c.usage()
		}
		return c.list(jsonOut)
	case "use":
		if len(args) != 2 || !validName(args[1]) {
			return c.usage()
		}
		return c.use(args[1])
	default:
		return c.usage()
	}
}

func parseImport(args []string) (string, string, bool) {
	var store, name string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--store" && i+1 < len(args):
			i++
			store = args[i]
		case strings.HasPrefix(arg, "--store="):
			store = strings.TrimPrefix(arg, "--store=")
		case arg == "--name" && i+1 < len(args):
			i++
			name = args[i]
		case strings.HasPrefix(arg, "--name="):
			name = strings.TrimPrefix(arg, "--name=")
		default:
			return "", "", false
		}
	}
	return store, name, store != "" && name != ""
}

func parseList(args []string) (bool, bool) {
	if len(args) == 0 {
		return false, true
	}
	if len(args) == 1 && args[0] == "--json" {
		return true, true
	}
	return false, false
}

func (c cli) importLocal(store, name string) int {
	prof, cred, reason := c.profileFromStore(store, name)
	if reason != "" {
		return c.denyProfile(name, "profile import", reason)
	}
	dataDir := c.dataDir()
	dst := filepath.Join(dataDir, "profiles", name, "caller.creds")
	body, err := os.ReadFile(cred)
	if err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	if err := write0600(dst, body); err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	prof.CredentialRef = filepath.ToSlash(filepath.Join("profiles", name, "caller.creds"))
	prof.CredentialRedacted = true
	profiles, err := c.loadProfiles()
	if err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	upsert(&profiles, prof)
	if err := c.saveProfiles(profiles); err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	fmt.Fprintf(c.out, "profile %s imported\n", name)
	return 0
}

func (c cli) profileFromStore(store, name string) (Profile, string, string) {
	store, err := filepath.Abs(store)
	if err != nil {
		return Profile{}, "", "import-source-invalid"
	}
	body, err := os.ReadFile(filepath.Join(store, "local-profile.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Profile{}, "", "import-source-missing"
		}
		return Profile{}, "", "import-source-invalid"
	}
	var desc descriptor
	if err := json.Unmarshal(body, &desc); err != nil {
		return Profile{}, "", "import-source-invalid"
	}
	if desc.Kind != "tinkabot.localProfile.v1" || desc.Trust != "local-owner" || desc.Source != "local-store:"+store {
		return Profile{}, "", "import-source-invalid"
	}
	if desc.Role != "caller" && desc.Role != "edge" {
		return Profile{}, "", "import-source-invalid"
	}
	if !validURL(desc.Server, "nats") || (desc.Shell != "" && !validURL(desc.Shell, "http", "https")) {
		return Profile{}, "", "import-source-invalid"
	}
	cred, ok := sourcePath(store, desc.Credential)
	if !ok {
		return Profile{}, "", "import-source-invalid"
	}
	if _, err := os.Stat(cred); err != nil {
		return Profile{}, "", "import-source-invalid"
	}
	return Profile{
		Name:   name,
		Server: desc.Server,
		Shell:  desc.Shell,
		Role:   desc.Role,
		Trust:  desc.Trust,
		Source: desc.Source,
	}, cred, ""
}

func sourcePath(root, rel string) (string, bool) {
	if rel == "" || filepath.IsAbs(rel) {
		return "", false
	}
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", false
	}
	path := filepath.Join(root, clean)
	back, err := filepath.Rel(root, path)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", false
	}
	return path, true
}

func validURL(raw string, schemes ...string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	for _, scheme := range schemes {
		if u.Scheme == scheme {
			return true
		}
	}
	return false
}

func (c cli) list(jsonOut bool) int {
	profiles, err := c.loadProfiles()
	if err != nil {
		return c.denyProfile("default", "profile list", "profile-not-found")
	}
	def := c.defaultProfile()
	if find(profiles, def) == nil {
		def = ""
	}
	if jsonOut {
		doc := struct {
			Default  string        `json:"default"`
			Profiles []listProfile `json:"profiles"`
		}{Default: def, Profiles: []listProfile{}}
		for _, prof := range profiles {
			doc.Profiles = append(doc.Profiles, listProfile{
				Name:               prof.Name,
				Default:            prof.Name == def,
				Server:             prof.Server,
				Shell:              prof.Shell,
				Role:               prof.Role,
				Trust:              prof.Trust,
				Source:             prof.Source,
				CredentialRef:      prof.CredentialRef,
				CredentialRedacted: prof.CredentialRedacted,
			})
		}
		enc := json.NewEncoder(c.out)
		if err := enc.Encode(doc); err != nil {
			return c.denyProfile("default", "profile list", "profile-not-found")
		}
		return 0
	}
	if len(profiles) == 0 {
		fmt.Fprintln(c.out, "no profiles")
		return 0
	}
	for _, prof := range profiles {
		mark := "-"
		if prof.Name == def {
			mark = "*"
		}
		fmt.Fprintf(c.out, "%s %s %s %s\n", mark, prof.Name, prof.Role, prof.Trust)
	}
	return 0
}

func (c cli) use(name string) int {
	profiles, err := c.loadProfiles()
	if err != nil || find(profiles, name) == nil {
		return c.denyProfile(name, "profile use", "profile-not-found")
	}
	if err := write0600(filepath.Join(c.configDir(), "default-profile"), []byte(name+"\n")); err != nil {
		return c.denyProfile(name, "profile use", "profile-not-found")
	}
	fmt.Fprintf(c.out, "profile %s selected\n", name)
	return 0
}

func (c cli) trigger(args []string) int {
	intent, profile, reqID, jsonOut, ok := parseTrigger(args)
	if !ok {
		return c.usage()
	}
	profiles, _ := c.loadProfiles()
	if profile == "" {
		profile = c.defaultProfile()
		if profile == "" {
			return c.denyTrigger("default", intent, "profile-not-found", reqID, jsonOut, nil)
		}
	}
	prof := find(profiles, profile)
	if prof == nil {
		return c.denyTrigger(profile, intent, "profile-not-found", reqID, jsonOut, nil)
	}
	if intent != "bundle.clock.tick" {
		return c.denyTrigger(profile, intent, "unknown-trigger", reqID, jsonOut, prof)
	}
	creds := filepath.Join(c.dataDir(), filepath.FromSlash(prof.CredentialRef))
	if _, err := os.Stat(creds); err != nil {
		return c.denyTrigger(profile, intent, "stale-credentials", reqID, jsonOut, prof)
	}
	if c.deniedNeighbor(*prof) {
		return c.denyTrigger(profile, intent, "denied-neighbor", reqID, jsonOut, prof)
	}
	return c.triggerLive(*prof, intent, reqID, jsonOut, creds)
}

func (c cli) triggerLive(prof Profile, intent, reqID string, jsonOut bool, creds string) int {
	nc, err := nats.Connect(prof.Server, nats.UserCredentials(creds), nats.NoReconnect(), nats.Timeout(2*time.Second), nats.ErrorHandler(func(*nats.Conn, *nats.Subscription, error) {}))
	if err != nil {
		return c.denyTrigger(prof.Name, intent, authReason(err, prof), reqID, jsonOut, &prof)
	}
	defer nc.Close()
	if reqID == "" {
		reqID = "tinkalet-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	msg := nats.NewMsg(subjectFor(intent))
	msg.Header.Set(embednats.HeaderRequestID, reqID)
	msg.Data = []byte(intent)
	reply, err := nc.RequestMsg(msg, 5*time.Second)
	if err != nil {
		return c.denyTrigger(prof.Name, intent, authReason(err, prof), reqID, jsonOut, &prof)
	}
	status := strings.TrimSpace(string(reply.Data))
	switch status {
	case "accepted", "duplicate":
		return c.okTrigger(prof, intent, status, reqID, jsonOut)
	default:
		return c.denyTrigger(prof.Name, intent, "malformed-response", reqID, jsonOut, &prof)
	}
}

const itemBucket = "tb_items"

type itemStored struct {
	Kind       string          `json:"kind"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	Reason     string          `json:"reason,omitempty"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProvenance  `json:"provenance"`
}

type itemProvenance struct {
	Profile string `json:"profile"`
	Source  string `json:"source"`
	Writer  string `json:"writer"`
}

type itemView struct {
	Kind       string          `json:"kind"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	Reason     string          `json:"reason,omitempty"`
	Revision   uint64          `json:"revision"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProvenance  `json:"provenance"`
}

func (c cli) item(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "create":
		key, status, value, jsonOut, ok := parseItemCreate(args[1:])
		if !ok || !validItemKey(key) || status != "pending" {
			return c.usage()
		}
		val, valid := jsonValue(value)
		if !valid {
			return c.denyItem(key, "create", "malformed-value")
		}
		return c.itemCreate(key, status, val, jsonOut)
	case "get":
		key, jsonOut, ok := parseItemGet(args[1:])
		if !ok || !validItemKey(key) {
			return c.usage()
		}
		return c.itemGet(key, jsonOut)
	case "resolve":
		key, value, rev, revSet, jsonOut, ok := parseItemResolve(args[1:])
		if !ok || !validItemKey(key) {
			return c.usage()
		}
		val, valid := jsonValue(value)
		if !valid {
			return c.denyItem(key, "resolve", "malformed-value")
		}
		return c.itemResolve(key, val, rev, revSet, jsonOut)
	case "wait":
		key, want, timeout, jsonOut, ok := parseItemWait(args[1:])
		if !ok || !validItemKey(key) || want != "resolved" {
			return c.usage()
		}
		return c.itemWait(key, want, timeout, jsonOut)
	default:
		return c.usage()
	}
}

func parseItemCreate(args []string) (string, string, string, bool, bool) {
	status := "pending"
	var key, value string
	var jsonOut bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			jsonOut = true
		case arg == "--status" && i+1 < len(args):
			i++
			status = args[i]
		case strings.HasPrefix(arg, "--status="):
			status = strings.TrimPrefix(arg, "--status=")
		case arg == "--value" && i+1 < len(args):
			i++
			value = args[i]
		case strings.HasPrefix(arg, "--value="):
			value = strings.TrimPrefix(arg, "--value=")
		case strings.HasPrefix(arg, "-"):
			return "", "", "", false, false
		case key == "":
			key = arg
		default:
			return "", "", "", false, false
		}
	}
	return key, status, value, jsonOut, key != ""
}

func parseItemGet(args []string) (string, bool, bool) {
	var key string
	var jsonOut bool
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOut = true
		case strings.HasPrefix(arg, "-"):
			return "", false, false
		case key == "":
			key = arg
		default:
			return "", false, false
		}
	}
	return key, jsonOut, key != ""
}

func parseItemResolve(args []string) (string, string, uint64, bool, bool, bool) {
	var key, value string
	var rev uint64
	var revSet bool
	var jsonOut bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			jsonOut = true
		case arg == "--value" && i+1 < len(args):
			i++
			value = args[i]
		case strings.HasPrefix(arg, "--value="):
			value = strings.TrimPrefix(arg, "--value=")
		case arg == "--revision" && i+1 < len(args):
			i++
			parsed, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return "", "", 0, false, false, false
			}
			rev = parsed
			revSet = true
		case strings.HasPrefix(arg, "--revision="):
			parsed, err := strconv.ParseUint(strings.TrimPrefix(arg, "--revision="), 10, 64)
			if err != nil {
				return "", "", 0, false, false, false
			}
			rev = parsed
			revSet = true
		case strings.HasPrefix(arg, "-"):
			return "", "", 0, false, false, false
		case key == "":
			key = arg
		default:
			return "", "", 0, false, false, false
		}
	}
	return key, value, rev, revSet, jsonOut, key != ""
}

func parseItemWait(args []string) (string, string, time.Duration, bool, bool) {
	timeout := 30 * time.Second
	var key, want string
	var jsonOut bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			jsonOut = true
		case arg == "--for" && i+1 < len(args):
			i++
			want = args[i]
		case strings.HasPrefix(arg, "--for="):
			want = strings.TrimPrefix(arg, "--for=")
		case arg == "--timeout" && i+1 < len(args):
			i++
			d, err := time.ParseDuration(args[i])
			if err != nil {
				return "", "", 0, false, false
			}
			timeout = d
		case strings.HasPrefix(arg, "--timeout="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return "", "", 0, false, false
			}
			timeout = d
		case strings.HasPrefix(arg, "-"):
			return "", "", 0, false, false
		case key == "":
			key = arg
		default:
			return "", "", 0, false, false
		}
	}
	return key, want, timeout, jsonOut, key != "" && want != "" && timeout > 0
}

func (c cli) itemCreate(key, status string, value json.RawMessage, jsonOut bool) int {
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "create", reason)
	}
	defer nc.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	rec := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       key,
		Status:    status,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  prof.Source,
			Writer:  "tinkalet",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return c.denyItem(key, "create", "malformed-value")
	}
	rev, err := kv.Create(key, body)
	if err != nil {
		if errors.Is(err, nats.ErrKeyExists) {
			return c.denyItem(key, "create", "duplicate-item")
		}
		return c.denyItem(key, "create", itemReason(err, *prof))
	}
	return c.okItem(viewItem(rec, rev), jsonOut)
}

func (c cli) itemGet(key string, jsonOut bool) int {
	_, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "get", reason)
	}
	defer nc.Close()
	item, reason := readItem(kv, key)
	if reason != "" {
		return c.denyItem(key, "get", reason)
	}
	return c.okItem(item, jsonOut)
}

func (c cli) itemResolve(key string, value json.RawMessage, rev uint64, revSet, jsonOut bool) int {
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "resolve", reason)
	}
	defer nc.Close()
	current, reason := readItem(kv, key)
	if reason != "" {
		return c.denyItem(key, "resolve", reason)
	}
	if !revSet {
		rev = current.Revision
	}
	rec := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       key,
		Status:    "resolved",
		Value:     value,
		CreatedAt: current.CreatedAt,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  prof.Source,
			Writer:  "tinkalet",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return c.denyItem(key, "resolve", "malformed-value")
	}
	next, err := kv.Update(key, body, rev)
	if err != nil {
		return c.denyItem(key, "resolve", itemReason(err, *prof))
	}
	return c.okItem(viewItem(rec, next), jsonOut)
}

func (c cli) itemWait(key, want string, timeout time.Duration, jsonOut bool) int {
	_, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "wait", reason)
	}
	defer nc.Close()
	deadline := time.Now().Add(timeout)
	for {
		item, reason := readItem(kv, key)
		if reason != "" {
			return c.denyItem(key, "wait", reason)
		}
		if item.Status == want {
			return c.okItem(item, jsonOut)
		}
		if time.Now().After(deadline) {
			return c.denyItem(key, "wait", "wait-timeout")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c cli) itemKV() (*Profile, nats.KeyValue, *nats.Conn, string) {
	profiles, _ := c.loadProfiles()
	name := c.defaultProfile()
	if name == "" {
		return nil, nil, nil, "profile-not-found"
	}
	prof := find(profiles, name)
	if prof == nil {
		return nil, nil, nil, "profile-not-found"
	}
	creds := filepath.Join(c.dataDir(), filepath.FromSlash(prof.CredentialRef))
	if _, err := os.Stat(creds); err != nil {
		return prof, nil, nil, "stale-credentials"
	}
	if c.deniedNeighbor(*prof) {
		return prof, nil, nil, "denied-neighbor"
	}
	nc, err := nats.Connect(prof.Server, nats.UserCredentials(creds), nats.NoReconnect(), nats.Timeout(2*time.Second), nats.ErrorHandler(func(*nats.Conn, *nats.Subscription, error) {}))
	if err != nil {
		return prof, nil, nil, authReason(err, *prof)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return prof, nil, nil, "connection-failed"
	}
	kv, err := js.KeyValue(itemBucket)
	if err != nil {
		nc.Close()
		return prof, nil, nil, itemReason(err, *prof)
	}
	return prof, kv, nc, ""
}

func readItem(kv nats.KeyValue, key string) (itemView, string) {
	entry, err := kv.Get(key)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			return itemView{}, "item-not-found"
		}
		return itemView{}, itemReason(err, Profile{})
	}
	var rec itemStored
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != "tinkabot.item.v1" || rec.Key != key {
		return itemView{}, "malformed-item"
	}
	return viewItem(rec, entry.Revision()), ""
}

func viewItem(rec itemStored, rev uint64) itemView {
	return itemView{
		Kind:       rec.Kind,
		Key:        rec.Key,
		Status:     rec.Status,
		Value:      rec.Value,
		Reason:     rec.Reason,
		Revision:   rev,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
		Provenance: rec.Provenance,
	}
}

func itemReason(err error, prof Profile) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "wrong last sequence"):
		return "stale-revision"
	case strings.Contains(msg, "key exists"):
		return "duplicate-item"
	case strings.Contains(msg, "not found"):
		return "item-not-found"
	case strings.Contains(msg, "authorization") || strings.Contains(msg, "authentication") || strings.Contains(msg, "permission"):
		return authReason(err, prof)
	default:
		return "connection-failed"
	}
}

func (c cli) okItem(item itemView, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(item)
		return 0
	}
	if string(item.Value) != "null" && len(item.Value) > 0 {
		fmt.Fprintf(c.out, "item %s %s rev %d value %s\n", item.Key, item.Status, item.Revision, item.Value)
		return 0
	}
	fmt.Fprintf(c.out, "item %s %s rev %d\n", item.Key, item.Status, item.Revision)
	return 0
}

func (c cli) denyItem(key, action, reason string) int {
	fmt.Fprintf(c.errOut, "item %s denied %s: %s\n", key, action, reason)
	return 1
}

func jsonValue(raw string) (json.RawMessage, bool) {
	if raw == "" {
		return json.RawMessage("null"), true
	}
	if !json.Valid([]byte(raw)) {
		return nil, false
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(raw)); err != nil {
		return nil, false
	}
	return json.RawMessage(buf.Bytes()), true
}

func validItemKey(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, "//") {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == '/':
		default:
			return false
		}
	}
	return !strings.Contains(key, "..")
}

func (c cli) deniedNeighbor(prof Profile) bool {
	const prefix = "local-store:"
	if !strings.HasPrefix(prof.Source, prefix) {
		return false
	}
	store := strings.TrimPrefix(prof.Source, prefix)
	body, err := os.ReadFile(filepath.Join(store, "local-profile.json"))
	if err != nil {
		return false
	}
	var desc descriptor
	if err := json.Unmarshal(body, &desc); err != nil {
		return false
	}
	return desc.Server != "" && desc.Server != prof.Server
}

func authReason(err error, prof Profile) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "revoked"):
		return "revoked-credentials"
	case strings.Contains(msg, "authorization") || strings.Contains(msg, "authentication"):
		if strings.HasPrefix(prof.Source, "local-store:") {
			return "revoked-credentials"
		}
		return "denied-trigger"
	default:
		return "connection-failed"
	}
}

func subjectFor(intent string) string {
	switch intent {
	case "bundle.clock.tick":
		return "tb.bundle.clock.tick"
	default:
		return ""
	}
}

func (c cli) okTrigger(prof Profile, intent, status, reqID string, jsonOut bool) int {
	if jsonOut {
		doc := triggerDoc{Profile: prof.Name, Intent: intent, Status: status, Reason: "", RequestID: reqID}
		doc.Diagnostics.Server = prof.Server
		doc.Diagnostics.Shell = prof.Shell
		_ = json.NewEncoder(c.out).Encode(doc)
		return 0
	}
	fmt.Fprintf(c.out, "profile %s %s %s\n", prof.Name, status, intent)
	return 0
}

func parseTrigger(args []string) (string, string, string, bool, bool) {
	var intent, profile, reqID string
	var jsonOut bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			jsonOut = true
		case arg == "--profile" && i+1 < len(args):
			i++
			profile = args[i]
		case strings.HasPrefix(arg, "--profile="):
			profile = strings.TrimPrefix(arg, "--profile=")
		case arg == "--request-id" && i+1 < len(args):
			i++
			reqID = args[i]
		case strings.HasPrefix(arg, "--request-id="):
			reqID = strings.TrimPrefix(arg, "--request-id=")
		case strings.HasPrefix(arg, "-"):
			return "", "", "", false, false
		case intent == "":
			intent = arg
		default:
			return "", "", "", false, false
		}
	}
	return intent, profile, reqID, jsonOut, intent != ""
}

func (c cli) denyProfile(profile, action, reason string) int {
	fmt.Fprintf(c.errOut, "profile %s denied %s: %s\n", profile, action, reason)
	return 1
}

func (c cli) denyTrigger(profile, intent, reason, reqID string, jsonOut bool, prof *Profile) int {
	if jsonOut {
		doc := triggerDoc{Profile: profile, Intent: intent, Status: "denied", Reason: reason, RequestID: reqID}
		if prof != nil {
			doc.Diagnostics.Server = prof.Server
			doc.Diagnostics.Shell = prof.Shell
		}
		_ = json.NewEncoder(c.out).Encode(doc)
		return 1
	}
	fmt.Fprintf(c.errOut, "profile %s denied %s: %s\n", profile, intent, reason)
	return 1
}

type triggerDoc struct {
	Profile     string `json:"profile"`
	Intent      string `json:"intent"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	RequestID   string `json:"requestId"`
	Diagnostics struct {
		Server string `json:"server"`
		Shell  string `json:"shell"`
	} `json:"diagnostics"`
}

func (c cli) loadProfiles() ([]Profile, error) {
	body, err := os.ReadFile(filepath.Join(c.configDir(), "profiles.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var file profilesFile
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, err
	}
	sortProfiles(file.Profiles)
	return file.Profiles, nil
}

func (c cli) saveProfiles(profiles []Profile) error {
	sortProfiles(profiles)
	body, err := json.MarshalIndent(profilesFile{Profiles: profiles}, "", "  ")
	if err != nil {
		return err
	}
	return write0600(filepath.Join(c.configDir(), "profiles.json"), append(body, '\n'))
}

func (c cli) defaultProfile() string {
	body, err := os.ReadFile(filepath.Join(c.configDir(), "default-profile"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func (c cli) configDir() string {
	if dir := c.env["TINKALET_CONFIG_DIR"]; dir != "" {
		return dir
	}
	if dir := c.env["XDG_CONFIG_HOME"]; dir != "" {
		return filepath.Join(dir, "tinkalet")
	}
	if home := c.env["HOME"]; home != "" {
		return filepath.Join(home, ".config", "tinkalet")
	}
	return ".tinkalet"
}

func (c cli) dataDir() string {
	if dir := c.env["TINKALET_DATA_DIR"]; dir != "" {
		return dir
	}
	if dir := c.env["XDG_STATE_HOME"]; dir != "" {
		return filepath.Join(dir, "tinkalet")
	}
	if home := c.env["HOME"]; home != "" {
		return filepath.Join(home, ".local", "state", "tinkalet")
	}
	return ".tinkalet-data"
}

func envMap(env []string) map[string]string {
	out := map[string]string{}
	for _, item := range env {
		key, val, ok := strings.Cut(item, "=")
		if ok {
			out[key] = val
		}
	}
	return out
}

func validName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name
}

func write0600(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func upsert(profiles *[]Profile, prof Profile) {
	for i := range *profiles {
		if (*profiles)[i].Name == prof.Name {
			(*profiles)[i] = prof
			return
		}
	}
	*profiles = append(*profiles, prof)
}

func find(profiles []Profile, name string) *Profile {
	for i := range profiles {
		if profiles[i].Name == name {
			return &profiles[i]
		}
	}
	return nil
}

func sortProfiles(profiles []Profile) {
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
}
