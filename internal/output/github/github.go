// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package github implements interface to GitHub API.
package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v71/github"
	"golang.org/x/oauth2"
)

// Output implements interface to GitHub API.
type Output struct {
	client *github.Client
}

// NewOutput creates new GitHub API output.
func NewOutput() *Output {
	output := &Output{}

	token, exists := os.LookupEnv("GITHUB_TOKEN")
	if !exists {
		fmt.Println("GITHUB_TOKEN is missing, GitHub API integration is disabled")

		return output
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	output.client = github.NewClient(oauth2.NewClient(ctx, ts))

	return output
}

// Generate implements Output interface.
func (o *Output) Generate() error {
	// GitHub API does all the work in Compile, so this method does nothing.
	return nil
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	if o.client == nil {
		// no token, skip it
		return nil
	}

	return compiler.CompileGitHub(o.client)
}

// Compiler is implemented by project blocks which support GitHub API interface.
type Compiler interface {
	CompileGitHub(*github.Client) error
}
