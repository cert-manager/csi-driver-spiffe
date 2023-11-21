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

package app

import (
	"context"
	"fmt"

	"github.com/cert-manager/cert-manager/pkg/api"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/scale/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/cert-manager/csi-driver-spiffe/internal/approver/app/options"
	"github.com/cert-manager/csi-driver-spiffe/internal/approver/controller"
	"github.com/cert-manager/csi-driver-spiffe/internal/approver/evaluator"
)

const (
	helpOutput = "A cert-manager Approver that is paired with a cert-manager SPIFFE CSI driver"
)

var intscheme = runtime.NewScheme()

func init() {
	utilruntime.Must(scheme.AddToScheme(intscheme))
	utilruntime.Must(api.AddToScheme(intscheme))
}

// NewCommand returns an new command instance of the approver component of csi-driver-spiffe.
func NewCommand(ctx context.Context) *cobra.Command {
	opts := options.New()

	cmd := &cobra.Command{
		Use:   "csi-driver-spiffe-approver",
		Short: helpOutput,
		Long:  helpOutput,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Complete()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log := opts.Logr.WithName("main")

			mgr, err := ctrl.NewManager(opts.RestConfig, ctrl.Options{
				Scheme:                        intscheme,
				LeaderElection:                true,
				LeaderElectionNamespace:       opts.Controller.LeaderElectionNamespace,
				LeaderElectionID:              opts.DriverName,
				LeaderElectionReleaseOnCancel: true,
				LeaderElectionResourceLock:    "leases",
				ReadinessEndpointName:         "/readyz",
				HealthProbeBindAddress:        opts.Controller.ReadyzAddress,
				Metrics: server.Options{
					BindAddress: opts.Controller.MetricsAddress,
				},
				Logger: opts.Logr.WithName("manager"),
			})
			if err != nil {
				return fmt.Errorf("failed to create manager: %w", err)
			}
			if err := mgr.AddReadyzCheck("manager", healthz.Ping); err != nil {
				return err
			}

			evaluator := evaluator.New(evaluator.Options{
				TrustDomain:                opts.CertManager.TrustDomain,
				CertificateRequestDuration: opts.CertManager.CertificateRequestDuration,
			})

			if err := controller.AddApprover(ctx, opts.Logr, controller.Options{
				IssuerRef: opts.CertManager.IssuerRef,
				Evaluator: evaluator,
				Manager:   mgr,
			}); err != nil {
				return fmt.Errorf("failed to register approver controller: %w", err)
			}

			log.Info("starting SPIFFE approver...")

			return mgr.Start(ctx)
		},
	}

	opts.Prepare(cmd)

	return cmd
}
