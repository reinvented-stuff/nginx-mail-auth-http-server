package handlers

import (
	"fmt"
	"html"
	"net/http"

	"nginx_auth_server/server/metrics"
)

func (env *Handlers) Index(rw http.ResponseWriter, req *http.Request) {
	metrics.Metrics.Inc("request_index", 1)
	fmt.Fprintf(rw, "%s v%s\n", html.EscapeString(env.ApplicationDescription), html.EscapeString(env.BuildVersion))
}
