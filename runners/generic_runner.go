package runners

import "sigs.k8s.io/controller-runtime/pkg/manager"

type OperandInitializer struct {
	cb func()
}

func NewOperandInitializer(cb func()) manager.Runnable {
	return &OperandInitializer{
		cb: cb,
	}
}

func (r *OperandInitializer) Start(<-chan struct{}) error {
	r.cb()
	return nil
}
