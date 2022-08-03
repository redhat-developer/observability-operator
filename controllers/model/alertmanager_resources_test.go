package model

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"testing"

	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"

	v14 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultAlertManagerName              = "kafka-alertmanager"
	defaultAlertmanagerObjectMeta        = metav1.ObjectMeta{Name: defaultAlertManagerName}
	alertmanagerTestName                 = "test-alert-manager-name"
	alertmanagerServiceAccountAnnotation = map[string]string{"serviceaccounts.openshift.io/oauth-redirectreference.primary": "{\"kind\":\"OAuthRedirectReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"Route\",\"name\":\"kafka-alertmanager\"}}"}
	testResourceList                     = map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: resource.MustParse("10Gi")}
)

func buildObservabilityCR(modifyFn func(obsCR *v1.Observability)) *v1.Observability {
	obsCR := &v1.Observability{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		},
		Spec: v1.ObservabilitySpec{
			AlertManagerDefaultName: "",
			SelfContained:           nil,
		},
	}
	if modifyFn != nil {
		modifyFn(obsCR)
	}
	return obsCR
}

func TestAlertManagerResources_GetDefaultNameAlertmanager(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{

			name: "return cr AlertManagerDefaultName if self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.AlertManagerDefaultName = alertmanagerTestName
				}),
			},
			want: alertmanagerTestName,
		},
		{
			name: "return `kafka-alertmanager` if NOT self contained and spec AlertManagerDefaultName is empty",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: defaultAlertManagerName,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDefaultNameAlertmanager(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerProxySecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "return alert manager proxy secret with cr namespace",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alertmanager-proxy",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerProxySecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerTLSSecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "return alert manager tls secret with cr namespace",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alertmanager-k8s-tls",
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerTLSSecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerRoute(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *routev1.Route
	}{
		{
			name: "return alert manager service account with cr namespace",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &routev1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultAlertManagerName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerRoute(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerServiceAccount(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.ServiceAccount
	}{
		{
			name: "return alert manager service account with cr namespace",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        defaultAlertManagerName,
					Namespace:   testNamespace,
					Annotations: alertmanagerServiceAccountAnnotation,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerServiceAccount(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerClusterRole(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *v14.ClusterRole
	}{
		{
			name: "return alert manager ClusterRole",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRole{
				ObjectMeta: defaultAlertmanagerObjectMeta,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerClusterRole(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerClusterRoleBinding(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *v14.ClusterRoleBinding
	}{
		{
			name: "return alert manager ClusterRoleBinding",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &v14.ClusterRoleBinding{
				ObjectMeta: defaultAlertmanagerObjectMeta,
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerClusterRoleBinding(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerCr(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *monitoringv1.Alertmanager
	}{
		{
			name: "return alert manager cr",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &monitoringv1.Alertmanager{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultAlertManagerName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerCr(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerSecret(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.Secret
	}{
		{
			name: "return alert manager cr",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "alertmanager-" + defaultAlertManagerName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerSecret(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerSecretName(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return cr AlertManagerConfigSecret if self contained and AlertManagerConfigSecret not empty",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						AlertManagerConfigSecret: "test",
					}
				}),
			},
			want: "test",
		},
		{
			name: "return default secret name if NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: "alertmanager-" + defaultAlertManagerName,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerSecretName(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerService(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.Service
	}{
		{
			name: "return alert manager service with cr namespace",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultAlertManagerName,
					Namespace: testNamespace,
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerService(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerVersion(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "return alert manager version from cr when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						AlertManagerVersion: "test-version",
					}
				}),
			},
			want: "test-version",
		},
		{
			name: "return empty when NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: "",
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerVersion(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestAlertManagerResources_GetAlertmanagerResourceRequirement(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}
	tests := []struct {
		name string
		args args
		want *corev1.ResourceRequirements
	}{
		{
			name: "return AlertManagerResourceRequirement from cr when self contained",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{
						AlertManagerResourceRequirement: &corev1.ResourceRequirements{
							Limits: testResourceList,
						},
					}
				}),
			},
			want: &corev1.ResourceRequirements{
				Limits: testResourceList,
			},
		},
		{
			name: "return empty ResourceRequirement when NOT self contained",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: &corev1.ResourceRequirements{},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerResourceRequirement(tt.args.cr)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestGetAlertManagerStorageSize(t *testing.T) {
	type args struct {
		cr      *v1.Observability
		indexes []v1.RepositoryIndex
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "cr storage is used when selfcontained is specified AND a storage value is provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						AlertManagerStorageSpec: &monitoringv1.StorageSpec{
							VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
								Spec: corev1.PersistentVolumeClaimSpec{
									Resources: corev1.ResourceRequirements{
										Requests: testResourceList,
									},
								},
							},
						},
					}
				}),
			},
			want: "10Gi",
		},
		{
			name: "empty string returned when selfcontained is NOT specified AND NO storage value is provided",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: "",
		},
		{
			name: "no nil failure when selfcontained is specified AND a storage value is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
				}),
			},
			want: "",
		},
		{
			name: "no nil failure when selfcontained is specified AND PersistentVolumeClaim is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						AlertManagerStorageSpec: &monitoringv1.StorageSpec{
							VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
								Spec: corev1.PersistentVolumeClaimSpec{},
							},
						},
					}
				}),
			},
			want: "",
		},
		{
			name: "no nil failure when selfcontained is specified AND EmbeddedPersistentVolumeClaim is NOT provided",
			args: args{
				cr: buildObservabilityCR(func(obsCR *v1.Observability) {
					obsCR.Spec.SelfContained = &v1.SelfContained{}
					obsCR.Spec.Storage = &v1.Storage{
						AlertManagerStorageSpec: &monitoringv1.StorageSpec{},
					}
				}),
			},
			want: "",
		},
		{
			name: "returns repo PVC override size if NOT self contained and OverrideAlertmanagerPvcSize is not empty",
			args: args{
				cr:      buildObservabilityCR(nil),
				indexes: testRepoIndexes,
			},
			want: "test-quantity",
		},
	}

	g := NewWithT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAlertmanagerStorageSize(tt.args.cr, tt.args.indexes)
			g.Expect(result).To(Equal(tt.want))
		})
	}
}
