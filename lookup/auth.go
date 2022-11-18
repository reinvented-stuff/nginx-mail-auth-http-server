package lookup

import (
	"bytes"
	"errors"
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

var mailHeaderRegex = regexp.MustCompile(`<(.*?)(\+.*?)?@(.*?)>`)

func parseEmailAddress(email string) (emailParsed string, emailParts []string, err error) {

	var nonVERPAddress bytes.Buffer

	emailMatch := mailHeaderRegex.FindStringSubmatch(email)

	log.Debug().
		Str("email", email).
		Msgf("Parsing email address")

	if len(emailMatch) == 4 {
		nonVERPAddress.WriteString(emailMatch[1])
		nonVERPAddress.WriteString("@")
		nonVERPAddress.WriteString(emailMatch[3])

		// nonVERPAddress.Reset()

		log.Debug().
			Str("email", email).
			Str("emailParsed", nonVERPAddress.String()).
			Strs("emailMatch", emailMatch).
			Msgf("Parsed email address")

		return nonVERPAddress.String(), emailMatch, nil

	} else {

		log.Warn().
			Str("nonVERPAddress", nonVERPAddress.String()).
			Str("email", email).
			Msgf("Email address is empty (could be incoming bounce)")

		return "", emailMatch, nil

	}

}

func canRelay(mailFrom string, rcptTo string) bool {
	if rcptTo != "" {
		return true
	} else {
		return false
	}
}

func canAuthenticate(user string, pass string) bool {
	if user != "" {
		return true
	} else {
		return false
	}
}

func Authenticate(user string, pass string, protocol string, mailFrom string, rcptTo string, clientIP string) (success bool, result authResultStruct, err error) {

	log.Info().
		Str("user", user).
		Str("pass", WrapSecret(pass)).
		Str("protocol", protocol).
		Str("mailFrom", mailFrom).
		Str("rcptTo", rcptTo).
		Msgf("Processing authentication request")

	var queries []string
	var queryParams = QueryParamsStruct{
		User:     user,
		Pass:     pass,
		RcptTo:   "",
		MailFrom: "",
		ClientIP: clientIP,
	}

	if canRelay(mailFrom, rcptTo) && !canAuthenticate(user, pass) {

		metrics.Metrics.Inc("AuthRequestsRelay", 1)
		result.AuthViaRelay = true

		log.Info().
			Str("protocol", protocol).
			Str("mailFrom", mailFrom).
			Str("rcptTo", rcptTo).
			Msg("Authenticating by relay access ('user' and 'pass' are empty)")

		queryParams.MailFrom, _, err = parseEmailAddress(mailFrom)
		if err != nil {
			metrics.Metrics.Inc("InternalErrors", 1)
			log.Error().
				Err(err).
				Str("queryParams.MailFrom", queryParams.MailFrom).
				Msgf("Error while parsing MailFrom address")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, errors.New("Error while parsing MailFrom address")
		}

		queryParams.RcptTo, _, err = parseEmailAddress(rcptTo)
		if err != nil {
			metrics.Metrics.Inc("InternalErrors", 1)
			log.Error().
				Err(err).
				Str("queryParams.RcptTo", queryParams.RcptTo).
				Msgf("Error while parsing RcptTo address")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, errors.New("Error while parsing RcptTo address")
		}

		if queryParams.RcptTo == "" {
			metrics.Metrics.Inc("InternalErrors", 1)
			log.Error().
				Str("rcptTo", rcptTo).
				Str("queryParams.RcptTo", queryParams.RcptTo).
				Msg("Can't parse RCPT TO command for relay")

			result.AuthStatus = "Temporary server problem, try again later"
			result.AuthErrorCode = "451 4.3.0"
			result.AuthWait = "5"

			return false, result, errors.New("Can't parse RCPT TO command for relay")
		}

		log.Debug().
			Str("MailFrom", mailFrom).
			Str("RcptTo", rcptTo).
			Str("queryParams.MailFrom", queryParams.MailFrom).
			Str("queryParams.RcptTo", queryParams.RcptTo).
			Msg("Relay lookup query parameters prepared")

		queries = configuration.Configuration.Database.RelayLookupQueries

	} else if canAuthenticate(user, pass) {

		metrics.Metrics.Inc("AuthRequestsLogin", 1)
		result.AuthViaLogin = true

		log.Info().
			Str("protocol", protocol).
			Str("user", user).
			Str("pass", WrapSecret(pass)).
			Msg("Authenticating by credentials")

		queries = configuration.Configuration.Database.AuthLookupQueries

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
		Strs("queries", queries).
		Msg("Lookup query prepared")

	for idx, query := range queries {

		log.Debug().
			Int("queryIdx", idx).
			Str("query", query).
			Msg("Submitting lookup query")

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
				Int("queryIdx", idx).
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
				Int("queryIdx", idx).
				Msgf("Found upstream")

			result.AuthStatus = "OK"
			result.AuthServer = upstream.Address
			result.AuthPort = upstream.Port

			return true, result, nil

		}

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
