package runtime

import (
	"context"
	"sync"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/log"
)

// GetRoutersByEntryPoints returns all the http routers by entry points name and routers name
func (c *Configuration) GetRoutersByEntryPoints(ctx context.Context, entryPoints []string, tls bool) map[string]map[string]*RouterInfo {
	entryPointsRouters := make(map[string]map[string]*RouterInfo)

	for rtName, rt := range c.Routers {
		if (tls && rt.TLS == nil) || (!tls && rt.TLS != nil) {
			continue
		}

		eps := rt.EntryPoints
		if len(eps) == 0 {
			eps = entryPoints
		}
		for _, entryPointName := range eps {
			if !contains(entryPoints, entryPointName) {
				log.FromContext(log.With(ctx, log.Str(log.EntryPointName, entryPointName))).
					Errorf("entryPoint %q doesn't exist", entryPointName)
				continue
			}

			if _, ok := entryPointsRouters[entryPointName]; !ok {
				entryPointsRouters[entryPointName] = make(map[string]*RouterInfo)
			}

			entryPointsRouters[entryPointName][rtName] = rt
		}
	}

	return entryPointsRouters
}

// RouterInfo holds information about a currently running HTTP router
type RouterInfo struct {
	*dynamic.Router // dynamic configuration
	// Err contains all the errors that occurred during router's creation.
	Err []string `json:"error,omitempty"`
	// Status reports whether the router is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string `json:"status,omitempty"`
}

// AddError adds err to r.Err, if it does not already exist.
// If critical is set, r is marked as disabled.
func (r *RouterInfo) AddError(err error, critical bool) {
	for _, value := range r.Err {
		if value == err.Error() {
			return
		}
	}

	r.Err = append(r.Err, err.Error())
	if critical {
		r.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if r.Status != StatusDisabled {
		r.Status = StatusWarning
	}
}

// MiddlewareInfo holds information about a currently running middleware
type MiddlewareInfo struct {
	*dynamic.Middleware // dynamic configuration
	// Err contains all the errors that occurred during service creation.
	Err    []string `json:"error,omitempty"`
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers and services using that middleware
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, m is marked as disabled.
func (m *MiddlewareInfo) AddError(err error, critical bool) {
	for _, value := range m.Err {
		if value == err.Error() {
			return
		}
	}

	m.Err = append(m.Err, err.Error())
	if critical {
		m.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if m.Status != StatusDisabled {
		m.Status = StatusWarning
	}
}

// ServiceInfo holds information about a currently running service
type ServiceInfo struct {
	*dynamic.Service // dynamic configuration
	// Err contains all the errors that occurred during service creation.
	Err []string `json:"error,omitempty"`
	// Status reports whether the service is disabled, in a warning state, or all good (enabled).
	// If not in "enabled" state, the reason for it should be in the list of Err.
	// It is the caller's responsibility to set the initial status.
	Status string   `json:"status,omitempty"`
	UsedBy []string `json:"usedBy,omitempty"` // list of routers using that service

	serverStatusMu sync.RWMutex
	serverStatus   map[string]string // keyed by server URL
}

// AddError adds err to s.Err, if it does not already exist.
// If critical is set, s is marked as disabled.
func (s *ServiceInfo) AddError(err error, critical bool) {
	for _, value := range s.Err {
		if value == err.Error() {
			return
		}
	}

	s.Err = append(s.Err, err.Error())
	if critical {
		s.Status = StatusDisabled
		return
	}

	// only set it to "warning" if not already in a worse state
	if s.Status != StatusDisabled {
		s.Status = StatusWarning
	}
}

// UpdateServerStatus sets the status of the server in the ServiceInfo.
// It is the responsibility of the caller to check that s is not nil.
func (s *ServiceInfo) UpdateServerStatus(server string, status string) {
	s.serverStatusMu.Lock()
	defer s.serverStatusMu.Unlock()

	if s.serverStatus == nil {
		s.serverStatus = make(map[string]string)
	}
	s.serverStatus[server] = status
}

// GetAllStatus returns all the statuses of all the servers in ServiceInfo.
// It is the responsibility of the caller to check that s is not nil
func (s *ServiceInfo) GetAllStatus() map[string]string {
	s.serverStatusMu.RLock()
	defer s.serverStatusMu.RUnlock()

	if len(s.serverStatus) == 0 {
		return nil
	}

	allStatus := make(map[string]string, len(s.serverStatus))
	for k, v := range s.serverStatus {
		allStatus[k] = v
	}
	return allStatus
}
