package handlers

import (
	"golang.org/x/sync/semaphore"

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

	AddQueue     chan AddRequestStruct ``
	AddSemaphore semaphore.Weighted    ``

	CheckDuplicatesQueue     chan interface{}   ``
	CheckDuplicatesSemaphore semaphore.Weighted ``
}

type ResponseErrorStruct struct {
	Message string `json:"message"`
}

type ResponseResultStruct struct {
	Error   *ResponseErrorStruct `json:"error,omitempty"`
	Success bool                 `json:"success"`
}

type AddResponseStruct struct {
	Result      ResponseResultStruct `json:"result"`
	NewRecordID uint64               `json:"new_record_id,omitempty"`
}

type GetResponseStruct struct {
	Password string               `json:"password"`
	Result   ResponseResultStruct `json:"result"`
}

type GetNewResponseStruct struct {
	NewPasswords []string             `json:"new_passwords"`
	Result       ResponseResultStruct `json:"result"`
}

type AddRequestStruct struct {
	Username   string `json:"username"    db:"username"    metrics:"username"`
	Password   string `json:"password"    db:"password"    metrics:"password"`
	RemoteHost string `json:"remote_host" db:"remote_host" metrics:"remote_host"`
	Source     string `json:"source"      db:"source"`
}
