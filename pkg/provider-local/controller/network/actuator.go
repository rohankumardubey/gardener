// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package network

import (
	"context"

	"github.com/gardener/gardener/charts"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/common"
	"github.com/gardener/gardener/extensions/pkg/controller/network"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	localimagevector "github.com/gardener/gardener/pkg/provider-local/imagevector"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gardener/gardener/pkg/utils/managedresources"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const managedResourceName = "extension-networking-local"

type actuator struct {
	logger logr.Logger
	common.RESTConfigContext
}

// NewActuator creates a new Actuator that updates the status of the handled Network resources.
func NewActuator() network.Actuator {
	return &actuator{
		logger: log.Log.WithName("network-actuator"),
	}
}

func (a *actuator) Reconcile(ctx context.Context, network *extensionsv1alpha1.Network, cluster *extensionscontroller.Cluster) error {
	image, err := localimagevector.ImageVector().FindImage(charts.ImageNameKindnet, imagevector.TargetVersion(cluster.Shoot.Spec.Kubernetes.Version))
	if err != nil {
		return err
	}

	var (
		labels         = map[string]string{"app": "kindnet"}
		fileOrCreate   = corev1.HostPathFileOrCreate
		maxSurge       = intstr.FromInt(0)
		maxUnavailable = intstr.FromInt(1)

		registry = managedresources.NewRegistry(kubernetes.ShootScheme, kubernetes.ShootCodec, kubernetes.ShootSerializer)

		serviceAccount = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kindnet",
				Namespace: metav1.NamespaceSystem,
			},
		}

		clusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kindnet",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"nodes"},
					Verbs:     []string{"list", "watch"},
				},
			},
		}

		clusterRoleBinding = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "kindnet",
				Annotations: map[string]string{resourcesv1alpha1.DeleteOnInvalidUpdate: "true"},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRole.Name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      serviceAccount.Name,
				Namespace: serviceAccount.Namespace,
			}},
		}

		daemonSet = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kindnet",
				Namespace: metav1.NamespaceSystem,
				Labels:    labels,
			},
			Spec: appsv1.DaemonSetSpec{
				RevisionHistoryLimit: pointer.Int32(2),
				Selector:             &metav1.LabelSelector{MatchLabels: labels},
				UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
					Type: appsv1.RollingUpdateDaemonSetStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDaemonSet{
						MaxSurge:       &maxSurge,
						MaxUnavailable: &maxUnavailable,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Env: []corev1.EnvVar{
								{
									Name: "HOST_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.hostIP",
										},
									},
								},
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "POD_SUBNET",
									Value: network.Spec.PodCIDR,
								},
							},
							Name:            "kindnet-cni",
							Image:           image.String(),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointer.Bool(false),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_RAW", "NET_ADMIN"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cni-cfg",
									MountPath: "/etc/cni/net.d",
								},
								{
									Name:      "xtables-lock",
									MountPath: "/run/xtables.lock",
								},
								{
									Name:      "lib-modules",
									MountPath: "/lib/modules",
									ReadOnly:  true,
								},
							},
						}},
						HostNetwork:        true,
						ServiceAccountName: serviceAccount.Name,
						Tolerations:        []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
						Volumes: []corev1.Volume{
							{
								Name: "cni-cfg",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/etc/cni/net.d",
									},
								},
							},
							{
								Name: "xtables-lock",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/run/xtables.lock",
										Type: &fileOrCreate,
									},
								},
							},
							{
								Name: "lib-modules",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/lib/modules",
									},
								},
							},
						},
					},
				},
			},
		}
	)

	data, err := registry.AddAllAndSerialize(
		serviceAccount,
		clusterRole,
		clusterRoleBinding,
		daemonSet,
	)
	if err != nil {
		return err
	}

	return managedresources.CreateForShoot(ctx, a.Client(), network.Namespace, managedResourceName, true, data)

}

func (a *actuator) Delete(ctx context.Context, network *extensionsv1alpha1.Network, _ *extensionscontroller.Cluster) error {
	return managedresources.DeleteForShoot(ctx, a.Client(), network.Namespace, managedResourceName)
}

func (a *actuator) Migrate(ctx context.Context, network *extensionsv1alpha1.Network, cluster *extensionscontroller.Cluster) error {
	return a.Delete(ctx, network, cluster)
}

func (a *actuator) Restore(ctx context.Context, network *extensionsv1alpha1.Network, cluster *extensionscontroller.Cluster) error {
	return a.Reconcile(ctx, network, cluster)
}
