package runners

import "sigs.k8s.io/controller-runtime/pkg/manager"

type OperandInitializer struct {
	cb func() error
}

func NewOperandInitializer(cb func() error) manager.Runnable {
	return &OperandInitializer{
		cb: cb,
	}
}

func (r *OperandInitializer) Start(<-chan struct{}) error {
	return r.cb()
}
