package configuration

import (
	"net"
	// "time"

	"shittypasswords/api/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/sony/sonyflake"
)

type DatabaseStruct struct {
	URI    string `json:"uri"`
	Driver string `json:"driver"`
}

type ConfigurationStruct struct {
	ListenAddress          string                `json:"listen"`
	ListenNetTCPAddr       *net.TCPAddr          ``
	Database               DatabaseStruct        `json:"database"`
	DB                     *sqlx.DB              ``
	Metrics                metrics.MetricsStruct ``
	ApplicationDescription string                ``
	BuildVersion           string                ``
	Flake                  sonyflake.Sonyflake   ``
}
