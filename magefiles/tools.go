//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint"
	_ "mvdan.cc/gofumpt"
)
