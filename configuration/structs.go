package configuration

import (
	"net"

	"nginx_auth_server/server/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/sony/sonyflake"
)

type DatabaseStruct struct {
	URI                string   `json:"uri"`
	Driver             string   `json:"driver"`
	AuthLookupQueries  []string `json:"auth_lookup_queries"`
	RelayLookupQueries []string `json:"relay_lookup_queries"`
}

type ConfigurationStruct struct {
	ListenAddress          string                `json:"listen"`
	ListenNetTCPAddr       *net.TCPAddr          ``
	Logfile                string                `json:"logfile"`
	Database               DatabaseStruct        `json:"database"`
	DB                     *sqlx.DB              ``
	Metrics                metrics.MetricsStruct ``
	Flake                  sonyflake.Sonyflake   ``
	ApplicationDescription string                ``
	BuildVersion           string                ``
	ConfigFile             string                ``
}

var Configuration = ConfigurationStruct{}
