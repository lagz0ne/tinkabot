package embednats

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	jwt "github.com/nats-io/jwt/v2"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

// Typed owners for the six operator/JWT failure families.
const (
	OperatorKeyFailed    Kind = "OperatorKeyFailed"
	AccountCompileFailed Kind = "AccountCompileFailed"
	JWTMintFailed        Kind = "JWTMintFailed"
	AccountUpdateFailed  Kind = "AccountUpdateFailed"
	RevocationFailed     Kind = "RevocationFailed"
	ProvenanceLost       Kind = "ProvenanceLost"
)

const (
	// OverbroadMint is returned when a mint request carries a session-subtree
	// wildcard (tb.session.-prefixed wildcard pattern) — denied by the
	// subject-breadth check in MintUser.
	OverbroadMint Kind = "OverbroadMint"
	// SteerAfterRevoke is returned by ApplySteerAfterRevoke when the steerer's
	// credential has been revoked in the TB_APP account. The apply-time
	// revocation re-check owns this failure family.
	SteerAfterRevoke Kind = "SteerAfterRevoke"
)

// Control-plane / app-plane account split.
const (
	ControlAccount = "TB_CONTROL"
	AppAccount     = "TB_APP"
)

// OperatorPosture reports the live operator/JWT authority: the substrate-held
// master operator identity and the key file it is reloaded from.
type OperatorPosture struct {
	Enabled   bool
	PublicKey string
	KeyFile   string
}

// UserCreds is a minted principal credential. Lease is decoded back from the
// signed user JWT, not echoed from config, so it proves the lease vocabulary
// survived into the token.
type UserCreds struct {
	UserPub string
	File    []byte
	Lease   core.Capability
}

// leaseTag prefixes the JWT tag that carries the lease vocabulary
// (hex-encoded JSON: tags are lowercased by the jwt library, hex is safe).
const leaseTag = "tblease:"

// operatorKeyFile is the substrate-held master operator key inside StoreDir.
const operatorKeyFile = "operator.nk"

type operator struct {
	mu       sync.Mutex
	pub      string
	keyFile  string
	trusted  []*jwt.OperatorClaims
	resolver *natsserver.MemAccResolver
	sysPub   string
	accounts map[string]*opAccount

	kp        nkeys.KeyPair
	probeJWT  string
	probeSeed string
}

type opAccount struct {
	pub      string
	kp       nkeys.KeyPair
	scopePub string
	scopeKP  nkeys.KeyPair
	claims   *jwt.AccountClaims
	minted   map[string]bool
}

// newOperator loads (or generates, on first start) the master operator key
// from the store dir and compiles the control/app account split into the
// in-memory account resolver. Account identities are ephemeral per process;
// the operator identity is the persistent authority.
func newOperator(dir string) (*operator, error) {
	kp, keyFile, err := operatorKey(dir)
	if err != nil {
		return nil, err
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return nil, fail(OperatorKeyFailed, "Start", "operator public key could not be derived", nil, err)
	}
	oc := jwt.NewOperatorClaims(pub)
	oc.Name = "tinkabot"
	token, err := oc.Encode(kp)
	if err != nil {
		return nil, fail(OperatorKeyFailed, "Start", "operator claims could not be signed", nil, err)
	}
	trusted, err := jwt.DecodeOperatorClaims(token)
	if err != nil {
		return nil, fail(OperatorKeyFailed, "Start", "operator claims round-trip failed", nil, err)
	}

	op := &operator{
		pub:      pub,
		keyFile:  keyFile,
		trusted:  []*jwt.OperatorClaims{trusted},
		resolver: &natsserver.MemAccResolver{},
		accounts: map[string]*opAccount{},
		kp:       kp,
	}
	if op.sysPub, err = op.systemAccount(); err != nil {
		return nil, err
	}
	for _, name := range []string{ControlAccount, AppAccount} {
		acc, err := op.newAccount(name)
		if err != nil {
			return nil, err
		}
		op.accounts[name] = acc
	}
	if err := op.mintProbe(); err != nil {
		return nil, err
	}
	return op, nil
}

// operatorKey generates the master key into the store dir at first start and
// reloads it afterwards. Corrupt material fails typed instead of silently
// regenerating a new authority.
func operatorKey(dir string) (nkeys.KeyPair, string, error) {
	path := filepath.Join(dir, operatorKeyFile)
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		kp, err := nkeys.CreateOperator()
		if err != nil {
			return nil, "", fail(OperatorKeyFailed, "Start", "operator key could not be generated", nil, err)
		}
		seed, err := kp.Seed()
		if err != nil {
			return nil, "", fail(OperatorKeyFailed, "Start", "operator seed could not be derived", nil, err)
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, "", fail(OperatorKeyFailed, "Start", "store dir could not be created", map[string]string{"dir": dir}, err)
		}
		if err := os.WriteFile(path, seed, 0o600); err != nil {
			return nil, "", fail(OperatorKeyFailed, "Start", "operator key could not be persisted", map[string]string{"file": path}, err)
		}
		return kp, path, nil
	}
	if err != nil {
		return nil, "", fail(OperatorKeyFailed, "Start", "operator key material unreadable", map[string]string{"file": path}, err)
	}
	kp, err := nkeys.FromSeed([]byte(strings.TrimSpace(string(raw))))
	if err != nil {
		return nil, "", fail(OperatorKeyFailed, "Start", "operator key material is corrupt", map[string]string{"file": path}, err)
	}
	pub, err := kp.PublicKey()
	if err != nil || !nkeys.IsValidPublicOperatorKey(pub) {
		return nil, "", fail(OperatorKeyFailed, "Start", "operator key material is not an operator key", map[string]string{"file": path}, err)
	}
	return kp, path, nil
}

func (o *operator) systemAccount() (string, error) {
	kp, err := nkeys.CreateAccount()
	if err != nil {
		return "", fail(AccountCompileFailed, "Start", "system account key could not be created", nil, err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return "", fail(AccountCompileFailed, "Start", "system account key could not be derived", nil, err)
	}
	claims := jwt.NewAccountClaims(pub)
	claims.Name = "TB_SYS"
	token, err := claims.Encode(o.kp)
	if err != nil {
		return "", fail(AccountCompileFailed, "Start", "system account claims could not be compiled", nil, err)
	}
	if err := o.resolver.Store(pub, token); err != nil {
		return "", fail(AccountUpdateFailed, "Start", "system account claims could not be stored", nil, err)
	}
	return pub, nil
}

// newAccount compiles one account of the split: a root key that signs users
// carrying their own permissions, and a scoped signing key whose template is
// the account default for users minted without permissions. JetStream is
// enabled so the substrate-owned client can probe it.
func (o *operator) newAccount(name string) (*opAccount, error) {
	kp, err := nkeys.CreateAccount()
	if err != nil {
		return nil, fail(AccountCompileFailed, "Start", "account key could not be created", map[string]string{"account": name}, err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return nil, fail(AccountCompileFailed, "Start", "account key could not be derived", map[string]string{"account": name}, err)
	}
	scopeKP, err := nkeys.CreateAccount()
	if err != nil {
		return nil, fail(AccountCompileFailed, "Start", "account signing key could not be created", map[string]string{"account": name}, err)
	}
	scopePub, err := scopeKP.PublicKey()
	if err != nil {
		return nil, fail(AccountCompileFailed, "Start", "account signing key could not be derived", map[string]string{"account": name}, err)
	}

	claims := jwt.NewAccountClaims(pub)
	claims.Name = name
	claims.Limits.JetStreamLimits = jwt.JetStreamLimits{
		MemoryStorage: jwt.NoLimit,
		DiskStorage:   jwt.NoLimit,
		Streams:       jwt.NoLimit,
		Consumer:      jwt.NoLimit,
	}
	acc := &opAccount{pub: pub, kp: kp, scopePub: scopePub, scopeKP: scopeKP, claims: claims, minted: map[string]bool{}}
	// Deny-by-default: a permissionless mint holds no authority until an
	// explicit UpdateAccountPerms sets the account default.
	acc.setScope(core.Permissions{
		Publish:   core.PermList{Deny: []string{">"}},
		Subscribe: core.PermList{Deny: []string{">"}},
	})
	token, err := claims.Encode(o.kp)
	if err != nil {
		return nil, fail(AccountCompileFailed, "Start", "account claims could not be compiled", map[string]string{"account": name}, err)
	}
	if err := o.resolver.Store(pub, token); err != nil {
		return nil, fail(AccountUpdateFailed, "Start", "account claims could not be stored", map[string]string{"account": name}, err)
	}
	return acc, nil
}

// setScope replaces the account-default permission template carried by the
// scoped signing key.
func (a *opAccount) setScope(perms core.Permissions) {
	scope := jwt.NewUserScope()
	scope.Key = a.scopePub
	scope.Template.Permissions = jwtPerms(perms)
	a.claims.SigningKeys.AddScopedSigner(scope)
}

// mintProbe issues the substrate-owned client credential in the control
// account, holding only the JetStream probe surface.
func (o *operator) mintProbe() error {
	ctl := o.accounts[ControlAccount]
	kp, err := nkeys.CreateUser()
	if err != nil {
		return fail(JWTMintFailed, "Start", "probe user key could not be created", nil, err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return fail(JWTMintFailed, "Start", "probe user key could not be derived", nil, err)
	}
	uc := jwt.NewUserClaims(pub)
	uc.Name = "_tb_probe"
	uc.Permissions.Pub.Allow.Add("$JS.API.INFO")
	uc.Permissions.Sub.Allow.Add("_INBOX.>")
	token, err := uc.Encode(ctl.kp)
	if err != nil {
		return fail(JWTMintFailed, "Start", "probe user JWT could not be minted", nil, err)
	}
	seed, err := kp.Seed()
	if err != nil {
		return fail(JWTMintFailed, "Start", "probe user seed could not be derived", nil, err)
	}
	o.probeJWT, o.probeSeed = token, string(seed)
	return nil
}

func (o *operator) posture() OperatorPosture {
	if o == nil {
		return OperatorPosture{}
	}
	return OperatorPosture{Enabled: true, PublicKey: o.pub, KeyFile: o.keyFile}
}

// isSessionSubtreeWildcard reports whether a subject is a tb.session.-prefixed
// wildcard pattern.  Catches both terminal wildcards (tb.session.abc.*)
// and infix wildcards (tb.session.*.ingest) — any .* or .> token anywhere in
// the tb.session. subtree is considered overbroad.  Legitimate wildcards like
// _INBOX.>, $KV.*.>, and $JS.API.* are not tb.session.-prefixed and are
// unaffected.
func isSessionSubtreeWildcard(subj string) bool {
	if !strings.HasPrefix(subj, "tb.session.") {
		return false
	}
	return strings.Contains(subj, ".*") || strings.Contains(subj, ".>")
}

// MintUser issues a short-lived user JWT in the given account, carrying the
// lease vocabulary of the capability. Users with explicit permissions are
// signed by the account root key; users without ride the account-default
// scoped signing key, so live account pushes supersede their permissions.
func (r *Runtime) MintUser(account string, auth core.Auth, ttl time.Duration) (UserCreds, error) {
	if r == nil || r.op == nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "operator mode is not enabled", nil, nil)
	}
	r.op.mu.Lock()
	defer r.op.mu.Unlock()
	acc := r.op.accounts[account]
	if acc == nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "unknown account", map[string]string{"account": account}, nil)
	}
	cap := auth.Capability
	switch {
	case strings.TrimSpace(auth.User) == "" || strings.TrimSpace(cap.LeaseID) == "":
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "principal and lease are required", cap.Details(), nil)
	case cap.LeaseStatus != "active":
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "lease is not active", cap.Details(), nil)
	case cap.PrincipalID != "" && cap.PrincipalID != auth.User:
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "lease principal does not match user", cap.Details(), nil)
	case auth.Permissions.AllowResponses.Max > 0 && auth.Permissions.AllowResponses.ExpiresMs <= 0:
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "bounded response TTL is required", cap.Details(), nil)
	case auth.Permissions.AllowResponses.Max <= 0 && auth.Permissions.AllowResponses.ExpiresMs > 0:
		// A degenerate bound grants nothing; minting it as "non-empty" perms
		// would produce an empty (allow-all) root-key-signed JWT.
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "response bound requires a positive max", cap.Details(), nil)
	case ttl <= 0:
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "bounded credential TTL is required", cap.Details(), nil)
	case strings.TrimSpace(cap.PrincipalID) == "" || strings.TrimSpace(cap.SessionID) == "" || strings.TrimSpace(cap.CapabilityID) == "":
		return UserCreds{}, fail(ProvenanceLost, "MintUser", "lease provenance is incomplete", cap.Details(), nil)
	}
	for _, subj := range append(append(append(auth.Permissions.Publish.Allow, auth.Permissions.Subscribe.Allow...), auth.Permissions.Publish.Deny...), auth.Permissions.Subscribe.Deny...) {
		if isSessionSubtreeWildcard(subj) {
			return UserCreds{}, fail(OverbroadMint, "MintUser", "session-subtree wildcard grant denied", map[string]string{"subject": subj}, nil)
		}
	}

	kp, err := nkeys.CreateUser()
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "user key could not be created", nil, err)
	}
	pub, err := kp.PublicKey()
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "user key could not be derived", nil, err)
	}
	payload, err := json.Marshal(cap)
	if err != nil {
		return UserCreds{}, fail(ProvenanceLost, "MintUser", "lease vocabulary could not be encoded", cap.Details(), err)
	}
	uc := jwt.NewUserClaims(pub)
	signer := acc.kp
	if emptyPerms(auth.Permissions) {
		uc.SetScoped(true)
		uc.IssuerAccount = acc.pub
		signer = acc.scopeKP
	} else {
		uc.Permissions = jwtPerms(auth.Permissions)
	}
	uc.Name = auth.User
	uc.Tags.Add(leaseTag + hex.EncodeToString(payload))
	uc.Expires = time.Now().Add(ttl).Unix()
	token, err := uc.Encode(signer)
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "user JWT could not be minted", nil, err)
	}
	minted, err := jwt.DecodeUserClaims(token)
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "minted user JWT does not decode", nil, err)
	}
	lease, err := leaseFromClaims(minted)
	if err != nil {
		return UserCreds{}, fail(ProvenanceLost, "MintUser", "lease vocabulary did not survive into the JWT", nil, err)
	}
	seed, err := kp.Seed()
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "user seed could not be derived", nil, err)
	}
	file, err := jwt.FormatUserConfig(token, seed)
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintUser", "user creds could not be formatted", nil, err)
	}
	acc.minted[pub] = true
	return UserCreds{UserPub: pub, File: file, Lease: lease}, nil
}

// ConnectCreds connects a minted credential through the declared exposure
// posture. Malformed and expired creds are denied at this boundary before any
// dial; the embedded server stays the authority for signature, account, and
// revocation checks on the wire.
func (r *Runtime) ConnectCreds(ctx context.Context, creds []byte) (*nats.Conn, error) {
	if r == nil || r.op == nil {
		return nil, fail(ClientConnectFailed, "Connect", "operator mode is not enabled", nil, nil)
	}
	token, err := jwt.ParseDecoratedJWT(creds)
	if err != nil {
		return nil, fail(ClientConnectFailed, "Connect", "user creds are malformed", nil, err)
	}
	uc, err := jwt.DecodeUserClaims(token)
	if err != nil {
		return nil, fail(ClientConnectFailed, "Connect", "user JWT is malformed", nil, err)
	}
	// Attribute lease-bearing denials with the lease the token still carries.
	var details map[string]string
	if lease, lerr := leaseFromClaims(uc); lerr == nil {
		details = lease.Details()
	}
	// JWT expiry is second-granular and the server enforces it with an async
	// post-handshake timer; deny expired creds deterministically here.
	if uc.Expires > 0 && time.Now().Unix() >= uc.Expires {
		return nil, fail(ClientConnectFailed, "Connect", "user JWT is expired", details, nil)
	}
	kp, err := jwt.ParseDecoratedUserNKey(creds)
	if err != nil {
		return nil, fail(ClientConnectFailed, "Connect", "user creds carry no signing key", nil, err)
	}
	seed, err := kp.Seed()
	if err != nil {
		return nil, fail(ClientConnectFailed, "Connect", "user seed could not be derived", nil, err)
	}
	nc, err := r.dial(ctx, []nats.Option{
		nats.UserJWTAndSeed(token, string(seed)),
		// Live account pushes kick scoped connections so they re-authenticate
		// under the superseding claims; reconnect fast so the same client
		// connection carries on.
		nats.ReconnectWait(25 * time.Millisecond),
		nats.ReconnectJitter(0, 0),
		nats.MaxReconnects(-1),
	})
	if err != nil {
		// Server-side denials (revocation, superseded claims) flow back through
		// dial; re-attribute them with the lease, keeping the typed kind.
		var ae *Error
		if details != nil && errors.As(err, &ae) {
			return nil, fail(ae.Kind, "Connect", "credentialed connection denied", details, err)
		}
		return nil, err
	}
	return nc, nil
}

// UpdateAccountPerms pushes new account-default permissions live: the account
// claims are recompiled, stored, and applied to the running server, and
// connections riding the superseded claims re-authenticate under the new ones.
func (r *Runtime) UpdateAccountPerms(account string, perms core.Permissions) error {
	if r == nil || r.op == nil {
		return fail(AccountUpdateFailed, "UpdateAccountPerms", "operator mode is not enabled", nil, nil)
	}
	if perms.AllowResponses.Max > 0 && perms.AllowResponses.ExpiresMs <= 0 {
		return fail(AccountCompileFailed, "UpdateAccountPerms", "bounded response TTL is required", map[string]string{"account": account}, nil)
	}
	r.op.mu.Lock()
	defer r.op.mu.Unlock()
	acc := r.op.accounts[account]
	if acc == nil {
		return fail(AccountUpdateFailed, "UpdateAccountPerms", "unknown account", map[string]string{"account": account}, nil)
	}
	acc.setScope(perms)
	live, before, err := r.pushAccount(acc)
	if err != nil {
		return err
	}
	// Kicked scoped connections are removed synchronously by the push; give
	// them a bounded window to return under the superseding claims so callers
	// observe the new posture once this returns.
	deadline := time.Now().Add(3 * time.Second)
	for live.NumConnections() < before && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	return nil
}

// Revoke revokes a minted credential: the live connection is disconnected by
// the claims push and reconnects are denied.
func (r *Runtime) Revoke(account, userPub string) error {
	if r == nil || r.op == nil {
		return fail(RevocationFailed, "Revoke", "operator mode is not enabled", nil, nil)
	}
	r.op.mu.Lock()
	defer r.op.mu.Unlock()
	acc := r.op.accounts[account]
	if acc == nil {
		return fail(RevocationFailed, "Revoke", "unknown account", map[string]string{"account": account}, nil)
	}
	if !acc.minted[userPub] {
		return fail(RevocationFailed, "Revoke", "credential was not minted here", map[string]string{"account": account, "user": userPub}, nil)
	}
	acc.claims.Revoke(userPub)
	if _, _, err := r.pushAccount(acc); err != nil {
		return fail(RevocationFailed, "Revoke", "revocation could not be enforced", map[string]string{"account": account, "user": userPub}, err)
	}
	return nil
}

// MintTrustedWrapper issues a leaf-scoped JWT credential in the TB_APP account
// for a trusted wrapper principal. The credential grants:
//   - publish allow on tb.session.<sessionID>.ingest only
//   - subscribe allow on tb.session.<sessionID>.steer only
//
// No JetStream API, no output-subject publish, no steer-publish authority.
// The synthetic lease carries the sessionID as both SessionID and CapabilityID
// so the wrapper credential is traceable without an external lease store.
func MintTrustedWrapper(rt *Runtime, sessionID string) (UserCreds, error) {
	if rt == nil || rt.op == nil {
		return UserCreds{}, fail(JWTMintFailed, "MintTrustedWrapper", "operator mode is not enabled", nil, nil)
	}
	id := "wrapper-" + sessionID
	leaseID, err := secret()
	if err != nil {
		return UserCreds{}, fail(JWTMintFailed, "MintTrustedWrapper", "lease id could not be generated", nil, err)
	}
	auth := core.Auth{
		User: id,
		Capability: core.Capability{
			PrincipalID:   id,
			SessionID:     sessionID,
			CapabilityID:  "wrapper-cap-" + sessionID,
			LeaseID:       leaseID,
			LeaseStatus:   "active",
			AppRevision:   "wrapper.v1",
			SchemaVersion: "v1",
		},
		Permissions: core.Permissions{
			Publish:   core.PermList{Allow: []string{"tb.session." + sessionID + ".ingest"}},
			Subscribe: core.PermList{Allow: []string{"tb.session." + sessionID + ".steer", "_INBOX.>"}},
		},
	}
	return rt.MintUser(AppAccount, auth, time.Hour)
}

// IsRevoked reports whether userPub has been revoked in the given account.
// Returns false if the runtime is not in operator mode or the account is unknown.
func (r *Runtime) IsRevoked(account, userPub string) bool {
	if r == nil || r.op == nil {
		return false
	}
	r.op.mu.Lock()
	defer r.op.mu.Unlock()
	acc := r.op.accounts[account]
	if acc == nil {
		return false
	}
	// Use zero time so any revocation timestamp (always > epoch) matches.
	return acc.claims.Revocations.IsRevoked(userPub, time.Time{})
}

// ApplySteerAfterRevoke re-checks the steerer's revocation status at apply
// time. If the credential identified by userPub has been revoked in the
// TB_APP account, a typed SteerAfterRevoke error is returned. Nil means the
// credential is still valid and the steer may be applied.
func ApplySteerAfterRevoke(rt *Runtime, userPub string) error {
	if rt.IsRevoked(AppAccount, userPub) {
		return fail(SteerAfterRevoke, "ApplySteerAfterRevoke", "steer denied: wrapper credential revoked", map[string]string{"userPub": userPub}, nil)
	}
	return nil
}

// pushAccount recompiles the account claims, stores them in the resolver, and
// applies them to the live server. Lock must be held. Returns the live server
// account and its connection count sampled before the update.
func (r *Runtime) pushAccount(acc *opAccount) (*natsserver.Account, int, error) {
	token, err := acc.claims.Encode(r.op.kp)
	if err != nil {
		return nil, 0, fail(AccountCompileFailed, "UpdateAccountClaims", "account claims could not be compiled", map[string]string{"account": acc.claims.Name}, err)
	}
	decoded, err := jwt.DecodeAccountClaims(token)
	if err != nil {
		return nil, 0, fail(AccountCompileFailed, "UpdateAccountClaims", "account claims round-trip failed", map[string]string{"account": acc.claims.Name}, err)
	}
	if err := r.op.resolver.Store(acc.pub, token); err != nil {
		return nil, 0, fail(AccountUpdateFailed, "UpdateAccountClaims", "account claims could not be stored", map[string]string{"account": acc.claims.Name}, err)
	}
	live, err := r.srv.LookupAccount(acc.pub)
	if err != nil {
		return nil, 0, fail(AccountUpdateFailed, "UpdateAccountClaims", "live account lookup failed", map[string]string{"account": acc.claims.Name}, err)
	}
	before := live.NumConnections()
	r.srv.UpdateAccountClaims(live, decoded)
	return live, before, nil
}

// emptyPerms decides whether a mint rides the deny-by-default scoped signer.
// AllowResponses counts only when it grants (Max > 0): a degenerate bound must
// never promote a subject-less mint onto the root key with allow-all perms.
func emptyPerms(p core.Permissions) bool {
	return len(p.Publish.Allow)+len(p.Publish.Deny)+len(p.Subscribe.Allow)+len(p.Subscribe.Deny) == 0 &&
		p.AllowResponses.Max <= 0
}

func jwtPerms(p core.Permissions) jwt.Permissions {
	out := jwt.Permissions{
		Pub: jwt.Permission{Allow: copyStrings(p.Publish.Allow), Deny: copyStrings(p.Publish.Deny)},
		Sub: jwt.Permission{Allow: copyStrings(p.Subscribe.Allow), Deny: copyStrings(p.Subscribe.Deny)},
	}
	if p.AllowResponses.Max > 0 {
		out.Resp = &jwt.ResponsePermission{
			MaxMsgs: p.AllowResponses.Max,
			Expires: time.Duration(p.AllowResponses.ExpiresMs) * time.Millisecond,
		}
	}
	return out
}

func leaseFromClaims(uc *jwt.UserClaims) (core.Capability, error) {
	for _, tag := range uc.Tags {
		rest, ok := strings.CutPrefix(tag, leaseTag)
		if !ok {
			continue
		}
		raw, err := hex.DecodeString(rest)
		if err != nil {
			return core.Capability{}, err
		}
		var cap core.Capability
		if err := json.Unmarshal(raw, &cap); err != nil {
			return core.Capability{}, err
		}
		return cap, nil
	}
	return core.Capability{}, errors.New("lease tag missing from user JWT")
}
