package api

import "strings"

// originChecker decides whether an Origin header value is permitted, based on a
// configured allowlist. An empty allowlist (or one containing "*") permits any
// origin — the default, since prod is same-origin (embedded SPA) and dev goes
// through the Vite proxy. A non-empty list is used when the frontend is served
// from a different origin than the backend (VITE_API_BASE builds).
type originChecker struct {
	allowAll bool
	allowed  map[string]struct{}
}

func newOriginChecker(allowOrigins []string) originChecker {
	oc := originChecker{allowed: make(map[string]struct{}, len(allowOrigins))}
	if len(allowOrigins) == 0 {
		oc.allowAll = true
		return oc
	}
	for _, o := range allowOrigins {
		if o == "*" {
			oc.allowAll = true
			continue
		}
		oc.allowed[strings.TrimRight(o, "/")] = struct{}{}
	}
	return oc
}

func (oc originChecker) allows(origin string) bool {
	if oc.allowAll {
		return true
	}
	_, ok := oc.allowed[strings.TrimRight(origin, "/")]
	return ok
}
