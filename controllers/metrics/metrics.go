package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	apiv1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	LabelStage             = "stage"
	LabelConfigurationSync = "configuration_sync"
)

var reconciliationsLabels = []string{
	LabelStage,
}

var totalReconciliationsMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "reconciler_total_count",
		Subsystem: "observability_operator",
		Help:      "Total number of reconciliations performed",
	},
	reconciliationsLabels,
)

var failedReconciliationsMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "reconciler_failure_count",
		Subsystem: "observability_operator",
		Help:      "Number of failed reconciliations",
	},
	reconciliationsLabels,
)

var successfulConfigurationSyncsMetric = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name:      "configuration_syncs_total_count",
		Subsystem: "observability_operator",
		Help:      "Total number of configuration syncs performed",
	},
)

var failedConfigurationSyncsMetric = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name:      "configuration_sync_failure_count",
		Subsystem: "observability_operator",
		Help:      "Number of failed configuration syncs",
	},
)

func IncreaseTotalReconciliationsMetric(stage apiv1.ObservabilityStageName) {
	labels := prometheus.Labels{
		LabelStage: string(stage),
	}
	totalReconciliationsMetric.With(labels).Inc()
}

func IncreaseFailedReconciliationsMetric(stage apiv1.ObservabilityStageName) {
	labels := prometheus.Labels{
		LabelStage: string(stage),
	}
	failedReconciliationsMetric.With(labels).Inc()
}

func IncreaseSuccessfulConfigurationSyncsMetric() {
	successfulConfigurationSyncsMetric.Inc()
}

func IncreaseFailedConfigurationSyncsMetric() {
	failedConfigurationSyncsMetric.Inc()
}

func init() {
	metrics.Registry.MustRegister(totalReconciliationsMetric)
	metrics.Registry.MustRegister(failedReconciliationsMetric)
	metrics.Registry.MustRegister(successfulConfigurationSyncsMetric)
	metrics.Registry.MustRegister(failedConfigurationSyncsMetric)
}
