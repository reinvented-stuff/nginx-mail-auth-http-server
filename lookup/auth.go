package lookup

import (
	"bytes"
	// "encoding/json"
	// "flag"
	// "fmt"
	// "html"
	// "net/http"
	// "os"
	// "os/signal"
	"regexp"
	// "strconv"
	// "sync/atomic"
	// "syscall"
	// "time"

	"github.com/rs/zerolog/log"

	"nginx_auth_server/server/configuration"
	"nginx_auth_server/server/metrics"
)

func Authenticate(user string, pass string, protocol string, mailFrom string, rcptTo string) (success bool, result authResultStruct, err error) {

	result = authResultStruct{}

	var nonVERPAddress bytes.Buffer
	var query string
	var queryParams = QueryParamsStruct{
		User:     user,
		Pass:     pass,
		RcptTo:   "",
		MailFrom: "",
	}

	log.Info().
		Str("user", user).
		Str("pass", WrapSecret(pass)).
		Str("protocol", protocol).
		Str("mailFrom", mailFrom).
		Str("rcptTo", rcptTo).
		Msgf("Processing authentication request")

	if user == "" &&
		pass == "" &&
		rcptTo != "" {

		// metrics.Metrics.Inc("AuthRequestsRelay", 1)

		result.AuthViaRelay = true

		log.Info().
			Str("protocol", protocol).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msg("Authenticating by relay access ('user' and 'pass' are empty)")

		mailHeaderRegex := regexp.MustCompile(`<(.*?)(\+.*?)?@(.*?)>`)
		mailFromEmailMatch := mailHeaderRegex.FindStringSubmatch(mailFrom)
		rcptToEmailMatch := mailHeaderRegex.FindStringSubmatch(rcptTo)

		if len(mailFromEmailMatch) != 4 {
			log.Warn().
				Str("mailFrom", mailFrom).
				Str("rcptTo", rcptTo).
				Msgf("MAIL FROM is empty (could be incoming bounce)")

		} else {
			nonVERPAddress.WriteString(mailFromEmailMatch[1])
			nonVERPAddress.WriteString("@")
			nonVERPAddress.WriteString(mailFromEmailMatch[3])

			queryParams.MailFrom = nonVERPAddress.String()

			nonVERPAddress.Reset()
		}

		if len(rcptToEmailMatch) == 4 {

			nonVERPAddress.WriteString(rcptToEmailMatch[1])
			nonVERPAddress.WriteString("@")
			nonVERPAddress.WriteString(rcptToEmailMatch[3])

			queryParams.RcptTo = nonVERPAddress.String()

			log.Debug().
				Str("rcptToEmailMatch[0]", rcptToEmailMatch[0]).
				Str("rcptToEmailMatch[1]", rcptToEmailMatch[1]).
				Str("rcptToEmailMatch[2]", rcptToEmailMatch[2]).
				Str("rcptToEmailMatch[3]", rcptToEmailMatch[3]).
				Str("nonVERPAddress.string()", nonVERPAddress.String()).
				Str("queryParams.RcptTo", queryParams.RcptTo).
				Str("rcptTo", rcptTo).
				Msgf("Fetched an email address out of rcptTo header (VERP section stripped)")

			nonVERPAddress.Reset()

		} else {

			metrics.Metrics.Inc("InternalErrors", 1)

			log.Error().
				Str("rcptTo", rcptTo).
				Int("rcptToEmailMatchLen", len(rcptToEmailMatch)).
				Msg("Can't parse MAIL FROM command")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		}

		log.Debug().
			Str("User", queryParams.User).
			Str("Pass", WrapSecret(queryParams.Pass)).
			Str("MailFrom", queryParams.MailFrom).
			Str("RcptTo", queryParams.RcptTo).
			Msg("Relay lookup query parameters prepared")

		query = configuration.Configuration.Database.RelayLookupQuery[0]

	} else if user != "" {

		metrics.Metrics.Inc("AuthRequestsLogin", 1)

		result.AuthViaLogin = true

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", WrapSecret(pass)).
			Msg("Authenticating by credentials")

		query = configuration.Configuration.Database.AuthLookupQuery[0]

	} else {
		metrics.Metrics.Inc("InternalErrors", 1)

		log.Error().
			Str("User", queryParams.User).
			Str("Pass", WrapSecret(queryParams.Pass)).
			Str("MailFrom", queryParams.MailFrom).
			Str("RcptTo", queryParams.RcptTo).
			Msg("Can't authenticate via relay nor login")

		result.AuthStatus = "Temporary server problem, try again later"
		result.AuthErrorCode = "451 4.3.0"
		result.AuthWait = "5"

		return false, result, err

	}

	log.Debug().
		Str("query", query).
		Msg("Lookup query prepared")

	queryResult, err := configuration.Configuration.DB.NamedQuery(query, queryParams)

	if err != nil {

		metrics.Metrics.Inc("InternalErrors", 1)

		log.Error().Err(err).Msgf("Error while executing query: %v", err)

		result.AuthStatus = "Temporary server problem, try again later"
		result.AuthErrorCode = "451 4.3.0"
		result.AuthWait = "5"

		return false, result, err
	}

	defer queryResult.Close()

	for queryResult.Next() {
		log.Debug().
			Str("protocol", protocol).
			Str("user", WrapSecret(user)).
			Str("pass", pass).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msgf("Found results after lookup query execution")

		var upstream = UpstreamStruct{}

		if err = queryResult.StructScan(&upstream); err != nil {

			metrics.Metrics.Inc("InternalErrors", 1)

			log.Error().Err(err).Msgf("Error while parsing lookup query result")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, err

		} else {
			log.Debug().Msgf("Lookup query results parsed successfully")
		}

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", WrapSecret(pass)).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Str("upstreamAddress", upstream.Address).
			Int("upstreamPort", upstream.Port).
			Msgf("Found upstream")

		result.AuthStatus = "OK"
		result.AuthServer = upstream.Address
		result.AuthPort = upstream.Port

		return true, result, nil

	}

	log.Info().
		Str("protocol", protocol).
		Str("user", user).
		Str("pass", WrapSecret(pass)).
		Str("mailFrom", mailFrom).
		Str("rcptTo", rcptTo).
		Msgf("No results after lookup")

	result.AuthStatus = "Error: authentication failed."
	result.AuthErrorCode = "535 5.7.8"
	result.AuthWait = "5"

	return false, result, nil
}
