package handlers

import (
	"fmt"
	"net/http"

	"nginx_auth_server/server/metrics"
)

func (env *Handlers) Metrics(rw http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(rw, "# TYPE AuthRequests counter\n")
	fmt.Fprintf(rw, "# HELP Number of events happened in Nginx Mail Auth Server\n")

	for item := range metrics.Metrics.Entities {
		fmt.Fprintf(rw, "AuthRequests{kind=\"%v\"} %v\n", item, metrics.Metrics.Entities[item])
	}

	// fmt.Fprintf(rw, "# TYPE AuthRequests counter\n")
	// fmt.Fprintf(rw, "AuthRequests{result=\"started\"} %v\n", metrics.Metrics.AuthRequests)
	// fmt.Fprintf(rw, "AuthRequests{result=\"fail\"} %v\n", metrics.Metrics.AuthRequestsFailed)
	// fmt.Fprintf(rw, "AuthRequests{result=\"fail_relay\"} %v\n", metrics.Metrics.AuthRequestsFailedRelay)
	// fmt.Fprintf(rw, "AuthRequests{result=\"fail_login\"} %v\n", metrics.Metrics.AuthRequestsFailedLogin)
	// fmt.Fprintf(rw, "AuthRequests{result=\"success\"} %v\n", metrics.Metrics.AuthRequestsSuccess)
	// fmt.Fprintf(rw, "AuthRequests{result=\"success_relay\"} %v\n", metrics.Metrics.AuthRequestsSuccessRelay)
	// fmt.Fprintf(rw, "AuthRequests{result=\"success_login\"} %v\n", metrics.Metrics.AuthRequestsSuccessLogin)
	// fmt.Fprintf(rw, "AuthRequests{kind=\"relay\"} %v\n", metrics.Metrics.AuthRequestsRelay)
	// fmt.Fprintf(rw, "AuthRequests{kind=\"login\"} %v\n", metrics.Metrics.AuthRequestsLogin)

	fmt.Fprintf(rw, "# TYPE InternalErrors counter\n")
	fmt.Fprintf(rw, "InternalErrors %v\n", metrics.Metrics.InternalErrors)

}
