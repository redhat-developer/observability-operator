package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	apiv1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	LabelStage = "stage"
)

var reconciliationsLabels = []string{
	"stage",
}

var totalReconciliationsMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "observability_operator",
		Subsystem: "reconciler",
		Help:      "Total number of reconciliations performed",
	},
	reconciliationsLabels,
)

var failedReconciliationsMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "observability_operator",
		Subsystem: "reconciler",
		Help:      "Number of failed reconciliations",
	},
	reconciliationsLabels,
)

func init() {
	metrics.Registry.MustRegister(totalReconciliationsMetric)
	metrics.Registry.MustRegister(failedReconciliationsMetric)
}

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
