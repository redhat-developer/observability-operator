package configuration

import (
	"context"
	"crypto/sha256"
	"fmt"
	v1 "github.com/bf2fc6cc711aee1a0c2a/observability-operator/api/v1"
	"github.com/bf2fc6cc711aee1a0c2a/observability-operator/controllers/model"
	"io"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v14 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Get the namespaces in which this Promtail instance should scrape the logs from all pods
// Based on the label selectors in the index
func (r *Reconciler) getScrapeNamespacesFor(ctx context.Context, cr *v1.Observability, index *v1.RepositoryIndex) ([]string, error) {
	if index.Config == nil || index.Config.Promtail == nil || index.Config.Promtail.Enabled == false {
		return nil, nil
	}

	var result []string
	list := &v12.NamespaceList{}
	selector := labels.SelectorFromSet(index.Config.Promtail.NamespaceLabelSelector)
	opts := &client.ListOptions{
		LabelSelector: selector,
	}

	err := r.client.List(ctx, list, opts)
	if err != nil {
		return nil, err
	}

	for _, ns := range list.Items {
		result = append(result, ns.Name)
	}

	return result, nil
}

// Create an index-specific Promtail config
func (r *Reconciler) createPromtailConfigFor(ctx context.Context, cr *v1.Observability, index *v1.RepositoryIndex) (*v12.ConfigMap, []byte, error) {
	namespaces, err := r.getScrapeNamespacesFor(ctx, cr, index)
	if err != nil {
		return nil, nil, err
	}

	configMap := model.GetPromtailConfigmap(cr, index.Id)
	config, err := model.GetPromtailConfig(index.Config.Prometheus.Observatorium, cr.Status.ClusterID, index.Id, namespaces)

	_, err = controllerutil.CreateOrUpdate(ctx, r.client, configMap, func() error {
		configMap.Labels = map[string]string{
			"managed-by": "observability-operator",
		}
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data["promtail.yaml"] = config
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	hash := sha256.New()
	io.WriteString(hash, config)
	return configMap, hash.Sum(nil), nil
}

// If new indexes are added or existing indexes change their id, we have to cleanup the outdated daemonsets
func (r *Reconciler) deleteUnrequestedDaemonsets(ctx context.Context, cr *v1.Observability, indexes []v1.RepositoryIndex) error {
	list := &v13.DaemonSetList{}
	opts := &client.ListOptions{
		Namespace: cr.Namespace,
	}

	err := r.client.List(ctx, list, opts)
	if err != nil {
		return err
	}

	shouldExist := func(name string) bool {
		for _, index := range indexes {
			expectedName := fmt.Sprintf("promtail-%s", index.Id)
			if name == expectedName {
				if index.Config == nil || index.Config.Promtail == nil || index.Config.Promtail.Enabled == false {
					return false
				}
				return true
			}
		}
		return false
	}

	for _, daemonset := range list.Items {
		if !shouldExist(daemonset.Name) {
			err = r.client.Delete(ctx, &daemonset)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Create an index-specific daemonset
func (r *Reconciler) createPromtailDaemonsetFor(ctx context.Context, cr *v1.Observability, index *v1.RepositoryIndex) error {
	if index.Config == nil || index.Config.Promtail == nil || index.Config.Promtail.Enabled == false {
		return nil
	}

	daemonset := model.GetPromtailDaemonSet(cr, index.Id)
	sa := model.GetPromtailServiceAccount(cr)

	config, hash, err := r.createPromtailConfigFor(ctx, cr, index)
	if err != nil {
		return err
	}

	var t = true
	_, err = controllerutil.CreateOrUpdate(ctx, r.client, daemonset, func() error {
		daemonset.Labels = map[string]string{
			"managed-by": "observability-operator",
		}
		daemonset.Spec = v13.DaemonSetSpec{
			Selector: &v14.LabelSelector{
				MatchLabels: model.GetResourceLabels(),
			},
			Template: v12.PodTemplateSpec{
				ObjectMeta: v14.ObjectMeta{
					Labels: model.GetResourceLabels(),
				},
				Spec: v12.PodSpec{
					Affinity: &v12.Affinity{
						NodeAffinity: &v12.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v12.NodeSelector{
								NodeSelectorTerms: []v12.NodeSelectorTerm{
									{
										MatchExpressions: []v12.NodeSelectorRequirement{
											{
												Key:      "node-role.kubernetes.io/infra",
												Operator: "DoesNotExist",
											},
										},
									},
								},
							},
						},
					},
					ServiceAccountName: sa.Name,
					Volumes: []v12.Volume{
						{
							Name: "config",
							VolumeSource: v12.VolumeSource{
								ConfigMap: &v12.ConfigMapVolumeSource{
									LocalObjectReference: v12.LocalObjectReference{
										Name: config.Name,
									},
								},
							},
						},
						{
							Name: "token",
							VolumeSource: v12.VolumeSource{
								Secret: &v12.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-%s", index.Id, "observatorium-credentials"),
								},
							},
						},
						{
							Name: "logs",
							VolumeSource: v12.VolumeSource{
								HostPath: &v12.HostPathVolumeSource{
									Path: "/var/log/pods",
								},
							},
						},
					},
					Containers: []v12.Container{
						{
							Name:  "promtail",
							Image: "quay.io/integreatly/promtail:latest",

							SecurityContext: &v12.SecurityContext{
								Privileged: &t,
							},
							Env: []v12.EnvVar{
								{
									Name: "HOSTNAME",
									ValueFrom: &v12.EnvVarSource{
										FieldRef: &v12.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "CONFIG_HASH",
									Value: fmt.Sprintf("%x", hash),
								},
							},
							Args: []string{
								"-config.file=/opt/config/promtail.yaml",
							},
							VolumeMounts: []v12.VolumeMount{
								{
									Name:      "config",
									MountPath: "/opt/config",
								},
								{
									Name:      "token",
									MountPath: "/opt/secrets",
								},
								{
									Name:      "logs",
									MountPath: "/var/log/pods",
								},
							},
							Ports: []v12.ContainerPort{
								{
									ContainerPort: 3100,
									Protocol:      "TCP",
								},
							},
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: "File",
							ImagePullPolicy:          "Always",
						},
					},
				},
			},
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
