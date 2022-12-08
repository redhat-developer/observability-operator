package configuration

import (
	"testing"

	. "github.com/onsi/gomega"
	v12 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiv1 "github.com/redhat-developer/observability-operator/v4/api/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testPrometheusRule = &v12.PrometheusRule{
		Spec: v12.PrometheusRuleSpec{
			Groups: []v12.RuleGroup{
				{
					Rules: []v12.Rule{
						{
							Labels: map[string]string{"test-key-1": "test-value-1", "test-key-2": "test-value-2"},
						},
					},
				},
			},
		},
	}

	testPrometheusRuleNoLabels = &v12.PrometheusRule{
		Spec: v12.PrometheusRuleSpec{
			Groups: []v12.RuleGroup{
				{
					Rules: []v12.Rule{
						{
							Labels: nil,
						},
					},
				},
			},
		},
	}
	prometheusRuleYaml = []byte(`metadata:
  name: test-name
  spec:
  `)
	invalidYaml = []byte(`invalid`)
)

func TestPrometheusRules_GetUniqueRules(t *testing.T) {
	type args struct {
		indexes []apiv1.RepositoryIndex
	}

	tests := []struct {
		name string
		args args
		want []ResourceInfo
	}{
		{
			name: "success creating ResourceInfo from repo",
			args: args{
				indexes: testRepoIndexes,
			},
			want: []ResourceInfo{
				{
					Id:          "test-id",
					Name:        "rule-1",
					Url:         "test-base-url/rule-1",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
				{
					Id:          "test-id",
					Name:        "rule-2",
					Url:         "test-base-url/rule-2",
					AccessToken: "test-access-token",
					Tag:         "test-tag",
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUniqueRules(tt.args.indexes)
			Expect(result).To(Equal(tt.want))
		})
	}
}

func TestPrometheusRules_InjectIdLabel(t *testing.T) {
	type args struct {
		rule *v12.PrometheusRule
		id   string
	}

	tests := []struct {
		name string
		args args
		want *v12.PrometheusRule
	}{
		{
			name: "success injecting Id label to exisitng labels",
			args: args{
				rule: testPrometheusRule,
				id:   "test-id",
			},
			want: &v12.PrometheusRule{
				Spec: v12.PrometheusRuleSpec{
					Groups: []v12.RuleGroup{
						{
							Rules: []v12.Rule{
								{
									Labels: map[string]string{"test-key-1": "test-value-1", "test-key-2": "test-value-2", "observability": "test-id"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "success injecting Id label when NO existing labels",
			args: args{
				rule: testPrometheusRuleNoLabels,
				id:   "test-id",
			},
			want: &v12.PrometheusRule{
				Spec: v12.PrometheusRuleSpec{
					Groups: []v12.RuleGroup{
						{
							Rules: []v12.Rule{
								{
									Labels: map[string]string{"observability": "test-id"},
								},
							},
						},
					},
				},
			},
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injectIdLabel(tt.args.rule, tt.args.id)
			Expect(tt.args.rule).To(Equal(tt.want))
		})
	}
}

func TestPrometheusRules_ParseRuleFromYaml(t *testing.T) {
	type args struct {
		cr     *apiv1.Observability
		name   string
		source []byte
	}

	tests := []struct {
		name    string
		args    args
		want    *v12.PrometheusRule
		wantErr bool
	}{
		{
			name: "success parsing Prometheus Rule from yaml",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-rule-name",
				source: prometheusRuleYaml,
			},
			want: &v12.PrometheusRule{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-rule-name",
					Namespace: "test-namespace",
				},
			},
		},
		{
			name: "error parsing Prometheus Rule from invalid yaml",
			args: args{
				cr:     buildObservabilityCR(nil),
				name:   "test-rule-name",
				source: invalidYaml,
			},
			want:    nil,
			wantErr: true,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRuleFromYaml(tt.args.cr, tt.args.name, tt.args.source)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(result).To(Equal(tt.want))
		})
	}
}
