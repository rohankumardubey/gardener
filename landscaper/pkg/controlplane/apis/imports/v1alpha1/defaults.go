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

package v1alpha1

import (
	"fmt"

	"github.com/gardener/gardener/pkg/scheduler/apis/config/encoding"
	schedulerconfigv1alpha1 "github.com/gardener/gardener/pkg/scheduler/apis/config/v1alpha1"
	landscaperv1alpha1 "github.com/gardener/landscaper/apis/core/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Imports sets defaults for the configuration of the ControlPlane component.
func SetDefaults_Imports(obj *Imports) {
	if obj.Rbac != nil &&
		obj.Rbac.SeedAuthorizer != nil &&
		obj.Rbac.SeedAuthorizer.Enabled != nil &&
		*obj.Rbac.SeedAuthorizer.Enabled &&
		obj.GardenerAdmissionController != nil &&
		obj.GardenerAdmissionController.Enabled &&
		obj.GardenerAdmissionController.SeedRestriction == nil {
		obj.GardenerAdmissionController.SeedRestriction = &SeedRestriction{Enabled: true}
	}

	// initialise empty as we anyways need to generate certificates
	if obj.GardenerControllerManager == nil {
		obj.GardenerControllerManager = &GardenerControllerManager{
			ComponentConfiguration: &ControllerManagerComponentConfiguration{},
		}
	}

	// initialise empty as we anyways need to generate certificates
	if obj.GardenerAdmissionController == nil {
		obj.GardenerAdmissionController = &GardenerAdmissionController{}
	}

	if obj.GardenerAPIServer.ComponentConfiguration.Admission != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.MutatingWebhook != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.MutatingWebhook.TokenProjection != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.MutatingWebhook.TokenProjection.Enabled &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.MutatingWebhook.Kubeconfig == nil {
		obj.GardenerAPIServer.ComponentConfiguration.Admission.MutatingWebhook.Kubeconfig = &landscaperv1alpha1.Target{
			Spec: landscaperv1alpha1.TargetSpec{
				Configuration: landscaperv1alpha1.AnyJSON{
					RawMessage: []byte(getVolumeProjectionKubeconfig("mutating")),
				},
			},
		}
	}

	if obj.GardenerAPIServer.ComponentConfiguration.Admission != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.ValidatingWebhook != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.ValidatingWebhook.TokenProjection != nil &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.ValidatingWebhook.TokenProjection.Enabled &&
		obj.GardenerAPIServer.ComponentConfiguration.Admission.ValidatingWebhook.Kubeconfig == nil {
		obj.GardenerAPIServer.ComponentConfiguration.Admission.ValidatingWebhook.Kubeconfig = &landscaperv1alpha1.Target{
			Spec: landscaperv1alpha1.TargetSpec{
				Configuration: landscaperv1alpha1.AnyJSON{
					RawMessage: []byte(getVolumeProjectionKubeconfig("validating")),
				},
			},
		}
	}
}

func getVolumeProjectionKubeconfig(name string) string {
	return fmt.Sprintf(`
---
apiVersion: v1
kind: Config
users:
- name: '*'
user:
  tokenFile: /var/run/secrets/admission-tokens/%s-webhook-token`, name)
}

// SetDefaults_GardenerScheduler sets the default values for the Gardener scheduler configuration
// in order to pass the validation
func SetDefaults_GardenerScheduler(obj *GardenerScheduler) {
	if obj.ComponentConfiguration == nil || obj.ComponentConfiguration.Config.Object == nil && len(obj.ComponentConfiguration.Config.Raw) == 0 {
		obj.ComponentConfiguration = &SchedulerComponentConfiguration{
			Config: runtime.RawExtension{
				Object: &schedulerconfigv1alpha1.SchedulerConfiguration{},
			},
		}
	}

	schedulerConfig, err := encoding.DecodeSchedulerConfiguration(&obj.ComponentConfiguration.Config, false)
	if err != nil {
		return
	}

	SetDefaultsSchedulerComponentConfiguration(schedulerConfig)

	obj.ComponentConfiguration.Config = runtime.RawExtension{Object: schedulerConfig}
}

// SetDefaultsSchedulerComponentConfiguration sets defaults for the Scheduler component configuration for the Landscaper imports
// we can safely assume that the configuration is not nil, as the encoding made that sure
func SetDefaultsSchedulerComponentConfiguration(config *schedulerconfigv1alpha1.SchedulerConfiguration) {
	// setup the scheduler with the minimal distance strategy
	if config.Schedulers.Shoot == nil {
		config.Schedulers.Shoot = &schedulerconfigv1alpha1.ShootSchedulerConfiguration{
			Strategy: schedulerconfigv1alpha1.MinimalDistance,
		}
	}
}
