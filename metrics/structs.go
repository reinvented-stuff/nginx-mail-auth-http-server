package metrics

import (
	"sync"
	"time"
)

type MetricEntity struct {
	Tag   string
	Value int32
}

type MetricsStruct struct {
	Warnings int32
	Errors   int32

	AuthRequests             int32
	AuthRequestsFailed       int32
	AuthRequestsFailedRelay  int32
	AuthRequestsFailedLogin  int32
	AuthRequestsSuccess      int32
	AuthRequestsSuccessRelay int32
	AuthRequestsSuccessLogin int32
	AuthRequestsRelay        int32
	AuthRequestsLogin        int32
	InternalErrors           int32

	Entities map[string]int32

	IncrementMutex sync.Mutex

	Started time.Time
}

var Metrics = MetricsStruct{
	AuthRequests:             0,
	AuthRequestsFailed:       0,
	AuthRequestsFailedRelay:  0,
	AuthRequestsFailedLogin:  0,
	AuthRequestsSuccess:      0,
	AuthRequestsSuccessRelay: 0,
	AuthRequestsSuccessLogin: 0,
	AuthRequestsRelay:        0,
	AuthRequestsLogin:        0,
	InternalErrors:           0,

	Entities: make(map[string]int32),
}
