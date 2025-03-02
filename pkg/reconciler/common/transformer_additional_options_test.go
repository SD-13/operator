/*
Copyright 2023 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/pipeline/test/diff"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

func TestExecuteAdditionalOptionsTransformer(t *testing.T) {
	// context
	ctx := context.TODO()
	targetNamespace := "tekton-pipelines"

	// verify the changes applied on the manifest

	tcs := []struct {
		name                   string
		additionalOptions      v1alpha1.AdditionalOptions
		inputFilename          string
		expectedResultFilename string
	}{
		{
			name: "verify-disabled",
			additionalOptions: v1alpha1.AdditionalOptions{
				Disabled: true,
			},
			inputFilename:          "./testdata/test-additional-options-base.yaml",
			expectedResultFilename: "./testdata/test-additional-options-base.yaml",
		},
		{
			name: "test-configmap",
			additionalOptions: v1alpha1.AdditionalOptions{
				Disabled: false,
				ConfigMaps: map[string]corev1.ConfigMap{
					"config-defaults": {
						ObjectMeta: metav1.ObjectMeta{
							Labels:      map[string]string{"new-label": "foo"},
							Annotations: map[string]string{"custom-annotation": "hello"},
						},
						Data: map[string]string{
							"user-data1": "top-secret",
						},
					},
					"config-events": {
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"my-custom-label": "bar"},
						},
					},
					"config-tracing": {
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"component": "operator",
							},
							Annotations: map[string]string{
								"tracing-enabled": "true",
							},
						},
						Data: map[string]string{
							"hostname": "localhost",
							"port":     "14560",
						},
					},
				},
			},
			inputFilename:          "./testdata/test-additional-options-base.yaml",
			expectedResultFilename: "./testdata/test-additional-options-test-configmap.yaml",
		},
		{
			name:                   "test-empty-options",
			additionalOptions:      v1alpha1.AdditionalOptions{},
			inputFilename:          "./testdata/test-additional-options-base.yaml",
			expectedResultFilename: "./testdata/test-additional-options-base.yaml",
		},
		{
			name: "test-deployment",
			additionalOptions: v1alpha1.AdditionalOptions{
				Disabled: false,
				Deployments: map[string]appsv1.Deployment{
					"tekton-pipelines-controller": {
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"controlled-by-options": "true",
							},
							Annotations: map[string]string{
								"hpa-enabled": "false",
							},
						},
						Spec: appsv1.DeploymentSpec{
							Replicas: ptr.Int32(4),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									NodeSelector: map[string]string{
										"zone": "east",
									},
									Tolerations: []corev1.Toleration{
										{
											Key:      "zone",
											Operator: corev1.TolerationOpEqual,
											Value:    "west",
											Effect:   corev1.TaintEffectNoSchedule,
										},
									},
									TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
										{
											MaxSkew:           1,
											TopologyKey:       "kubernetes.io/hostname",
											WhenUnsatisfiable: corev1.DoNotSchedule,
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"app": "foo",
												},
											},
											MatchLabelKeys: []string{"pod-template-hash"},
										},
									},
									Affinity: &corev1.Affinity{
										NodeAffinity: &corev1.NodeAffinity{
											RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
												NodeSelectorTerms: []corev1.NodeSelectorTerm{
													{
														MatchExpressions: []corev1.NodeSelectorRequirement{
															{
																Key:      "disktype",
																Operator: "In",
																Values:   []string{"ssd", "nvme", "ramdisk"},
															},
														},
													},
												},
											},
										},
									},
									PriorityClassName: "test",
									Volumes: []corev1.Volume{
										{
											Name: "my-custom-logs",
											VolumeSource: corev1.VolumeSource{
												HostPath: &corev1.HostPathVolumeSource{
													Path: "/var/custom/logs",
												},
											},
										},
										{
											Name: "config-logging",
											VolumeSource: corev1.VolumeSource{
												HostPath: &corev1.HostPathVolumeSource{
													Path: "/etc/config-logging",
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name: "tekton-pipelines-controller",
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													corev1.ResourceCPU:    resource.MustParse("2"),
													corev1.ResourceMemory: resource.MustParse("4Gi"),
												},
											},
											Env: []corev1.EnvVar{
												{
													Name:  "ENV_FOO",
													Value: "bar",
												},
												{
													Name: "ENV_FROM_CONFIG_MAP",
													ValueFrom: &corev1.EnvVarSource{
														ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
															LocalObjectReference: corev1.LocalObjectReference{
																Name: "config-map-foo",
															},
															Key:      "foo",
															Optional: ptr.Bool(true),
														},
													},
												},
												{
													Name:  "CONFIG_LOGGING_NAME",
													Value: "pipeline-config-logging",
												},
											},
											Args: []string{
												"--disable-ha=false",
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "custom-mount",
													MountPath: "/etc/custom-mount",
												},
												{
													Name:      "config-logging",
													MountPath: "/etc/config-logging-tmp",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inputFilename:          "./testdata/test-additional-options-base.yaml",
			expectedResultFilename: "./testdata/test-additional-options-test-deployment.yaml",
		},
		{
			name: "empty-labels-and-annotations",
			additionalOptions: v1alpha1.AdditionalOptions{
				Disabled: false,
				ConfigMaps: map[string]corev1.ConfigMap{
					"config-defaults": {
						ObjectMeta: metav1.ObjectMeta{
							Labels:      map[string]string{},
							Annotations: map[string]string{},
						},
					},
				},
			},
			inputFilename:          "./testdata/test-additional-options-base.yaml",
			expectedResultFilename: "./testdata/test-additional-options-base.yaml",
		},
		{
			name: "test-statefulsets",
			additionalOptions: v1alpha1.AdditionalOptions{
				Disabled: false,
				StatefulSets: map[string]appsv1.StatefulSet{
					"web": {
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"controlled-by-options": "true",
							},
						},
						Spec: appsv1.StatefulSetSpec{
							Replicas:            ptr.Int32(4),
							ServiceName:         "www-n",
							PodManagementPolicy: appsv1.ParallelPodManagement,
							VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "www",
									},
									Spec: corev1.PersistentVolumeClaimSpec{
										AccessModes: []corev1.PersistentVolumeAccessMode{
											corev1.ReadWriteMany,
										},
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												"storage": resource.MustParse("2Gi"),
											},
										},
									},
								},
								{
									ObjectMeta: metav1.ObjectMeta{
										Name: "www-2",
									},
									Spec: corev1.PersistentVolumeClaimSpec{
										AccessModes: []corev1.PersistentVolumeAccessMode{
											corev1.ReadWriteMany,
										},
										Resources: corev1.ResourceRequirements{
											Requests: corev1.ResourceList{
												"storage": resource.MustParse("4Gi"),
											},
										},
									},
								},
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									NodeSelector: map[string]string{
										"zone": "east",
									},
									Tolerations: []corev1.Toleration{
										{
											Key:      "zone",
											Operator: corev1.TolerationOpEqual,
											Value:    "west",
											Effect:   corev1.TaintEffectNoSchedule,
										},
									},
									TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
										{
											MaxSkew:           1,
											TopologyKey:       "kubernetes.io/hostname",
											WhenUnsatisfiable: corev1.DoNotSchedule,
											LabelSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"app": "foo",
												},
											},
											MatchLabelKeys: []string{"pod-template-hash"},
										},
									},
									Affinity: &corev1.Affinity{
										NodeAffinity: &corev1.NodeAffinity{
											RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
												NodeSelectorTerms: []corev1.NodeSelectorTerm{
													{
														MatchExpressions: []corev1.NodeSelectorRequirement{
															{
																Key:      "disktype",
																Operator: "In",
																Values:   []string{"ssd", "nvme", "ramdisk"},
															},
														},
													},
												},
											},
										},
									},
									PriorityClassName: "test",
									Volumes: []corev1.Volume{
										{
											Name: "my-custom-logs",
											VolumeSource: corev1.VolumeSource{
												HostPath: &corev1.HostPathVolumeSource{
													Path: "/var/custom/logs",
												},
											},
										},
										{
											Name: "config-logging",
											VolumeSource: corev1.VolumeSource{
												HostPath: &corev1.HostPathVolumeSource{
													Path: "/etc/config-logging",
												},
											},
										},
									},
									Containers: []corev1.Container{
										{
											Name: "nginx",
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													corev1.ResourceCPU:    resource.MustParse("2"),
													corev1.ResourceMemory: resource.MustParse("4Gi"),
												},
											},
											Env: []corev1.EnvVar{
												{
													Name:  "ENV_FOO",
													Value: "bar",
												},
												{
													Name: "ENV_FROM_CONFIG_MAP",
													ValueFrom: &corev1.EnvVarSource{
														ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
															LocalObjectReference: corev1.LocalObjectReference{
																Name: "config-map-foo",
															},
															Key:      "foo",
															Optional: ptr.Bool(true),
														},
													},
												},
												{
													Name:  "CONFIG_LOGGING_NAME",
													Value: "pipeline-config-logging",
												},
											},
											Args: []string{
												"--mode=production",
											},
											VolumeMounts: []corev1.VolumeMount{
												{
													Name:      "custom-mount",
													MountPath: "/etc/custom-mount",
												},
												{
													Name:      "config-logging",
													MountPath: "/etc/config-logging-tmp",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			inputFilename:          "./testdata/test-additional-options-base-statefulsets.yaml",
			expectedResultFilename: "./testdata/test-additional-options-test-statefulsets.yaml",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// fetch base manifests
			targetManifest, err := Fetch(tc.inputFilename)
			assert.NoError(t, err)

			// fetch expected manifests
			expectedManifest, err := Fetch(tc.expectedResultFilename)
			assert.NoError(t, err)

			// execute with additional options transformer
			err = ExecuteAdditionalOptionsTransformer(ctx, &targetManifest, targetNamespace, tc.additionalOptions)
			assert.NoError(t, err)

			if d := cmp.Diff(expectedManifest.Resources(), targetManifest.Resources()); d != "" {
				t.Errorf("Diff %s", diff.PrintWantGot(d))
			}
		})
	}
}
