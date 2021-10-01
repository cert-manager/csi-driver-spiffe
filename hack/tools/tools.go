//go:build tools
// +build tools

// This file exists to force 'go mod' to fetch tool dependencies
// See: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package bin

import (
	_ "github.com/jetstack/cert-manager/cmd/ctl"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "sigs.k8s.io/kind"
)
