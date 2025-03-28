/*
Copyright 2019 The Volcano Authors.

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

package admission

import (
	"strings"
	"testing"

	kubebatchclient "github.com/kubernetes-sigs/kube-batch/pkg/client/clientset/versioned/fake"

	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kbv1aplha1 "github.com/kubernetes-sigs/kube-batch/pkg/apis/scheduling/v1alpha1"
	v1alpha1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

func TestValidateExecution(t *testing.T) {

	namespace := "test"
	var invTTL int32 = -1
	var policyExitCode int32 = -1

	testCases := []struct {
		Name           string
		Job            v1alpha1.Job
		ExpectErr      bool
		reviewResponse v1beta1.AdmissionResponse
		ret            string
	}{
		{
			Name: "validate valid-job",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-Job",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "",
			ExpectErr:      false,
		},
		// duplicate task name
		{
			Name: "duplicate-task-job",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-task-job",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "duplicated-task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
						{
							Name:     "duplicated-task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicated task name duplicated-task-1",
			ExpectErr:      true,
		},
		// Duplicated Policy Event
		{
			Name: "job-policy-duplicated",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-policy-duplicated",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.PodFailedEvent,
							Action: v1alpha1.AbortJobAction,
						},
						{
							Event:  v1alpha1.PodFailedEvent,
							Action: v1alpha1.RestartJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicate",
			ExpectErr:      true,
		},
		// Min Available illegal
		{
			Name: "Min Available illegal",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-min-illegal",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 2,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "'minAvailable' should not be greater than total replicas in tasks",
			ExpectErr:      true,
		},
		// Job Plugin illegal
		{
			Name: "Job Plugin illegal",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-plugin-illegal",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Plugins: map[string][]string{
						"big_plugin": {},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "unable to find job plugin: big_plugin",
			ExpectErr:      true,
		},
		// ttl-illegal
		{
			Name: "job-ttl-illegal",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-ttl-illegal",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					TTLSecondsAfterFinished: &invTTL,
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "'ttlSecondsAfterFinished' cannot be less than zero",
			ExpectErr:      true,
		},
		// min-MinAvailable less than zero
		{
			Name: "minAvailable-lessThanZero",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minAvailable-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: -1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "'minAvailable' cannot be less than zero.",
			ExpectErr:      true,
		},
		// maxretry less than zero
		{
			Name: "maxretry-lessThanZero",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "maxretry-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					MaxRetry:     -1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "'maxRetry' cannot be less than zero.",
			ExpectErr:      true,
		},
		// no task specified in the job
		{
			Name: "no-task",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-task",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks:        []v1alpha1.TaskSpec{},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "No task specified in job spec",
			ExpectErr:      true,
		},
		// replica set less than zero
		{
			Name: "replica-lessThanZero",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: -1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret:            "'replicas' is not set positive in task: task-1;",
			ExpectErr:      true,
		},
		// task name error
		{
			Name: "nonDNS-task",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-lessThanZero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "Task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: false},
			ret: "[a DNS-1123 label must consist of lower case alphanumeric characters or '-', and " +
				"must start and end with an alphanumeric character (e.g. 'my-name',  " +
				"or '123-abc', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?')];",
			ExpectErr: true,
		},
		// Policy Event with exit code
		{
			Name: "job-policy-withExitCode",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-policy-withExitCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:    v1alpha1.PodFailedEvent,
							Action:   v1alpha1.AbortJobAction,
							ExitCode: &policyExitCode,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "must not specify event and exitCode simultaneously",
			ExpectErr:      true,
		},
		// Both policy event and exit code are nil
		{
			Name: "policy-noEvent-noExCode",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "policy-noEvent-noExCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Action: v1alpha1.AbortJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "either event and exitCode should be specified",
			ExpectErr:      true,
		},
		// invalid policy event
		{
			Name: "invalid-policy-event",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-policy-event",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.Event("someFakeEvent"),
							Action: v1alpha1.AbortJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "invalid policy event",
			ExpectErr:      true,
		},
		// invalid policy action
		{
			Name: "invalid-policy-action",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-policy-action",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.PodEvictedEvent,
							Action: v1alpha1.Action("someFakeAction"),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "invalid policy action",
			ExpectErr:      true,
		},
		// policy exit-code zero
		{
			Name: "policy-extcode-zero",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "policy-extcode-zero",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Action: v1alpha1.AbortJobAction,
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(0)),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "0 is not a valid error code",
			ExpectErr:      true,
		},
		// duplicate policy exit-code
		{
			Name: "duplicate-exitcode",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-exitcode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(1)),
						},
						{
							ExitCode: func(i int32) *int32 {
								return &i
							}(int32(1)),
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "duplicate exitCode 1",
			ExpectErr:      true,
		},
		// Policy with any event and other events
		{
			Name: "job-policy-withExitCode",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-policy-withExitCode",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.AnyEvent,
							Action: v1alpha1.AbortJobAction,
						},
						{
							Event:  v1alpha1.PodFailedEvent,
							Action: v1alpha1.RestartJobAction,
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "if there's * here, no other policy should be here",
			ExpectErr:      true,
		},
		// invalid mount volume
		{
			Name: "invalid-mount-volume",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-mount-volume",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.AnyEvent,
							Action: v1alpha1.AbortJobAction,
						},
					},
					Volumes: []v1alpha1.VolumeSpec{
						{
							MountPath: "",
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            " mountPath is required;",
			ExpectErr:      true,
		},
		// duplicate mount volume
		{
			Name: "duplicate-mount-volume",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "duplicate-mount-volume",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
					Policies: []v1alpha1.LifecyclePolicy{
						{
							Event:  v1alpha1.AnyEvent,
							Action: v1alpha1.AbortJobAction,
						},
					},
					Volumes: []v1alpha1.VolumeSpec{
						{
							MountPath: "/var",
						},
						{
							MountPath: "/var",
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            " duplicated mountPath: /var;",
			ExpectErr:      true,
		},
		// task Policy with any event and other events
		{
			Name: "taskpolicy-withAnyandOthrEvent",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "taskpolicy-withAnyandOthrEvent",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "default",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
							Policies: []v1alpha1.LifecyclePolicy{
								{
									Event:  v1alpha1.AnyEvent,
									Action: v1alpha1.AbortJobAction,
								},
								{
									Event:  v1alpha1.PodFailedEvent,
									Action: v1alpha1.RestartJobAction,
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "if there's * here, no other policy should be here",
			ExpectErr:      true,
		},
		// job with no queue created
		{
			Name: "job-with-noQueue",
			Job: v1alpha1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job-with-noQueue",
					Namespace: namespace,
				},
				Spec: v1alpha1.JobSpec{
					MinAvailable: 1,
					Queue:        "jobQueue",
					Tasks: []v1alpha1.TaskSpec{
						{
							Name:     "task-1",
							Replicas: 1,
							Template: v1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: map[string]string{"name": "test"},
								},
								Spec: v1.PodSpec{
									Containers: []v1.Container{
										{
											Name:  "fake-name",
											Image: "busybox:1.24",
										},
									},
								},
							},
						},
					},
				},
			},
			reviewResponse: v1beta1.AdmissionResponse{Allowed: true},
			ret:            "Job not created with error: ",
			ExpectErr:      true,
		},
	}

	for _, testCase := range testCases {

		defaultqueue := kbv1aplha1.Queue{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: kbv1aplha1.QueueSpec{
				Weight: 1,
			},
		}
		// create fake kube-batch clientset
		KubeBatchClientSet = kubebatchclient.NewSimpleClientset()

		//create default queue
		_, err := KubeBatchClientSet.SchedulingV1alpha1().Queues().Create(&defaultqueue)
		if err != nil {
			t.Error("Queue Creation Failed")
		}

		ret := validateJob(testCase.Job, &testCase.reviewResponse)
		//fmt.Printf("test-case name:%s, ret:%v  testCase.reviewResponse:%v \n", testCase.Name, ret,testCase.reviewResponse)
		if testCase.ExpectErr == true && ret == "" {
			t.Errorf("%s: test case Expect error msg :%s, but got nil.", testCase.Name, testCase.ret)
		}
		if testCase.ExpectErr == true && testCase.reviewResponse.Allowed != false {
			t.Errorf("%s: test case Expect Allowed as false but got true.", testCase.Name)
		}
		if testCase.ExpectErr == true && !strings.Contains(ret, testCase.ret) {
			t.Errorf("%s: test case Expect error msg :%s, but got diff error %v", testCase.Name, testCase.ret, ret)
		}

		if testCase.ExpectErr == false && ret != "" {
			t.Errorf("%s: test case Expect no error, but got error %v", testCase.Name, ret)
		}
		if testCase.ExpectErr == false && testCase.reviewResponse.Allowed != true {
			t.Errorf("%s: test case Expect Allowed as true but got false. %v", testCase.Name, testCase.reviewResponse)
		}

	}

}
