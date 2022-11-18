package handlers

import (
	"github.com/jmoiron/sqlx"
	"github.com/sony/sonyflake"

	"nginx_auth_server/server/lookup"
	"nginx_auth_server/server/metrics"
)

type Handlers struct {
	DB                     *sqlx.DB               ``
	MetricsCounter         *metrics.MetricsStruct ``
	Lookup                 *lookup.LookupStruct   ``
	ApplicationDescription string                 ``
	BuildVersion           string                 ``
	Flake                  *sonyflake.Sonyflake   ``
}
