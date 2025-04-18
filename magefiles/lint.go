//go:build mage

package main

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

type Lint mg.Namespace

// All Run all linters
func (l Lint) All() error {
	mg.Deps(l.Go)
	return nil
}

// Go Run all go linters
func (l Lint) Go() error {
	mg.Deps(l.Gofumpt, l.Golangcilint)
	return nil
}

// Gofumpt Run gofumpt
func (Lint) Gofumpt() error {
	fmt.Println("formatting go")
	return RunSh("go", Tool())("run", "mvdan.cc/gofumpt", "-l", "-w", "..")
}

// Golangcilint Run golangci-lint
func (Lint) Golangcilint() error {
	fmt.Println("running golangci-lint")
	return RunSh("go", WithV())("run", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint", "run", "--fix")
}
