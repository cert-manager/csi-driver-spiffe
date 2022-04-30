/*
Copyright 2021 The cert-manager Authors.

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

package fake

import (
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"

	"github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator"
)

var _ evaluator.Interface = &FakeEvaluator{}

// FakeEvaluator is a implementation of Evaluator which can be mocked for testing.
type FakeEvaluator struct {
	funcEvaluate func(*cmapi.CertificateRequest) error
}

// New returns a new FakeEvaluator
func New() *FakeEvaluator {
	return &FakeEvaluator{
		funcEvaluate: func(_ *cmapi.CertificateRequest) error {
			return nil
		},
	}
}

func (f *FakeEvaluator) WithEvaluate(fn func(_ *cmapi.CertificateRequest) error) *FakeEvaluator {
	f.funcEvaluate = fn
	return f
}

func (f *FakeEvaluator) Evaluate(req *cmapi.CertificateRequest) error {
	return f.funcEvaluate(req)
}
