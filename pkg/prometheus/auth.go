package prometheus

import (
	"net/http"
	"strings"
)

type Authentificator interface {
	PerformAuthentification(handler http.HandlerFunc) http.HandlerFunc
}

type BasicAuthConfig struct {
	Username string
	Password string
}

type NoAuth struct {
}

func (c *NoAuth) PerformAuthentification(handler http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, rq *http.Request) {
		handler(rw, rq)
	}
}

func (c *BasicAuthConfig) PerformAuthentification(handler http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, rq *http.Request) {
		u, p, ok := rq.BasicAuth()
		if !ok || len(strings.TrimSpace(u)) < 1 || len(strings.TrimSpace(p)) < 1 {
			unauthorised(rw)
			return
		}

		if u != c.Username || p != c.Password {
			unauthorised(rw)
			return
		}

		handler(rw, rq)
	}
}

func unauthorised(rw http.ResponseWriter) {
	rw.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
	rw.WriteHeader(http.StatusUnauthorized)
}
