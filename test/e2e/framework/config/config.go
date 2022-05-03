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

package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	sharedConfig = &Config{}
)

func SetConfig(config *Config) {
	sharedConfig = config
}

func GetConfig() *Config {
	return sharedConfig
}

type Config struct {
	kubeConfig string

	IssuerRef             cmmeta.ObjectReference
	IssuerSecretNamespace string
	IssuerSecretName      string
	RestConfig            *rest.Config
	KubectlBinPath        string
}

func (c *Config) AddFlags(fs *flag.FlagSet) *Config {
	return c.addFlags(fs)
}

func (c *Config) Complete() error {
	if c.kubeConfig == "" {
		return errors.New("--kubeconfig-path must be specified")
	}

	if c.KubectlBinPath == "" {
		return errors.New("--kubectl-path must be specified")
	}

	var err error
	c.RestConfig, err = clientcmd.BuildConfigFromFlags("", c.kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to build kubernetes rest config: %s", err)
	}

	return nil
}

func (c *Config) addFlags(fs *flag.FlagSet) *Config {
	kubeConfigFile := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if kubeConfigFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("Failed to get user home directory: " + err.Error())
		}
		kubeConfigFile = filepath.Join(homeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
	}

	fs.StringVar(&c.kubeConfig, "kubeconfig-path", kubeConfigFile, "Path to config containing embedded authinfo for kubernetes. Default value is from environment variable "+clientcmd.RecommendedConfigPathEnvVar)
	fs.StringVar(&c.KubectlBinPath, "kubectl-path", "", "Path to a authenticated kubectl binary")
	fs.StringVar(&c.IssuerRef.Name, "issuer-name", "csi-driver-spiffe-ca", "Name of issuer which has been created for the test")
	fs.StringVar(&c.IssuerRef.Kind, "issuer-kind", "ClusterIssuer", "Kind of issuer which has been created for the test")
	fs.StringVar(&c.IssuerRef.Group, "issuer-group", "cert-manager.io", "Group of issuer which has been created for the test")
	fs.StringVar(&c.IssuerSecretName, "issuer-secret-name", "csi-driver-spiffe-ca", "Name of the CA certificate Secret")
	fs.StringVar(&c.IssuerSecretNamespace, "issuer-secret-namespace", "cert-manager", "Namespace where the CA certificate Secret is stored")
	return c
}
