package token

import (
	_ "github.com/jeremyary/observability-operator/api/v1"
	v1 "github.com/jeremyary/observability-operator/api/v1"
)

type AuthTokenFetcher interface {
	Fetch(cr *v1.Observability) (string, error)
}

type NilTokenFetcher struct{}

type DexTokenFetcher struct{}

func GetTokenFetcher(cr *v1.Observability) AuthTokenFetcher {
	if cr.Spec.Observatorium == nil {
		return NewNilTokenFetcher()
	}

	switch cr.Spec.Observatorium.AuthType {
	case v1.AuthTypeDex:
		return NewDexTokenFetcher()
	default:
		return NewNilTokenFetcher()
	}
}

func NewNilTokenFetcher() AuthTokenFetcher {
	return &NilTokenFetcher{}
}

func (r *NilTokenFetcher) Fetch(*v1.Observability) (string, error) {
	return "", nil
}

func NewDexTokenFetcher() AuthTokenFetcher {
	return &DexTokenFetcher{}
}

func (r *DexTokenFetcher) Fetch(cr *v1.Observability) (string, error) {
	return "", nil
}
