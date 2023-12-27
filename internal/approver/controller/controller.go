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

package controller

import (
	"context"
	"os"

	apiutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator"
)

type Options struct {
	// IssuerRef will be used to match against CertificateRequest that need
	// evaluation.
	IssuerRef cmmeta.ObjectReference

	// Evaluator will be used to evaluate whether CertificateRequests should be
	// Approved or Denied.
	Evaluator evaluator.Interface

	// Manager is a controller-runtime Manager that the Approver controller will
	// be registered against.
	Manager manager.Manager
}

// approver watches for CertificateRequests which have been created by the
// SPIFFE CSI driver and evaluates whether they should be approved or denied.
type approver struct {
	// log is logger for the approver controller.
	log logr.Logger

	// client is a client for interacting with the Kubernetes API.
	client client.Client

	// lister makes requests to the informer cache for getting and listing
	// objects.
	lister client.Reader

	// issuerRef is the issuerRef that will be matched on CertificateRequests for
	// evaluation.
	issuerRef cmmeta.ObjectReference

	// evaluator evaluates matched CertificateRequests for whether they should be
	// approved or denied.
	evaluator evaluator.Interface
}

// AddApprover will register the approver controller.
func AddApprover(ctx context.Context, log logr.Logger, opts Options) error {
	a := &approver{
		log:       log.WithName("controller"),
		client:    opts.Manager.GetClient(),
		lister:    opts.Manager.GetCache(),
		issuerRef: opts.IssuerRef,
		evaluator: opts.Evaluator,
	}

	return ctrl.NewControllerManagedBy(opts.Manager).
		For(new(cmapi.CertificateRequest)).
		WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			var req cmapi.CertificateRequest
			err := a.lister.Get(ctx, client.ObjectKeyFromObject(obj), &req)
			if apierrors.IsNotFound(err) {
				// Ignore CertificateRequests that have been deleted.
				return false
			}

			// If an error happens here and we do nothing, we run the risk of not
			// processing CertificateRequests.
			// Exiting error is the safest option, as it will force a resync on all
			// CertificateRequests on start.
			if err != nil {
				a.log.Error(err, "failed to list all CertificateRequests, exiting error")
				os.Exit(-1)
			}

			// Ignore requests that already have an Approved or Denied condition.
			if apiutil.CertificateRequestIsApproved(&req) || apiutil.CertificateRequestIsDenied(&req) {
				return false
			}

			return req.Spec.IssuerRef == a.issuerRef
		})).
		Complete(a)
}

// Reconcile is called when a CertificateRequest is synced which has been
// neither approved or denied yet, and matches the issuerRef configured.
func (a *approver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := a.log.WithValues("namespace", req.NamespacedName.Namespace, "name", req.NamespacedName.Name)
	log.V(2).Info("syncing certificaterequest")
	defer log.V(2).Info("finished syncing certificaterequest")

	var cr cmapi.CertificateRequest
	if err := a.lister.Get(ctx, req.NamespacedName, &cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := a.evaluator.Evaluate(&cr); err != nil {
		log.Error(err, "denying request")
		apiutil.SetCertificateRequestCondition(&cr, cmapi.CertificateRequestConditionDenied, cmmeta.ConditionTrue, "spiffe.csi.cert-manager.io", "Denied request: "+err.Error())
		return ctrl.Result{}, a.client.Status().Update(ctx, &cr)
	}

	log.Info("approving request")
	apiutil.SetCertificateRequestCondition(&cr, cmapi.CertificateRequestConditionApproved, cmmeta.ConditionTrue, "spiffe.csi.cert-manager.io", "Approved request")
	return ctrl.Result{}, a.client.Status().Update(ctx, &cr)
}
