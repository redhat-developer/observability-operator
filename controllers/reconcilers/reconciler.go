package reconcilers

import (
	"context"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/v3/api/v1"
)

type ObservabilityReconciler interface {
	Reconcile(ctx context.Context, cr *v1.Observability, status *v1.ObservabilityStatus) (v1.ObservabilityStageStatus, error)
	Cleanup(ctx context.Context, cr *v1.Observability) (v1.ObservabilityStageStatus, error)
}
