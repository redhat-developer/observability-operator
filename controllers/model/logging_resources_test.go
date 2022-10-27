package model

import (
	"testing"

	. "github.com/onsi/gomega"
	v1 "github.com/redhat-developer/observability-operator/v3/api/v1"
)

var (
	loggingName           = "cluster-logging"
	loggingNamespace      = "openshift-logging"
	logForwarderName      = "instance"
	logForwarderNamespace = "openshift-logging"
	loggingCRName         = "instance"
	loggingCRNamespace    = "openshift-logging"
)

func TestLoggingResources_GetLoggingSubscriptionName(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns correct subscription name",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: loggingName,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLoggingSubscription(tt.args.cr)
			Expect(result.Name).To(Equal(tt.want))
		})
	}
}

func TestLoggingResources_GetLoggingSubscriptionNamespace(t *testing.T) {
	type args struct {
		cr *v1.Observability
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "returns correct subscription name",
			args: args{
				cr: buildObservabilityCR(nil),
			},
			want: loggingNamespace,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLoggingSubscription(tt.args.cr)
			Expect(result.Namespace).To(Equal(tt.want))
		})
	}
}

func TestLoggingResources_GetLogForwarderName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "returns correct log forwarder name",
			want: logForwarderName,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClusterLogForwarderCR()
			Expect(result.Name).To(Equal(tt.want))
		})
	}
}

func TestLoggingResources_GetLogForwarderNamespace(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "returns correct log forwarder name",
			want: logForwarderNamespace,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClusterLogForwarderCR()
			Expect(result.Namespace).To(Equal(tt.want))
		})
	}
}

func TestLoggingResources_GetLoggingName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "returns correct logging name",
			want: loggingCRName,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClusterLoggingCR()
			Expect(result.Name).To(Equal(tt.want))
		})
	}
}

func TestLoggingResources_GetLoggingNamespace(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "returns correct logging name",
			want: loggingCRNamespace,
		},
	}

	RegisterTestingT(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetClusterLoggingCR()
			Expect(result.Namespace).To(Equal(tt.want))
		})
	}
}
