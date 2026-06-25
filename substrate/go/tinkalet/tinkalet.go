package tinkalet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
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
	AppID              string `json:"appId,omitempty"`
	ParticipantID      string `json:"participantId,omitempty"`
	WatchScope         string `json:"watchScope,omitempty"`
	WatchTarget        string `json:"watchTarget,omitempty"`
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
	AppID              string `json:"appId,omitempty"`
	ParticipantID      string `json:"participantId,omitempty"`
	WatchScope         string `json:"watchScope,omitempty"`
	WatchTarget        string `json:"watchTarget,omitempty"`
	CredentialRef      string `json:"credentialRef"`
	CredentialRedacted bool   `json:"credentialRedacted"`
}

type descriptor struct {
	Kind          string `json:"kind"`
	Server        string `json:"server"`
	Shell         string `json:"shell"`
	Credential    string `json:"credential"`
	Role          string `json:"role"`
	Trust         string `json:"trust"`
	Source        string `json:"source"`
	Status        string `json:"status"`
	AppID         string `json:"appId"`
	ParticipantID string `json:"participantId"`
	WatchScope    string `json:"watchScope"`
	WatchTarget   string `json:"watchTarget"`
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
	case "action":
		return c.action(args[1:])
	case "schedule":
		return c.schedule(args[1:])
	case "reaction":
		return c.reaction(args[1:])
	case "watch":
		return c.watch(args[1:], false)
	case "daemon":
		if len(args) > 1 && args[1] == "watch" {
			return c.watch(args[2:], true)
		}
		if len(args) > 1 && args[1] == "react" {
			return c.react(args[2:])
		}
		return c.usage()
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
  action submit <action-id> --state <item-key> --base-revision <n> [--value <json>] [--json]
  action apply <action-key> --value <json> [--json]
  action reject <action-key> --reason <reason-token> [--json]
  schedule set <name> --every <duration> --write <item-key> [--value <json>] [--json]
  schedule off <name> [--json]
  watch item <key> [--cursor <name>] [--limit <n>] [--timeout <duration>] [--json]
  watch prefix <prefix> [--cursor <name>] [--limit <n>] [--timeout <duration>] [--json]
  daemon watch item <key> --cursor <name> [--limit <n>] [--timeout <duration>] [--json]
  daemon watch prefix <prefix> --cursor <name> [--limit <n>] [--timeout <duration>] [--json]
  reaction add <name> --watch item <key> --for resolved --cmd <path> [--arg <arg>] --write <item-key>
  daemon react <name> --once [--timeout <duration>] [--json]
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
	credName := filepath.Base(cred)
	if credName == "." || credName == string(filepath.Separator) {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	dst := filepath.Join(dataDir, "profiles", name, credName)
	body, err := os.ReadFile(cred)
	if err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	if err := write0600(dst, body); err != nil {
		return c.denyProfile(name, "profile import", "import-source-invalid")
	}
	prof.CredentialRef = filepath.ToSlash(filepath.Join("profiles", name, credName))
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
	if desc.Kind != "tinkabot.localProfile.v1" || desc.Source != "local-store:"+store {
		return Profile{}, "", "import-source-invalid"
	}
	switch desc.Trust {
	case "local-owner":
		if desc.Role != "caller" && desc.Role != "edge" {
			return Profile{}, "", "import-source-invalid"
		}
	case "app-participant":
		if desc.Role != "participant" || desc.Status == "revoked" || !validSubjectToken(desc.AppID) || !validSubjectToken(desc.ParticipantID) {
			return Profile{}, "", "import-source-invalid"
		}
	case "item-watcher":
		if desc.Role != "watcher" || desc.Status == "revoked" || !validWatcherScope(desc.WatchScope, desc.WatchTarget) {
			return Profile{}, "", "import-source-invalid"
		}
	default:
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
		Name:          name,
		Server:        desc.Server,
		Shell:         desc.Shell,
		Role:          desc.Role,
		Trust:         desc.Trust,
		Source:        desc.Source,
		AppID:         desc.AppID,
		ParticipantID: desc.ParticipantID,
		WatchScope:    desc.WatchScope,
		WatchTarget:   desc.WatchTarget,
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
				AppID:              prof.AppID,
				ParticipantID:      prof.ParticipantID,
				WatchScope:         prof.WatchScope,
				WatchTarget:        prof.WatchTarget,
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
	if subjectFor(intent) == "" {
		return c.denyTrigger(profile, intent, "unknown-trigger", reqID, jsonOut, prof)
	}
	creds := filepath.Join(c.dataDir(), filepath.FromSlash(prof.CredentialRef))
	if _, err := os.Stat(creds); err != nil {
		return c.denyTrigger(profile, intent, "stale-credentials", reqID, jsonOut, prof)
	}
	if c.deniedNeighbor(*prof) {
		return c.denyTrigger(profile, intent, "denied-neighbor", reqID, jsonOut, prof)
	}
	if c.revokedProfile(*prof) {
		return c.denyTrigger(profile, intent, "revoked-credentials", reqID, jsonOut, prof)
	}
	if restrictedProfile(*prof) {
		return c.denyTrigger(profile, intent, "denied-scope", reqID, jsonOut, prof)
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

const (
	itemBucket        = "tb_items"
	scheduleBucket    = "tb_schedules"
	minScheduleEvery  = 100 * time.Millisecond
	scheduleRecordV1  = "tinkabot.schedule.v1"
	scheduleStatusOn  = "active"
	scheduleStatusOff = "off"
)

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

type actionSubmitReq struct {
	ActionID     string
	StateKey     string
	BaseRevision uint64
	ValueRaw     string
	JSON         bool
}

type actionApplyReq struct {
	ActionKey string
	ValueRaw  string
	JSON      bool
}

type actionRejectReq struct {
	ActionKey string
	Reason    string
	JSON      bool
}

type actionWireReq struct {
	ActionID     string          `json:"actionId"`
	StateKey     string          `json:"stateKey"`
	BaseRevision uint64          `json:"baseRevision"`
	Value        json.RawMessage `json:"value"`
}

type actionWireResp struct {
	Status string    `json:"status"`
	Reason string    `json:"reason,omitempty"`
	Item   *itemView `json:"item,omitempty"`
}

type appActionStored struct {
	Kind          string          `json:"kind"`
	AppID         string          `json:"appId"`
	ParticipantID string          `json:"participantId"`
	ActionID      string          `json:"actionId"`
	StateKey      string          `json:"stateKey"`
	BaseRevision  uint64          `json:"baseRevision"`
	Payload       json.RawMessage `json:"payload"`
}

type actionReceiptStored struct {
	Kind           string `json:"kind"`
	ActionKey      string `json:"actionKey"`
	StateKey       string `json:"stateKey"`
	ActionRevision uint64 `json:"actionRevision"`
	StateRevision  uint64 `json:"stateRevision"`
	Outcome        string `json:"outcome,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type scheduleRecord struct {
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	EveryMs    int64           `json:"everyMs"`
	WriteItem  string          `json:"writeItem"`
	Value      json.RawMessage `json:"value"`
	Sequence   int             `json:"sequence"`
	LastTickAt string          `json:"lastTickAt,omitempty"`
	UpdatedAt  string          `json:"updatedAt"`
	Provenance itemProvenance  `json:"provenance"`
}

type scheduleSetReq struct {
	Name      string
	EveryRaw  string
	WriteItem string
	ValueRaw  string
	JSON      bool
}

type watchReq struct {
	Scope   string
	Target  string
	Cursor  string
	Limit   int
	Timeout time.Duration
	JSON    bool
}

type watchCursor struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Profile   string `json:"profile"`
	Source    string `json:"source"`
	Scope     string `json:"scope"`
	Target    string `json:"target"`
	Revision  uint64 `json:"revision"`
	UpdatedAt string `json:"updatedAt"`
}

type watchEvent struct {
	Kind       string          `json:"kind"`
	Cursor     string          `json:"cursor,omitempty"`
	Scope      string          `json:"scope"`
	Key        string          `json:"key"`
	Status     string          `json:"status"`
	Value      json.RawMessage `json:"value"`
	Revision   uint64          `json:"revision"`
	Source     string          `json:"source"`
	ObservedAt string          `json:"observedAt"`
}

type reactionRecord struct {
	Kind      string          `json:"kind"`
	Name      string          `json:"name"`
	Profile   string          `json:"profile"`
	Source    string          `json:"source"`
	Watch     reactionWatch   `json:"watch"`
	Command   reactionCommand `json:"command"`
	Write     reactionWrite   `json:"write"`
	CreatedAt string          `json:"createdAt"`
}

type reactionWatch struct {
	Scope  string `json:"scope"`
	Target string `json:"target"`
	Status string `json:"status"`
}

type reactionCommand struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
}

type reactionWrite struct {
	Item string `json:"item"`
}

type reactionOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
}

type reactionRunDoc struct {
	Reaction string `json:"reaction"`
	Status   string `json:"status"`
	Item     string `json:"item"`
	ExitCode int    `json:"exitCode"`
}

func (c cli) schedule(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "set":
		req, ok := parseScheduleSet(args[1:])
		if !ok || !validName(req.Name) || !validItemKey(req.WriteItem) {
			return c.usage()
		}
		every, err := time.ParseDuration(req.EveryRaw)
		if err != nil || every < minScheduleEvery {
			return c.denySchedule(req.Name, "set", "malformed-duration")
		}
		val, valid := jsonValue(req.ValueRaw)
		if !valid {
			return c.denySchedule(req.Name, "set", "malformed-value")
		}
		return c.scheduleSet(req, every, val)
	case "off":
		name, jsonOut, ok := parseScheduleOff(args[1:])
		if !ok || !validName(name) {
			return c.usage()
		}
		return c.scheduleOff(name, jsonOut)
	default:
		return c.usage()
	}
}

func parseScheduleSet(args []string) (scheduleSetReq, bool) {
	var req scheduleSetReq
	if len(args) == 0 {
		return req, false
	}
	req.Name = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--every" && i+1 < len(args):
			i++
			req.EveryRaw = args[i]
		case strings.HasPrefix(arg, "--every="):
			req.EveryRaw = strings.TrimPrefix(arg, "--every=")
		case arg == "--write" && i+1 < len(args):
			i++
			req.WriteItem = args[i]
		case strings.HasPrefix(arg, "--write="):
			req.WriteItem = strings.TrimPrefix(arg, "--write=")
		case arg == "--value" && i+1 < len(args):
			i++
			req.ValueRaw = args[i]
		case strings.HasPrefix(arg, "--value="):
			req.ValueRaw = strings.TrimPrefix(arg, "--value=")
		default:
			return req, false
		}
	}
	return req, req.Name != "" && req.EveryRaw != "" && req.WriteItem != ""
}

func parseScheduleOff(args []string) (string, bool, bool) {
	var name string
	var jsonOut bool
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOut = true
		case strings.HasPrefix(arg, "-"):
			return "", false, false
		case name == "":
			name = arg
		default:
			return "", false, false
		}
	}
	return name, jsonOut, name != ""
}

func (c cli) scheduleSet(req scheduleSetReq, every time.Duration, value json.RawMessage) int {
	prof, kv, nc, reason := c.scheduleKV()
	if reason != "" {
		return c.denySchedule(req.Name, "set", reason)
	}
	defer nc.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	rec := scheduleRecord{
		Kind:      scheduleRecordV1,
		Name:      req.Name,
		Status:    scheduleStatusOn,
		EveryMs:   every.Milliseconds(),
		WriteItem: req.WriteItem,
		Value:     value,
		UpdatedAt: now,
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  prof.Source,
			Writer:  "tinkalet",
		},
	}
	if entry, err := kv.Get(req.Name); err == nil {
		var prev scheduleRecord
		if json.Unmarshal(entry.Value(), &prev) == nil && prev.Kind == scheduleRecordV1 && prev.Name == req.Name {
			rec.Sequence = prev.Sequence
			rec.LastTickAt = prev.LastTickAt
		}
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return c.denySchedule(req.Name, "set", "malformed-value")
	}
	if _, err := kv.Put(req.Name, body); err != nil {
		return c.denySchedule(req.Name, "set", scheduleReason(err, *prof))
	}
	return c.okSchedule(rec, every, req.JSON)
}

func (c cli) scheduleOff(name string, jsonOut bool) int {
	prof, kv, nc, reason := c.scheduleKV()
	if reason != "" {
		return c.denySchedule(name, "off", reason)
	}
	defer nc.Close()
	entry, err := kv.Get(name)
	if err != nil {
		return c.denySchedule(name, "off", scheduleReason(err, *prof))
	}
	var rec scheduleRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != scheduleRecordV1 || rec.Name != name {
		return c.denySchedule(name, "off", "schedule-invalid")
	}
	rec.Status = scheduleStatusOff
	rec.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	rec.Provenance = itemProvenance{Profile: prof.Name, Source: prof.Source, Writer: "tinkalet"}
	body, err := json.Marshal(rec)
	if err != nil {
		return c.denySchedule(name, "off", "schedule-invalid")
	}
	if _, err := kv.Update(name, body, entry.Revision()); err != nil {
		return c.denySchedule(name, "off", scheduleReason(err, *prof))
	}
	return c.okSchedule(rec, time.Duration(rec.EveryMs)*time.Millisecond, jsonOut)
}

func (c cli) scheduleKV() (*Profile, nats.KeyValue, *nats.Conn, string) {
	name := c.defaultProfile()
	if name == "" {
		return nil, nil, nil, "profile-not-found"
	}
	profiles, _ := c.loadProfiles()
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
	if c.revokedProfile(*prof) {
		return prof, nil, nil, "revoked-credentials"
	}
	if restrictedProfile(*prof) {
		return prof, nil, nil, "denied-scope"
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
	kv, err := js.KeyValue(scheduleBucket)
	if err != nil {
		nc.Close()
		return prof, nil, nil, scheduleReason(err, *prof)
	}
	return prof, kv, nc, ""
}

func scheduleReason(err error, prof Profile) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "wrong last sequence"):
		return "stale-revision"
	case strings.Contains(msg, "not found"):
		return "schedule-not-found"
	case strings.Contains(msg, "authorization") || strings.Contains(msg, "authentication") || strings.Contains(msg, "permission"):
		return authReason(err, prof)
	default:
		if restrictedProfile(prof) {
			return "denied-scope"
		}
		return "connection-failed"
	}
}

func (c cli) okSchedule(rec scheduleRecord, every time.Duration, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(rec)
		return 0
	}
	if rec.Status == scheduleStatusOff {
		fmt.Fprintf(c.out, "schedule %s off\n", rec.Name)
		return 0
	}
	fmt.Fprintf(c.out, "schedule %s active every %s -> %s\n", rec.Name, every, rec.WriteItem)
	return 0
}

func (c cli) denySchedule(name, action, reason string) int {
	fmt.Fprintf(c.errOut, "schedule %s denied %s: %s\n", name, action, reason)
	return 1
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

func (c cli) action(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "submit":
		req, ok := parseActionSubmit(args[1:])
		if !ok || !validSubjectToken(req.ActionID) || !validItemKey(req.StateKey) || req.BaseRevision == 0 {
			return c.usage()
		}
		val, valid := jsonValue(req.ValueRaw)
		if !valid {
			return c.denyAction(req.ActionID, "submit", "malformed-value")
		}
		return c.actionSubmit(req, val)
	case "apply":
		req, ok := parseActionApply(args[1:])
		if !ok || !validItemKey(req.ActionKey) {
			return c.usage()
		}
		val, valid := jsonValue(req.ValueRaw)
		if !valid {
			return c.denyAction(req.ActionKey, "apply", "malformed-value")
		}
		return c.actionApply(req, val)
	case "reject":
		req, ok := parseActionReject(args[1:])
		if !ok || !validItemKey(req.ActionKey) || !validSubjectToken(req.Reason) {
			return c.usage()
		}
		return c.actionReject(req)
	default:
		return c.usage()
	}
}

func (c cli) reaction(args []string) int {
	if len(args) == 0 {
		return c.usage()
	}
	switch args[0] {
	case "add":
		rec, ok := parseReactionAdd(args[1:])
		if !ok || !validName(rec.Name) || rec.Watch.Scope != "item" || rec.Watch.Status != "resolved" || !validItemKey(rec.Watch.Target) || !validItemKey(rec.Write.Item) {
			return c.usage()
		}
		return c.reactionAdd(rec)
	default:
		return c.usage()
	}
}

func parseReactionAdd(args []string) (reactionRecord, bool) {
	var rec reactionRecord
	if len(args) == 0 {
		return rec, false
	}
	rec.Name = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--watch" && i+2 < len(args):
			i++
			rec.Watch.Scope = args[i]
			i++
			rec.Watch.Target = args[i]
		case arg == "--for" && i+1 < len(args):
			i++
			rec.Watch.Status = args[i]
		case arg == "--cmd" && i+1 < len(args):
			i++
			rec.Command.Cmd = args[i]
		case arg == "--arg" && i+1 < len(args):
			i++
			rec.Command.Args = append(rec.Command.Args, args[i])
		case arg == "--write" && i+1 < len(args):
			i++
			rec.Write.Item = args[i]
		default:
			return rec, false
		}
	}
	return rec, rec.Name != "" && rec.Watch.Scope != "" && rec.Watch.Target != "" && rec.Watch.Status != "" && rec.Command.Cmd != "" && rec.Write.Item != ""
}

func (c cli) reactionAdd(rec reactionRecord) int {
	profiles, _ := c.loadProfiles()
	name := c.defaultProfile()
	if name == "" {
		return c.denyReaction(rec.Name, "add", "profile-not-found")
	}
	prof := find(profiles, name)
	if prof == nil {
		return c.denyReaction(rec.Name, "add", "profile-not-found")
	}
	if c.deniedNeighbor(*prof) {
		return c.denyReaction(rec.Name, "add", "denied-neighbor")
	}
	if c.revokedProfile(*prof) {
		return c.denyReaction(rec.Name, "add", "revoked-credentials")
	}
	if restrictedProfile(*prof) {
		return c.denyReaction(rec.Name, "add", "denied-scope")
	}
	rec.Kind = "tinkalet.reaction.v1"
	rec.Profile = prof.Name
	rec.Source = prof.Source
	rec.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return c.denyReaction(rec.Name, "add", "reaction-invalid")
	}
	if err := write0600(c.reactionPath(rec.Name), append(body, '\n')); err != nil {
		return c.denyReaction(rec.Name, "add", "reaction-invalid")
	}
	fmt.Fprintf(c.out, "reaction %s added\n", rec.Name)
	return 0
}

func (c cli) react(args []string) int {
	name, timeout, jsonOut, ok := parseReact(args)
	if !ok || !validName(name) {
		return c.usage()
	}
	rec, reason := c.loadReaction(name)
	if reason != "" {
		return c.denyReaction(name, "run", reason)
	}
	return c.runReaction(rec, timeout, jsonOut)
}

func parseReact(args []string) (string, time.Duration, bool, bool) {
	timeout := 30 * time.Second
	var name string
	var once, jsonOut bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--once":
			once = true
		case arg == "--json":
			jsonOut = true
		case arg == "--timeout" && i+1 < len(args):
			i++
			parsed, err := time.ParseDuration(args[i])
			if err != nil || parsed <= 0 {
				return "", 0, false, false
			}
			timeout = parsed
		case strings.HasPrefix(arg, "--timeout="):
			parsed, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil || parsed <= 0 {
				return "", 0, false, false
			}
			timeout = parsed
		case strings.HasPrefix(arg, "-"):
			return "", 0, false, false
		case name == "":
			name = arg
		default:
			return "", 0, false, false
		}
	}
	return name, timeout, jsonOut, name != "" && once
}

func (c cli) runReaction(rec reactionRecord, timeout time.Duration, jsonOut bool) int {
	prof, kv, nc, reason := c.itemKVFor(rec.Profile)
	if reason != "" {
		return c.denyReaction(rec.Name, "run", reason)
	}
	defer nc.Close()
	if restrictedProfile(*prof) {
		return c.denyReaction(rec.Name, "run", "denied-scope")
	}
	req := watchReq{Scope: rec.Watch.Scope, Target: rec.Watch.Target, Cursor: "reaction-" + rec.Name, Limit: 1, Timeout: timeout}
	cur, reason := c.loadCursor(req, *prof)
	if reason != "" {
		return c.denyReaction(rec.Name, "run", reason)
	}
	if reason := validateCursor(kv, cur); reason != "" {
		return c.denyReaction(rec.Name, "run", reason)
	}
	w, err := kv.WatchAll(nats.IncludeHistory(), nats.IgnoreDeletes())
	if err != nil {
		return c.denyReaction(rec.Name, "run", itemReason(err, *prof))
	}
	defer w.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	replay := true
	for {
		select {
		case err, ok := <-w.Error():
			if ok && err != nil {
				return c.denyReaction(rec.Name, "run", "connection-lost")
			}
		case entry, ok := <-w.Updates():
			if !ok {
				return c.denyReaction(rec.Name, "run", "connection-lost")
			}
			if entry == nil {
				replay = false
				continue
			}
			ev, match, reason := c.watchEvent(req, entry, cur, replay)
			if reason != "" {
				return c.denyReaction(rec.Name, "run", reason)
			}
			if !match {
				continue
			}
			if ev.Status != rec.Watch.Status {
				if err := c.saveCursor(req, *prof, ev.Revision); err != nil {
					return c.denyReaction(rec.Name, "run", "stale-cursor")
				}
				cur.Revision = ev.Revision
				continue
			}
			out, reason := runLocalCommand(rec, timeout)
			if reason != "" {
				return c.denyReaction(rec.Name, "run", reason)
			}
			if reason := c.writeReactionResult(kv, *prof, rec, out); reason != "" {
				return c.denyReaction(rec.Name, "run", reason)
			}
			if err := c.saveCursor(req, *prof, ev.Revision); err != nil {
				return c.denyReaction(rec.Name, "run", "stale-cursor")
			}
			return c.okReaction(rec, out.ExitCode, jsonOut)
		case <-timer.C:
			return c.denyReaction(rec.Name, "run", "watch-timeout")
		}
	}
}

func runLocalCommand(rec reactionRecord, timeout time.Duration) (reactionOutput, string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, rec.Command.Cmd, rec.Command.Args...)
	cmd.Env = []string{}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		}
		return reactionOutput{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}, "command-failed"
	}
	return reactionOutput{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}, ""
}

func (c cli) writeReactionResult(kv nats.KeyValue, prof Profile, rec reactionRecord, out reactionOutput) string {
	value, err := json.Marshal(out)
	if err != nil {
		return "denied-writeback"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stored := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       rec.Write.Item,
		Status:    "resolved",
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  prof.Source,
			Writer:  "tinkalet-reaction:" + rec.Name,
		},
	}
	body, err := json.Marshal(stored)
	if err != nil {
		return "denied-writeback"
	}
	if _, err := kv.Create(rec.Write.Item, body); err != nil {
		return "denied-writeback"
	}
	return ""
}

func (c cli) loadReaction(name string) (reactionRecord, string) {
	body, err := os.ReadFile(c.reactionPath(name))
	if os.IsNotExist(err) {
		return reactionRecord{}, "reaction-not-found"
	}
	if err != nil {
		return reactionRecord{}, "reaction-invalid"
	}
	var rec reactionRecord
	if err := json.Unmarshal(body, &rec); err != nil || rec.Kind != "tinkalet.reaction.v1" || rec.Name != name {
		return reactionRecord{}, "reaction-invalid"
	}
	return rec, ""
}

func (c cli) reactionPath(name string) string {
	return filepath.Join(c.dataDir(), "reactions", name+".json")
}

func (c cli) okReaction(rec reactionRecord, code int, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(reactionRunDoc{Reaction: rec.Name, Status: "ran", Item: rec.Write.Item, ExitCode: code})
		return 0
	}
	fmt.Fprintf(c.out, "reaction %s ran %s\n", rec.Name, rec.Write.Item)
	return 0
}

func (c cli) denyReaction(name, action, reason string) int {
	fmt.Fprintf(c.errOut, "reaction %s denied %s: %s\n", name, action, reason)
	return 1
}

func (c cli) watch(args []string, daemon bool) int {
	req, ok := parseWatch(args, daemon)
	if !ok {
		return c.usage()
	}
	if req.Scope == "item" && !validItemKey(req.Target) {
		return c.usage()
	}
	if req.Scope == "prefix" && !validItemPrefix(req.Target) {
		return c.usage()
	}
	if req.Cursor != "" && !validName(req.Cursor) {
		return c.usage()
	}
	if daemon && req.Cursor == "" {
		return c.usage()
	}
	return c.itemWatch(req)
}

func parseWatch(args []string, daemon bool) (watchReq, bool) {
	req := watchReq{Limit: 1, Timeout: 30 * time.Second}
	if len(args) < 2 {
		return watchReq{}, false
	}
	req.Scope, req.Target = args[0], args[1]
	if req.Scope != "item" && req.Scope != "prefix" {
		return watchReq{}, false
	}
	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--cursor" && i+1 < len(args):
			i++
			req.Cursor = args[i]
		case strings.HasPrefix(arg, "--cursor="):
			req.Cursor = strings.TrimPrefix(arg, "--cursor=")
		case arg == "--limit" && i+1 < len(args):
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil || limit <= 0 {
				return watchReq{}, false
			}
			req.Limit = limit
		case strings.HasPrefix(arg, "--limit="):
			limit, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil || limit <= 0 {
				return watchReq{}, false
			}
			req.Limit = limit
		case arg == "--timeout" && i+1 < len(args):
			i++
			timeout, err := time.ParseDuration(args[i])
			if err != nil || timeout <= 0 {
				return watchReq{}, false
			}
			req.Timeout = timeout
		case strings.HasPrefix(arg, "--timeout="):
			timeout, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil || timeout <= 0 {
				return watchReq{}, false
			}
			req.Timeout = timeout
		default:
			return watchReq{}, false
		}
	}
	return req, true
}

func (c cli) itemWatch(req watchReq) int {
	prof, reason := c.profilePolicy()
	if reason != "" {
		return c.denyWatch(req, reason)
	}
	filters, reason := watcherWatchFilters(req, *prof)
	if filters == nil && reason == "" {
		filters, reason = participantWatchFilters(req, *prof)
	}
	if reason != "" {
		return c.denyWatch(req, reason)
	}
	cur, reason := c.loadCursor(req, *prof)
	if reason != "" {
		return c.denyWatch(req, reason)
	}
	kv, nc, reason := c.itemKVForProfile(prof)
	if reason != "" {
		return c.denyWatch(req, reason)
	}
	defer nc.Close()
	if reason := validateCursor(kv, cur); reason != "" {
		return c.denyWatch(req, reason)
	}
	var w nats.KeyWatcher
	var err error
	if filters == nil {
		w, err = kv.WatchAll(nats.IncludeHistory(), nats.IgnoreDeletes())
	} else {
		w, err = kv.WatchFiltered(filters, nats.IncludeHistory(), nats.IgnoreDeletes())
	}
	if err != nil {
		return c.denyWatch(req, itemReason(err, *prof))
	}
	defer w.Stop()

	timer := time.NewTimer(req.Timeout)
	defer timer.Stop()
	seen := 0
	replay := true
	for {
		select {
		case err, ok := <-w.Error():
			if ok && err != nil {
				return c.denyWatch(req, "connection-lost")
			}
		case entry, ok := <-w.Updates():
			if !ok {
				return c.denyWatch(req, "connection-lost")
			}
			if entry == nil {
				replay = false
				continue
			}
			ev, match, reason := c.watchEvent(req, entry, cur, replay)
			if reason != "" {
				return c.denyWatch(req, reason)
			}
			if !match {
				continue
			}
			if err := c.saveCursor(req, *prof, ev.Revision); err != nil {
				return c.denyWatch(req, "stale-cursor")
			}
			cur.Revision = ev.Revision
			c.okWatch(ev, req.JSON)
			seen++
			if seen >= req.Limit {
				return 0
			}
		case <-timer.C:
			return c.denyWatch(req, "watch-timeout")
		}
	}
}

func (c cli) loadCursor(req watchReq, prof Profile) (watchCursor, string) {
	cur := watchCursor{
		Kind:    "tinkalet.cursor.item.v1",
		Name:    req.Cursor,
		Profile: prof.Name,
		Source:  prof.Source,
		Scope:   req.Scope,
		Target:  req.Target,
	}
	if req.Cursor == "" {
		return cur, ""
	}
	body, err := os.ReadFile(c.cursorPath(req.Cursor))
	if os.IsNotExist(err) {
		return cur, ""
	}
	if err != nil {
		return cur, "stale-cursor"
	}
	if err := json.Unmarshal(body, &cur); err != nil {
		return cur, "stale-cursor"
	}
	if cur.Kind != "tinkalet.cursor.item.v1" || cur.Name != req.Cursor || cur.Profile != prof.Name || cur.Source != prof.Source || cur.Scope != req.Scope || cur.Target != req.Target {
		return cur, "stale-cursor"
	}
	return cur, ""
}

func validateCursor(kv nats.KeyValue, cur watchCursor) string {
	if cur.Revision == 0 {
		return ""
	}
	status, err := kv.Status()
	if err != nil {
		return itemReason(err, Profile{})
	}
	type streamStatus interface {
		StreamInfo() *nats.StreamInfo
	}
	if st, ok := status.(streamStatus); ok && st.StreamInfo() != nil && cur.Revision > st.StreamInfo().State.LastSeq {
		return "stale-cursor"
	}
	return ""
}

func (c cli) saveCursor(req watchReq, prof Profile, rev uint64) error {
	if req.Cursor == "" {
		return nil
	}
	cur := watchCursor{
		Kind:      "tinkalet.cursor.item.v1",
		Name:      req.Cursor,
		Profile:   prof.Name,
		Source:    prof.Source,
		Scope:     req.Scope,
		Target:    req.Target,
		Revision:  rev,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return err
	}
	return write0600(c.cursorPath(req.Cursor), append(body, '\n'))
}

func (c cli) cursorPath(name string) string {
	return filepath.Join(c.dataDir(), "cursors", name+".json")
}

func (c cli) watchEvent(req watchReq, entry nats.KeyValueEntry, cur watchCursor, replay bool) (watchEvent, bool, string) {
	if !watchMatches(req, entry.Key()) {
		return watchEvent{}, false, ""
	}
	if entry.Revision() <= cur.Revision {
		return watchEvent{}, false, ""
	}
	var rec itemStored
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != "tinkabot.item.v1" || rec.Key != entry.Key() {
		return watchEvent{}, false, "malformed-event"
	}
	src := "watch"
	if replay {
		src = "replay"
	}
	return watchEvent{
		Kind:       "tinkalet.itemEvent.v1",
		Cursor:     req.Cursor,
		Scope:      req.Scope,
		Key:        rec.Key,
		Status:     rec.Status,
		Value:      rec.Value,
		Revision:   entry.Revision(),
		Source:     src,
		ObservedAt: time.Now().UTC().Format(time.RFC3339),
	}, true, ""
}

func watchMatches(req watchReq, key string) bool {
	switch req.Scope {
	case "item":
		return key == req.Target
	case "prefix":
		return strings.HasPrefix(key, req.Target)
	default:
		return false
	}
}

func participantWatchFilters(req watchReq, prof Profile) ([]string, string) {
	if prof.Trust != "app-participant" {
		return nil, ""
	}
	if !validSubjectToken(prof.AppID) || !validSubjectToken(prof.ParticipantID) {
		return nil, "profile-not-participant"
	}
	state := "apps." + prof.AppID + ".state"
	actions := "apps." + prof.AppID + ".participants." + prof.ParticipantID + ".actions"
	switch req.Scope {
	case "item":
		if !validItemKey(req.Target) {
			return nil, "denied-scope"
		}
		if strings.HasPrefix(req.Target, state+".") || strings.HasPrefix(req.Target, actions+".") {
			return []string{req.Target}, ""
		}
	case "prefix":
		if !validItemPrefix(req.Target) {
			return nil, "denied-scope"
		}
		switch strings.TrimSuffix(req.Target, ".") {
		case state:
			return []string{state + ".>"}, ""
		case actions:
			return []string{actions + ".>"}, ""
		}
	}
	return nil, "denied-scope"
}

func watcherWatchFilters(req watchReq, prof Profile) ([]string, string) {
	if prof.Trust != "item-watcher" {
		return nil, ""
	}
	if !validWatcherScope(prof.WatchScope, prof.WatchTarget) {
		return nil, "denied-scope"
	}
	if prof.WatchScope == "item" {
		if req.Scope == "item" && req.Target == prof.WatchTarget {
			return []string{req.Target}, ""
		}
		return nil, "denied-scope"
	}
	prefix := strings.TrimSuffix(prof.WatchTarget, ".")
	switch req.Scope {
	case "item":
		if req.Target == prefix || strings.HasPrefix(req.Target, prefix+".") {
			return []string{req.Target}, ""
		}
	case "prefix":
		if strings.TrimSuffix(req.Target, ".") == prefix {
			return []string{prefix + ".>"}, ""
		}
	}
	return nil, "denied-scope"
}

func validWatcherScope(scope, target string) bool {
	switch scope {
	case "item":
		return validItemKey(target)
	case "prefix":
		return validItemPrefix(target)
	default:
		return false
	}
}

func (c cli) okWatch(ev watchEvent, jsonOut bool) {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(ev)
		return
	}
	fmt.Fprintf(c.out, "item %s %s rev %d\n", ev.Key, ev.Status, ev.Revision)
}

func (c cli) denyWatch(req watchReq, reason string) int {
	fmt.Fprintf(c.errOut, "watch %s denied %s: %s\n", req.Target, req.Scope, reason)
	return 1
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

func parseActionSubmit(args []string) (actionSubmitReq, bool) {
	var req actionSubmitReq
	if len(args) == 0 {
		return req, false
	}
	req.ActionID = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--state" && i+1 < len(args):
			i++
			req.StateKey = args[i]
		case strings.HasPrefix(arg, "--state="):
			req.StateKey = strings.TrimPrefix(arg, "--state=")
		case arg == "--base-revision" && i+1 < len(args):
			i++
			rev, err := strconv.ParseUint(args[i], 10, 64)
			if err != nil {
				return actionSubmitReq{}, false
			}
			req.BaseRevision = rev
		case strings.HasPrefix(arg, "--base-revision="):
			rev, err := strconv.ParseUint(strings.TrimPrefix(arg, "--base-revision="), 10, 64)
			if err != nil {
				return actionSubmitReq{}, false
			}
			req.BaseRevision = rev
		case arg == "--value" && i+1 < len(args):
			i++
			req.ValueRaw = args[i]
		case strings.HasPrefix(arg, "--value="):
			req.ValueRaw = strings.TrimPrefix(arg, "--value=")
		case strings.HasPrefix(arg, "-"):
			return actionSubmitReq{}, false
		default:
			return actionSubmitReq{}, false
		}
	}
	return req, req.ActionID != "" && req.StateKey != "" && req.BaseRevision > 0
}

func parseActionApply(args []string) (actionApplyReq, bool) {
	var req actionApplyReq
	if len(args) == 0 {
		return req, false
	}
	req.ActionKey = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--value" && i+1 < len(args):
			i++
			req.ValueRaw = args[i]
		case strings.HasPrefix(arg, "--value="):
			req.ValueRaw = strings.TrimPrefix(arg, "--value=")
		default:
			return actionApplyReq{}, false
		}
	}
	return req, req.ActionKey != "" && req.ValueRaw != ""
}

func parseActionReject(args []string) (actionRejectReq, bool) {
	var req actionRejectReq
	if len(args) == 0 {
		return req, false
	}
	req.ActionKey = args[0]
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--reason" && i+1 < len(args):
			i++
			req.Reason = args[i]
		case strings.HasPrefix(arg, "--reason="):
			req.Reason = strings.TrimPrefix(arg, "--reason=")
		default:
			return actionRejectReq{}, false
		}
	}
	return req, req.ActionKey != "" && req.Reason != ""
}

func (c cli) itemCreate(key, status string, value json.RawMessage, jsonOut bool) int {
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyItem(key, "create", reason)
	} else if restrictedProfile(*prof) {
		return c.denyItem(key, "create", "denied-scope")
	}
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

func (c cli) actionSubmit(req actionSubmitReq, value json.RawMessage) int {
	prof, nc, reason := c.profileConnFor(c.defaultProfile())
	if reason != "" {
		return c.denyAction(req.ActionID, "submit", reason)
	}
	defer nc.Close()
	if prof.Trust == "item-watcher" {
		return c.denyAction(req.ActionID, "submit", "denied-scope")
	}
	if prof.Trust != "app-participant" || !validSubjectToken(prof.AppID) || !validSubjectToken(prof.ParticipantID) {
		return c.denyAction(req.ActionID, "submit", "profile-not-participant")
	}
	body, err := json.Marshal(actionWireReq{
		ActionID:     req.ActionID,
		StateKey:     req.StateKey,
		BaseRevision: req.BaseRevision,
		Value:        value,
	})
	if err != nil {
		return c.denyAction(req.ActionID, "submit", "malformed-value")
	}
	reply, err := nc.Request(actionSubject(*prof), body, 5*time.Second)
	if err != nil {
		return c.denyAction(req.ActionID, "submit", authReason(err, *prof))
	}
	var resp actionWireResp
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return c.denyAction(req.ActionID, "submit", "malformed-response")
	}
	if resp.Status != "accepted" {
		reason := resp.Reason
		if reason == "" {
			reason = "action-denied"
		}
		return c.denyAction(req.ActionID, "submit", reason)
	}
	if resp.Item == nil {
		return c.denyAction(req.ActionID, "submit", "malformed-response")
	}
	return c.okAction(req.ActionID, *resp.Item, req.JSON)
}

func (c cli) actionApply(req actionApplyReq, value json.RawMessage) int {
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	} else if restrictedProfile(*prof) {
		return c.denyAction(req.ActionKey, "apply", "denied-scope")
	}
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	defer nc.Close()

	receiptKey := req.ActionKey + ".receipt"
	if _, reason := readItemAs(kv, receiptKey, *prof); reason == "" {
		return c.denyAction(req.ActionKey, "apply", "duplicate-action")
	} else if reason != "item-not-found" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}

	action, reason := readItemAs(kv, req.ActionKey, *prof)
	if reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	if action.Status != "pending" {
		return c.denyAction(req.ActionKey, "apply", "duplicate-action")
	}
	var act appActionStored
	if err := json.Unmarshal(action.Value, &act); err != nil || !validAppAction(act, req.ActionKey) {
		return c.denyAction(req.ActionKey, "apply", "malformed-action")
	}
	state, reason := readItemAs(kv, act.StateKey, *prof)
	if reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	if state.Revision != act.BaseRevision {
		return c.denyAction(req.ActionKey, "apply", "stale-revision")
	}
	receipt, reason := createActionReceipt(kv, *prof, req.ActionKey, receiptKey, act.StateKey, action.Revision, 0, "pending", "applying", "")
	if reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	stateRev, reason := updateActionState(kv, *prof, act, req.ActionKey, state, value)
	if reason != "" {
		if reason == "stale-revision" {
			latest, latestReason := readItemAs(kv, act.StateKey, *prof)
			if latestReason == "" {
				_, _ = updateActionReceipt(kv, *prof, receipt, req.ActionKey, act.StateKey, action.Revision, latest.Revision, "denied", "rejected", "stale-revision")
			}
		}
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	receipt, reason = updateActionReceipt(kv, *prof, receipt, req.ActionKey, act.StateKey, action.Revision, stateRev, "resolved", "applied", "")
	if reason != "" {
		return c.denyAction(req.ActionKey, "apply", reason)
	}
	return c.okActionApply(req.ActionKey, receipt, req.JSON)
}

func (c cli) actionReject(req actionRejectReq) int {
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyAction(req.ActionKey, "reject", reason)
	} else if restrictedProfile(*prof) {
		return c.denyAction(req.ActionKey, "reject", "denied-scope")
	}
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyAction(req.ActionKey, "reject", reason)
	}
	defer nc.Close()

	receiptKey := req.ActionKey + ".receipt"
	if _, reason := readItemAs(kv, receiptKey, *prof); reason == "" {
		return c.denyAction(req.ActionKey, "reject", "duplicate-action")
	} else if reason != "item-not-found" {
		return c.denyAction(req.ActionKey, "reject", reason)
	}

	action, reason := readItemAs(kv, req.ActionKey, *prof)
	if reason != "" {
		return c.denyAction(req.ActionKey, "reject", reason)
	}
	if action.Status != "pending" {
		return c.denyAction(req.ActionKey, "reject", "duplicate-action")
	}
	var act appActionStored
	if err := json.Unmarshal(action.Value, &act); err != nil || !validAppAction(act, req.ActionKey) {
		return c.denyAction(req.ActionKey, "reject", "malformed-action")
	}
	state, reason := readItemAs(kv, act.StateKey, *prof)
	if reason != "" {
		return c.denyAction(req.ActionKey, "reject", reason)
	}
	if state.Revision != act.BaseRevision {
		return c.denyAction(req.ActionKey, "reject", "stale-revision")
	}
	receipt, reason := createActionReceipt(kv, *prof, req.ActionKey, receiptKey, act.StateKey, action.Revision, state.Revision, "denied", "rejected", req.Reason)
	if reason != "" {
		return c.denyAction(req.ActionKey, "reject", reason)
	}
	return c.okActionReject(req.ActionKey, receipt, req.JSON)
}

func (c cli) itemGet(key string, jsonOut bool) int {
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyItem(key, "get", reason)
	} else if prof.Trust == "item-watcher" {
		return c.denyItem(key, "get", "denied-scope")
	}
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "get", reason)
	}
	defer nc.Close()
	item, reason := readItemAs(kv, key, *prof)
	if reason != "" {
		return c.denyItem(key, "get", reason)
	}
	return c.okItem(item, jsonOut)
}

func (c cli) itemResolve(key string, value json.RawMessage, rev uint64, revSet, jsonOut bool) int {
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyItem(key, "resolve", reason)
	} else if restrictedProfile(*prof) {
		return c.denyItem(key, "resolve", "denied-scope")
	}
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "resolve", reason)
	}
	defer nc.Close()
	current, reason := readItemAs(kv, key, *prof)
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
	if prof, reason := c.profilePolicy(); reason != "" {
		return c.denyItem(key, "wait", reason)
	} else if prof.Trust == "item-watcher" {
		return c.denyItem(key, "wait", "denied-scope")
	}
	prof, kv, nc, reason := c.itemKV()
	if reason != "" {
		return c.denyItem(key, "wait", reason)
	}
	defer nc.Close()
	deadline := time.Now().Add(timeout)
	for {
		item, reason := readItemAs(kv, key, *prof)
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
	name := c.defaultProfile()
	if name == "" {
		return nil, nil, nil, "profile-not-found"
	}
	return c.itemKVFor(name)
}

func (c cli) itemKVFor(name string) (*Profile, nats.KeyValue, *nats.Conn, string) {
	prof, nc, reason := c.profileConnFor(name)
	if reason != "" {
		return prof, nil, nil, reason
	}
	kv, nc, reason := c.itemKVForConn(prof, nc)
	if reason != "" {
		return prof, nil, nil, reason
	}
	return prof, kv, nc, ""
}

func (c cli) itemKVForProfile(prof *Profile) (nats.KeyValue, *nats.Conn, string) {
	nc, reason := c.connectProfile(prof)
	if reason != "" {
		return nil, nil, reason
	}
	return c.itemKVForConn(prof, nc)
}

func (c cli) itemKVForConn(prof *Profile, nc *nats.Conn) (nats.KeyValue, *nats.Conn, string) {
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, "connection-failed"
	}
	kv, err := js.KeyValue(itemBucket)
	if err != nil {
		nc.Close()
		return nil, nil, itemReason(err, *prof)
	}
	return kv, nc, ""
}

func (c cli) profilePolicy() (*Profile, string) {
	name := c.defaultProfile()
	if name == "" {
		return nil, "profile-not-found"
	}
	profiles, _ := c.loadProfiles()
	prof := find(profiles, name)
	if prof == nil {
		return nil, "profile-not-found"
	}
	if c.deniedNeighbor(*prof) {
		return prof, "denied-neighbor"
	}
	if c.revokedProfile(*prof) {
		return prof, "revoked-credentials"
	}
	return prof, ""
}

func (c cli) profileConnFor(name string) (*Profile, *nats.Conn, string) {
	if name == "" {
		return nil, nil, "profile-not-found"
	}
	profiles, _ := c.loadProfiles()
	prof := find(profiles, name)
	if prof == nil {
		return nil, nil, "profile-not-found"
	}
	nc, reason := c.connectProfile(prof)
	if reason != "" {
		return prof, nil, reason
	}
	return prof, nc, ""
}

func (c cli) connectProfile(prof *Profile) (*nats.Conn, string) {
	if prof == nil {
		return nil, "profile-not-found"
	}
	creds := filepath.Join(c.dataDir(), filepath.FromSlash(prof.CredentialRef))
	if _, err := os.Stat(creds); err != nil {
		return nil, "stale-credentials"
	}
	if c.deniedNeighbor(*prof) {
		return nil, "denied-neighbor"
	}
	if c.revokedProfile(*prof) {
		return nil, "revoked-credentials"
	}
	nc, err := nats.Connect(prof.Server, nats.UserCredentials(creds), nats.NoReconnect(), nats.Timeout(2*time.Second), nats.ErrorHandler(func(*nats.Conn, *nats.Subscription, error) {}))
	if err != nil {
		return nil, authReason(err, *prof)
	}
	return nc, ""
}

func readItem(kv nats.KeyValue, key string) (itemView, string) {
	return readItemAs(kv, key, Profile{})
}

func readItemAs(kv nats.KeyValue, key string, prof Profile) (itemView, string) {
	entry, err := kv.Get(key)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			return itemView{}, "item-not-found"
		}
		return itemView{}, itemReason(err, prof)
	}
	var rec itemStored
	if err := json.Unmarshal(entry.Value(), &rec); err != nil || rec.Kind != "tinkabot.item.v1" || rec.Key != key {
		return itemView{}, "malformed-item"
	}
	return viewItem(rec, entry.Revision()), ""
}

func actionKey(act appActionStored) string {
	return "apps." + act.AppID + ".participants." + act.ParticipantID + ".actions." + act.ActionID
}

func validAppAction(act appActionStored, key string) bool {
	return act.Kind == "tinkabot.appAction.v1" &&
		validSubjectToken(act.AppID) &&
		validSubjectToken(act.ParticipantID) &&
		validSubjectToken(act.ActionID) &&
		validItemKey(act.StateKey) &&
		strings.HasPrefix(act.StateKey, "apps."+act.AppID+".state.") &&
		act.BaseRevision != 0 &&
		actionKey(act) == key
}

func updateActionState(kv nats.KeyValue, prof Profile, act appActionStored, actionKey string, state itemView, value json.RawMessage) (uint64, string) {
	now := time.Now().UTC().Format(time.RFC3339)
	rec := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       act.StateKey,
		Status:    "resolved",
		Value:     value,
		CreatedAt: state.CreatedAt,
		UpdatedAt: now,
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  "app-action:" + actionKey,
			Writer:  "tinkalet-action-reducer",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return 0, "malformed-value"
	}
	rev, err := kv.Update(act.StateKey, body, act.BaseRevision)
	if err != nil {
		return 0, itemReason(err, prof)
	}
	return rev, ""
}

func createActionReceipt(kv nats.KeyValue, prof Profile, actionKey, receiptKey, stateKey string, actionRev, stateRev uint64, status, outcome, reason string) (itemView, string) {
	val, err := json.Marshal(actionReceiptStored{
		Kind:           "tinkabot.appActionReceipt.v1",
		ActionKey:      actionKey,
		StateKey:       stateKey,
		ActionRevision: actionRev,
		StateRevision:  stateRev,
		Outcome:        outcome,
		Reason:         reason,
	})
	if err != nil {
		return itemView{}, "malformed-value"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rec := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       receiptKey,
		Status:    status,
		Value:     val,
		CreatedAt: now,
		UpdatedAt: now,
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  "app-action:" + actionKey,
			Writer:  "tinkalet-action-reducer",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return itemView{}, "malformed-value"
	}
	rev, err := kv.Create(receiptKey, body)
	if err != nil {
		if errors.Is(err, nats.ErrKeyExists) {
			return itemView{}, "duplicate-action"
		}
		return itemView{}, itemReason(err, prof)
	}
	return viewItem(rec, rev), ""
}

func updateActionReceipt(kv nats.KeyValue, prof Profile, receipt itemView, actionKey, stateKey string, actionRev, stateRev uint64, status, outcome, reason string) (itemView, string) {
	val, err := json.Marshal(actionReceiptStored{
		Kind:           "tinkabot.appActionReceipt.v1",
		ActionKey:      actionKey,
		StateKey:       stateKey,
		ActionRevision: actionRev,
		StateRevision:  stateRev,
		Outcome:        outcome,
		Reason:         reason,
	})
	if err != nil {
		return itemView{}, "malformed-value"
	}
	rec := itemStored{
		Kind:      "tinkabot.item.v1",
		Key:       receipt.Key,
		Status:    status,
		Value:     val,
		CreatedAt: receipt.CreatedAt,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Provenance: itemProvenance{
			Profile: prof.Name,
			Source:  "app-action:" + actionKey,
			Writer:  "tinkalet-action-reducer",
		},
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return itemView{}, "malformed-value"
	}
	rev, err := kv.Update(receipt.Key, body, receipt.Revision)
	if err != nil {
		return itemView{}, itemReason(err, prof)
	}
	return viewItem(rec, rev), ""
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
		if restrictedProfile(prof) {
			return "denied-scope"
		}
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

func actionSubject(prof Profile) string {
	return "tb.app." + prof.AppID + ".participants." + prof.ParticipantID + ".action"
}

func (c cli) okAction(actionID string, item itemView, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(item)
		return 0
	}
	fmt.Fprintf(c.out, "action %s submitted rev %d\n", actionID, item.Revision)
	return 0
}

func (c cli) okActionApply(actionKey string, item itemView, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(item)
		return 0
	}
	fmt.Fprintf(c.out, "action %s applied rev %d\n", actionKey, item.Revision)
	return 0
}

func (c cli) okActionReject(actionKey string, item itemView, jsonOut bool) int {
	if jsonOut {
		_ = json.NewEncoder(c.out).Encode(item)
		return 0
	}
	fmt.Fprintf(c.out, "action %s rejected rev %d\n", actionKey, item.Revision)
	return 0
}

func (c cli) denyAction(actionID, action, reason string) int {
	fmt.Fprintf(c.errOut, "action %s denied %s: %s\n", actionID, action, reason)
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
	return validItemPath(key)
}

func validItemPrefix(prefix string) bool {
	if prefix == "" || strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "//") {
		return false
	}
	return validItemPath(prefix)
}

func validItemPath(path string) bool {
	for _, r := range path {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == '/':
		default:
			return false
		}
	}
	return !strings.Contains(path, "..")
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

func (c cli) revokedProfile(prof Profile) bool {
	if prof.Trust != "app-participant" && prof.Trust != "item-watcher" {
		return false
	}
	const prefix = "local-store:"
	if !strings.HasPrefix(prof.Source, prefix) {
		return false
	}
	body, err := os.ReadFile(filepath.Join(strings.TrimPrefix(prof.Source, prefix), "local-profile.json"))
	if err != nil {
		return false
	}
	var desc descriptor
	return json.Unmarshal(body, &desc) == nil && desc.Status == "revoked"
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
		if restrictedProfile(prof) {
			return "denied-scope"
		}
		if strings.HasPrefix(prof.Source, "local-store:") {
			return "revoked-credentials"
		}
		return "denied-trigger"
	default:
		return "connection-failed"
	}
}

func restrictedProfile(prof Profile) bool {
	return prof.Trust == "app-participant" || prof.Trust == "item-watcher"
}

func subjectFor(intent string) string {
	parts := strings.Split(intent, ".")
	if len(parts) != 3 || parts[0] != "bundle" || !validSubjectToken(parts[1]) || !validSubjectToken(parts[2]) {
		return ""
	}
	return "tb.bundle." + parts[1] + "." + parts[2]
}

func validSubjectToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
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
