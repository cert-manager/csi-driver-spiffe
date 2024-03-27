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

package framework

import (
	"context"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cert-manager/csi-driver-spiffe/test/e2e/framework/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Framework struct {
	BaseName  string
	Namespace corev1.Namespace

	client client.Client
	config *config.Config

	ctx    context.Context
	cancel func()
}

func NewDefaultFramework(baseName string) *Framework {
	return NewFramework(baseName, config.GetConfig())
}

func NewFramework(baseName string, config *config.Config) *Framework {
	f := &Framework{
		BaseName: baseName,
		config:   config,
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

func (f *Framework) BeforeEach() {
	f.ctx, f.cancel = context.WithTimeout(context.Background(), time.Second*600)

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(rbacv1.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(cmapi.AddToScheme(scheme)).NotTo(HaveOccurred())

	var err error
	By("Creating a Kubernetes client")
	f.client, err = client.New(f.config.RestConfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	By("Creating test Namespace")
	f.Namespace = corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "test-csi-driver-spiffe-"}}
	Expect(f.client.Create(f.ctx, &f.Namespace)).NotTo(HaveOccurred())
}

func (f *Framework) AfterEach() {
	Expect(f.client.Delete(f.ctx, &f.Namespace)).NotTo(HaveOccurred())
	f.cancel()
}

func (f *Framework) Config() *config.Config {
	return f.config
}

func (f *Framework) Client() client.Client {
	return f.client
}

func (f *Framework) Context() context.Context {
	return f.ctx
}

func CasesDescribe(text string, body func()) bool {
	return Describe("[cert-manager-csi-driver-spiffe] "+text, body)
}
