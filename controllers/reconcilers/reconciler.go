package reconcilers

import (
	"context"
	v1 "github.com/redhat-developer/observability-operator/v4/api/v1"
)

type ObservabilityReconciler interface {
	Reconcile(ctx context.Context, cr *v1.Observability, status *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error)
	Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error)
}
