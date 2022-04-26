//go:build tools
// +build tools

package tools

import (
	// Adds targets for makefile
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/openshift/build-machinery-go"
)
