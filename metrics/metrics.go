package metrics

import (
	"github.com/rs/zerolog/log"
)

func (env *MetricsStruct) Inc(metricName string, increment int32) bool {

	log.Debug().
		Str("metricName", metricName).
		Int32("increment", increment).
		Interface("metricCurrentValue", env.Entities[metricName]).
		Msgf("Incrementing metric")

	env.IncrementMutex.Lock()
	env.Entities[metricName] += increment
	env.IncrementMutex.Unlock()

	log.Debug().
		Str("metricName", metricName).
		Int32("increment", increment).
		Interface("metricCurrentValue", env.Entities[metricName]).
		Msgf("Metric has been incremented")

	return true
}

func (env *MetricsStruct) Error(errorSource string, increment int32) bool {

	log.Debug().
		Str("errorSource", errorSource).
		Int32("increment", increment).
		Int32("Errors", env.Errors).
		Msgf("Incrementing the errors counter")

	env.IncrementMutex.Lock()
	env.Errors += increment
	env.IncrementMutex.Unlock()

	log.Debug().
		Str("errorSource", errorSource).
		Int32("increment", increment).
		Int32("Errors", env.Errors).
		Msgf("Errors counter has been incremented")

	return true
}

func (env *MetricsStruct) Warning(warningSource string, increment int32) bool {

	log.Debug().
		Str("warningSource", warningSource).
		Int32("increment", increment).
		Int32("Warnings", env.Warnings).
		Msgf("Incrementing the errors counter")

	env.IncrementMutex.Lock()
	env.Warnings += increment
	env.IncrementMutex.Unlock()

	log.Debug().
		Str("warningSource", warningSource).
		Int32("increment", increment).
		Int32("Warnings", env.Warnings).
		Msgf("Warnings counter has been incremented")

	return true
}
