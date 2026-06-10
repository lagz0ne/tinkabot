package embednats

// Exposure is the typed posture a runtime declares instead of raw socket
// fields: in-process by default (no TCP socket, served over
// Server.InProcessConn), loopback as an explicit opt-in, and external as a
// typed, denied-by-default opt-in per surface.

type ExposureMode string

const (
	ExposeInProcess ExposureMode = "in-process"
	ExposeLoopback  ExposureMode = "loopback"
	ExposeExternal  ExposureMode = "external"
)

// AuthTier names the auth tier an exposure surface requires. TierExternal is
// a typed policy requirement only: the live tier backend lands with the
// operator-jwt-authority slice, so declaring it never enables serving here.
type AuthTier string

const TierExternal AuthTier = "external"

// TLSFiles declares TLS material for an external surface. This slice types
// the declaration and its denial paths; no certificate is ever loaded.
type TLSFiles struct {
	Cert string
	Key  string
}

// ExternalSurfaces is the per-surface external opt-in: each enabled surface
// must carry a matching auth tier and TLS beyond loopback or Start fails
// with a typed ExposureDenied error.
type ExternalSurfaces struct {
	NATS      bool
	WebSocket bool
	Gateway   bool
	AuthTier  AuthTier
	TLS       TLSFiles
}

type Exposure struct {
	Mode     ExposureMode
	External ExternalSurfaces
}

func InProcess() Exposure { return Exposure{Mode: ExposeInProcess} }
func Loopback() Exposure  { return Exposure{Mode: ExposeLoopback} }
func External(s ExternalSurfaces) Exposure {
	return Exposure{Mode: ExposeExternal, External: s}
}

// ExposurePosture is the observable side of a declared Exposure: the mode the
// runtime actually runs under and its bound address ("" when no socket exists).
type ExposurePosture struct {
	Mode ExposureMode
	Addr string
}

// exposure resolves the declared posture against the raw socket fields before
// defaults apply: a socket request without a matching declaration is a typed
// failure, and external surfaces are denied unless fully declared (and stay
// denied in this slice — live external serving is not provided).
func (cfg Config) exposure() (ExposureMode, error) {
	mode := cfg.Exposure.Mode
	if mode == "" {
		mode = ExposeInProcess
	}
	switch mode {
	case ExposeInProcess:
		if cfg.Host != "" || cfg.Port != 0 || cfg.WebSocket != (WebSocket{}) || cfg.Core.Topology.WebSocket.Enabled {
			return "", fail(ExposureUndeclared, "Start", "socket fields require a declared loopback posture", nil, nil)
		}
	case ExposeLoopback:
		// Every socket under a loopback posture must be loopback BEFORE any
		// server is constructed: a non-loopback host is an external surface and
		// requires ExposeExternal. Refusing here (deny-wins) means the server
		// never binds — Start's post-start srv.Addr() mismatch check stays only
		// as a backstop. An empty websocket host inherits cfg.Host via
		// defaults(), so checking cfg.Host covers both listeners.
		if cfg.Host != "" && cfg.Host != "127.0.0.1" {
			return "", fail(ExposureDenied, "Start", "host beyond loopback requires a declared external posture",
				map[string]string{"host": cfg.Host}, nil)
		}
		if cfg.WebSocket.Host != "" && cfg.WebSocket.Host != "127.0.0.1" {
			return "", fail(ExposureDenied, "Start", "websocket host beyond loopback requires a declared external posture",
				map[string]string{"host": cfg.WebSocket.Host}, nil)
		}
	case ExposeExternal:
		return "", cfg.Exposure.External.deny()
	default:
		return "", fail(ExposureUndeclared, "Start", "unknown exposure mode", map[string]string{"mode": string(mode)}, nil)
	}
	return mode, nil
}

func (s ExternalSurfaces) deny() *Error {
	surfaces := []struct {
		name string
		on   bool
	}{
		{"nats", s.NATS},
		{"websocket", s.WebSocket},
		{"gateway", s.Gateway},
	}
	for _, surface := range surfaces {
		if !surface.on {
			continue
		}
		details := map[string]string{"surface": surface.name}
		if s.AuthTier != TierExternal {
			return fail(ExposureDenied, "Start", "external surface requires a matching auth tier", details, nil)
		}
		if s.TLS.Cert == "" || s.TLS.Key == "" {
			return fail(ExposureDenied, "Start", "external surface requires TLS beyond loopback", details, nil)
		}
	}
	// Denied-by-default: even a fully declared external tier only passes the
	// policy check; serving real external traffic is out of scope for this
	// slice (docs/matched-abstraction/plan/endgame-app.md:177).
	return fail(ExposureDenied, "Start", "external exposure is denied by default; live external serving is not provided", nil, nil)
}
