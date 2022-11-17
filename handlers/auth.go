package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"nginx_auth_server/server/lookup"
	"nginx_auth_server/server/metrics"
)

func (env *Handlers) Auth(rw http.ResponseWriter, req *http.Request) {

	metrics.Metrics.Inc("AuthRequests", 1)

	authMethod := req.Header.Get("Auth-Method")
	authUser := req.Header.Get("Auth-User")
	authPass := req.Header.Get("Auth-Pass")
	authProtocol := req.Header.Get("Auth-Protocol")
	authLoginAttempt := req.Header.Get("Auth-Login-Attempt")
	clientIP := req.Header.Get("Client-IP")
	clientHost := req.Header.Get("Client-Host")
	authSMTPHelo := req.Header.Get("Auth-SMTP-Helo")
	authSMTPFrom := req.Header.Get("Auth-SMTP-From")
	authSMTPTo := req.Header.Get("Auth-SMTP-To")

	log.Info().
		Str("authMethod", authMethod).
		Str("authUser", authUser).
		Str("authPass", lookup.WrapSecret(authPass)).
		Str("authProtocol", authProtocol).
		Str("authLoginAttempt", authLoginAttempt).
		Str("clientIP", clientIP).
		Str("clientHost", clientHost).
		Str("authSMTPHelo", authSMTPHelo).
		Str("authSMTPFrom", authSMTPFrom).
		Str("authSMTPTo", authSMTPTo).
		Str("event", "auth").
		Msgf("Incoming auth request")

	success, result, err := lookup.Authenticate(authUser, authPass, authProtocol, authSMTPFrom, authSMTPTo)
	if err != nil {

		metrics.Metrics.Inc("InternalErrors", 1)

		log.Error().
			Err(err).
			Str("event", "auth").
			Msgf("Can't authenticate %s:%s", authUser, lookup.WrapSecret(authPass))
	}

	rw.Header().Set("Auth-Status", result.AuthStatus)

	log.Debug().
		Str("AuthStatus", result.AuthStatus).
		Str("AuthServer", result.AuthServer).
		Int("AuthPort", result.AuthPort).
		Str("AuthWait", result.AuthWait).
		Str("AuthErrorCode", result.AuthErrorCode).
		Bool("AuthViaRelay", result.AuthViaRelay).
		Bool("AuthViaLogin", result.AuthViaLogin).
		Bool("success", success).
		Msgf("Got result from authentication function")

	// if success {

	// 	log.Debug().
	// 		Msgf("Registering successful authentication in metrics")

	// 	metrics.Metrics.Inc("AuthRequestsSuccess", 1)

	// 	if result.AuthViaRelay {
	// 		metrics.Metrics.Inc("AuthRequestsSuccessRelay", 1)

	// 	} else if result.AuthViaLogin {
	// 		metrics.Metrics.Inc("AuthRequestsSuccessLogin", 1)
	// 	}

	// 	log.Info().
	// 		Str("authMethod", authMethod).
	// 		Str("authUser", authUser).
	// 		Str("authPass", lookup.WrapSecret(authPass)).
	// 		Str("authProtocol", authProtocol).
	// 		Str("authLoginAttempt", authLoginAttempt).
	// 		Str("clientIP", clientIP).
	// 		Str("clientHost", clientHost).
	// 		Str("authSMTPHelo", authSMTPHelo).
	// 		Str("authSMTPFrom", authSMTPFrom).
	// 		Str("authSMTPTo", authSMTPTo).
	// 		Str("event", "auth").
	// 		Str("AuthStatus", result.AuthStatus).
	// 		Str("AuthServer", result.AuthServer).
	// 		Int("AuthPort", result.AuthPort).
	// 		Str("AuthWait", result.AuthWait).
	// 		Str("AuthErrorCode", result.AuthErrorCode).
	// 		Str("event", "auth_ok").
	// 		Msgf("Successful authentication")

	// 	rw.Header().Set("Auth-Server", result.AuthServer)
	// 	rw.Header().Set("Auth-Port", strconv.Itoa(result.AuthPort))

	// } else {

	// 	metrics.Metrics.Inc("AuthRequestsFailed", 1)

	// 	if result.AuthViaRelay {
	// 		metrics.Metrics.Inc("AuthRequestsFailedRelay", 1)

	// 	} else if result.AuthViaLogin {
	// 		metrics.Metrics.Inc("AuthRequestsFailedLogin", 1)
	// 	}

	// 	log.Info().
	// 		Str("authMethod", authMethod).
	// 		Str("authUser", authUser).
	// 		Str("authPass", lookup.WrapSecret(authPass)).
	// 		Str("authProtocol", authProtocol).
	// 		Str("authLoginAttempt", authLoginAttempt).
	// 		Str("clientIP", clientIP).
	// 		Str("clientHost", clientHost).
	// 		Str("authSMTPHelo", authSMTPHelo).
	// 		Str("authSMTPFrom", authSMTPFrom).
	// 		Str("authSMTPTo", authSMTPTo).
	// 		Str("event", "auth").
	// 		Str("AuthStatus", result.AuthStatus).
	// 		Str("AuthServer", result.AuthServer).
	// 		Int("AuthPort", result.AuthPort).
	// 		Str("AuthWait", result.AuthWait).
	// 		Str("AuthErrorCode", result.AuthErrorCode).
	// 		Str("event", "auth_failed").
	// 		Msgf("Failed to authenticate")

	// 	rw.Header().Set("Auth-Wait", result.AuthWait)

	// 	if result.AuthErrorCode != "" {
	// 		rw.Header().Set("Auth-Error-Code", result.AuthErrorCode)
	// 	}
	// }

}
