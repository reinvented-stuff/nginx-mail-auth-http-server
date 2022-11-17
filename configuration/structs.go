package configuration

import (
	"net"
	// "time"

	"nginx_auth_server/server/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/sony/sonyflake"
)

type DatabaseStruct struct {
	URI              string   `json:"uri"`
	Driver           string   `json:"driver"`
	AuthLookupQuery  []string `json:"auth_lookup_queries"`
	RelayLookupQuery []string `json:"relay_lookup_queries"`
}

// type ConfigurationStruct struct {
// 	ListenAddress          string                `json:"listen"`
// 	ListenNetTCPAddr       *net.TCPAddr          ``
// 	Database               DatabaseStruct        `json:"database"`
// 	DB                     *sqlx.DB              ``
// 	Metrics                metrics.MetricsStruct ``
// 	ApplicationDescription string                ``
// 	BuildVersion           string                ``
// 	Flake                  sonyflake.Sonyflake   ``
// }

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
	// SSL              *SSLStruct     `json:"ssl,omitempty"`
}

var Configuration = ConfigurationStruct{}
