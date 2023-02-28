package configuration

import v1 "github.com/redhat-developer/observability-operator/v4/api/v1"

const DefaultOriginOauthProxyImage = "quay.io/openshift/origin-oauth-proxy:4.9"

func GetOriginOauthProxyImage(cr *v1.Observability) string {
	if cr.Spec.SelfContained != nil && cr.Spec.SelfContained.OriginOauthProxyImage != "" {
		return cr.Spec.SelfContained.OriginOauthProxyImage
	}

	return DefaultOriginOauthProxyImage
}
