package reconcilers

import (
	"context"
	v1 "github.com/jeremyary/observability-operator/api/v1"
)

type ObservabilityReconciler interface {
	Reconcile(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error)
	Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error)
}
