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

package shootextensionstatus

import (
	"context"

	"github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/apis/core/validation"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/storage/names"
)

type shootExtensionStatusStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy defines the storage strategy for ShootExtensionStatus resources.
var Strategy = shootExtensionStatusStrategy{api.Scheme, names.SimpleNameGenerator}

func (shootExtensionStatusStrategy) NamespaceScoped() bool {
	return true
}

func (shootExtensionStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	extensionStatus := obj.(*core.ShootExtensionStatus)

	extensionStatus.Generation = 1
}

func (shootExtensionStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newShootExtensionStatus := obj.(*core.ShootExtensionStatus)
	oldShootExtensionStatus := old.(*core.ShootExtensionStatus)

	if mustIncreaseGeneration(oldShootExtensionStatus, newShootExtensionStatus) {
		newShootExtensionStatus.Generation = oldShootExtensionStatus.Generation + 1
	}
}

func mustIncreaseGeneration(oldShootExtensionStatus, newShootExtensionStatus *core.ShootExtensionStatus) bool {
	// The ShootState specification changes.
	if !apiequality.Semantic.DeepEqual(oldShootExtensionStatus.Statuses, newShootExtensionStatus.Statuses) {
		return true
	}

	// The deletion timestamp was set.
	if oldShootExtensionStatus.DeletionTimestamp == nil && newShootExtensionStatus.DeletionTimestamp != nil {
		return true
	}

	return false
}

func (shootExtensionStatusStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	shootState := obj.(*core.ShootExtensionStatus)
	return validation.ValidateShootExtensionStatus(shootState)
}

func (shootExtensionStatusStrategy) Canonicalize(obj runtime.Object) {
}

func (shootExtensionStatusStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (shootExtensionStatusStrategy) ValidateUpdate(ctx context.Context, newObj, oldObj runtime.Object) field.ErrorList {
	newExtensionStatus := newObj.(*core.ShootExtensionStatus)
	oldExtensionStatus := oldObj.(*core.ShootExtensionStatus)
	return validation.ValidateShootExtensionStatusUpdate(newExtensionStatus, oldExtensionStatus)
}

func (shootExtensionStatusStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// WarningsOnCreate returns warnings to the client performing a create.
func (shootExtensionStatusStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate returns warnings to the client performing the update.
func (shootExtensionStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
