// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package github implements interface to GitHub API.
package github

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/google/go-github/v92/github"
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

	ghclient, err := github.NewClient(
		github.WithAuthToken(token),
		github.WithTransport(&ssoHintTransport{base: http.DefaultTransport}),
	)
	if err != nil {
		fmt.Printf("Failed to create GitHub client: %v\n", err)

		return output
	}

	output.client = ghclient

	return output
}

// Generate implements Output interface.
func (o *Output) Generate() error {
	// GitHub API does all the work in Compile, so this method does nothing.
	return nil
}

// Compile implements [output.TypedWriter] interface. The client may be nil
// when GITHUB_TOKEN is not set, which disables GitHub API integration. The
// nil client is forwarded to the compiler as-is; dry-run behavior must be
// opted into explicitly by the compiler.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGitHub(o.client)
}

// Compiler is implemented by project blocks which support GitHub API interface.
type Compiler interface {
	CompileGitHub(*github.Client) error
}

// ssoHintTransport prints a hint how to authorize the GITHUB_TOKEN when GitHub rejects a request
// because the token is not authorized for the organization SAML SSO.
type ssoHintTransport struct {
	base http.RoundTripper
	once sync.Once
}

// RoundTrip implements http.RoundTripper.
func (t *ssoHintTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusForbidden {
		return resp, err
	}

	// format: "required; url=https://github.com/orgs/<org>/sso?authorization_request=..."
	if _, url, ok := strings.Cut(resp.Header.Get("X-Github-Sso"), "url="); ok {
		t.once.Do(func() {
			fmt.Fprintf(os.Stderr, "GITHUB_TOKEN is not authorized for the organization SSO, authorize it at: %s\n", url)
		})
	}

	return resp, err
}
