package runners

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type OperandInitializer struct {
	cb func() error
}

func NewOperandInitializer(cb func() error) manager.Runnable {
	return &OperandInitializer{
		cb: cb,
	}
}

func (r *OperandInitializer) Start(context.Context) error {
	return r.cb()
}
