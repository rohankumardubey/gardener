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

package operatingsystemconfig

import (
	_ "embed"

	ostemplate "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig/oscommon/template"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"k8s.io/apimachinery/pkg/util/runtime"
)

var (
	//go:embed templates/cloud-init.template
	cloudInitTemplateString string
	cloudInitGenerator      *ostemplate.CloudInitGenerator
)

func init() {
	cloudInitTemplate, err := ostemplate.NewTemplate("cloud-init").Parse(cloudInitTemplateString)
	runtime.Must(err)

	cloudInitGenerator = ostemplate.NewCloudInitGenerator(
		cloudInitTemplate,
		ostemplate.DefaultUnitsPath,
		"/usr/bin/env bash %s",
		func(*extensionsv1alpha1.OperatingSystemConfig) (map[string]interface{}, error) {
			return nil, nil
		},
	)
}
