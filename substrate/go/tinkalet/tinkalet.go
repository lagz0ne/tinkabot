package tinkalet

import (
	"encoding/json"
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
