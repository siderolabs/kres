// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v92/github"
	"golang.org/x/crypto/nacl/box"
)

const (
	// SlashMergeAppClientIDEnv is the environment variable kres reads the slash-merge GitHub App client ID from.
	SlashMergeAppClientIDEnv = "KRES_SLASH_MERGE_APP_CLIENT_ID"
	// SlashMergeAppPrivateKeyEnv is the environment variable kres reads the slash-merge GitHub App private key (PEM) from.
	SlashMergeAppPrivateKeyEnv = "KRES_SLASH_MERGE_APP_PRIVATE_KEY"
)

// slashMergeAppCredentials are the GitHub App credentials kres stores in the slash-merge environment.
type slashMergeAppCredentials struct {
	privateKey    *rsa.PrivateKey
	clientID      string
	privateKeyPEM string
}

// slashMergeAppCredentialsFromEnv reads the GitHub App credentials from the environment.
//
// It returns nil if the credentials are not provided.
func slashMergeAppCredentialsFromEnv() (*slashMergeAppCredentials, error) {
	clientID := os.Getenv(SlashMergeAppClientIDEnv)
	privateKeyPEM := os.Getenv(SlashMergeAppPrivateKeyEnv)

	switch {
	case clientID == "" && privateKeyPEM == "":
		return nil, nil //nolint:nilnil
	case clientID == "" || privateKeyPEM == "":
		return nil, fmt.Errorf("both %s and %s should be set", SlashMergeAppClientIDEnv, SlashMergeAppPrivateKeyEnv)
	}

	return parseSlashMergeAppCredentials(clientID, privateKeyPEM)
}

func parseSlashMergeAppCredentials(clientID, privateKeyPEM string) (*slashMergeAppCredentials, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("%s doesn't contain a PEM-encoded private key", SlashMergeAppPrivateKeyEnv)
	}

	var (
		privateKey *rsa.PrivateKey
		err        error
	)

	switch block.Type {
	case "RSA PRIVATE KEY": // GitHub generates PKCS#1 keys
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		var key any

		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			var ok bool

			if privateKey, ok = key.(*rsa.PrivateKey); !ok {
				err = errors.New("not an RSA key")
			}
		}
	default:
		err = fmt.Errorf("unexpected PEM block type %q", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", SlashMergeAppPrivateKeyEnv, err)
	}

	return &slashMergeAppCredentials{
		clientID:      clientID,
		privateKeyPEM: privateKeyPEM,
		privateKey:    privateKey,
	}, nil
}

// jwt returns a short-lived JWT to authenticate as the GitHub App.
func (creds *slashMergeAppCredentials) jwt(now time.Time) (string, error) {
	enc := base64.RawURLEncoding

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}

	claims, err := json.Marshal(map[string]any{
		"iat": now.Add(-time.Minute).Unix(), // allow for clock drift
		"exp": now.Add(5 * time.Minute).Unix(),
		"iss": creds.clientID,
	})
	if err != nil {
		return "", err
	}

	signingInput := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))

	signature, err := rsa.SignPKCS1v15(rand.Reader, creds.privateKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}

	return signingInput + "." + enc.EncodeToString(signature), nil
}

// provisionSlashMergeApp stores the GitHub App credentials (if provided) in the slash-merge environment,
// or checks that they are present.
//
// kres can't create GitHub App credentials (there is no API for that), they have to be provided by the caller.
// The credentials are verified before they are stored: the private key must match the client ID, and the app must be
// installed on the repository with the permissions the workflow requests.
func (r *Repository) provisionSlashMergeApp(client *github.Client, creds *slashMergeAppCredentials) error {
	if r.DryRun {
		if creds != nil {
			fmt.Printf("dry-run: repository would store GitHub App credentials in environment %q\n", slashMergeEnvironment)
		} else {
			fmt.Printf("dry-run: repository would check GitHub App credentials in environment %q\n", slashMergeEnvironment)
		}

		return nil
	}

	if creds == nil {
		return r.checkSlashMergeApp(client)
	}

	slug, err := r.verifySlashMergeApp(client, creds)
	if err != nil {
		return err
	}

	ctx := context.Background()
	owner, repo := r.meta.GitHubOrganization, r.meta.GitHubRepository

	publicKey, _, err := client.Actions.GetEnvPublicKey(ctx, owner, repo, slashMergeEnvironment)
	if err != nil {
		return fmt.Errorf("failed to get public key of environment %q: %w", slashMergeEnvironment, err)
	}

	encrypted, err := encryptSecret(publicKey.GetKey(), creds.privateKeyPEM)
	if err != nil {
		return err
	}

	if _, err = client.Actions.CreateOrUpdateEnvSecret(ctx, owner, repo, slashMergeEnvironment, slashMergeAppPrivateKeySecret, github.SecretRequest{
		KeyID:          publicKey.GetKeyID(),
		EncryptedValue: encrypted,
	}); err != nil {
		return fmt.Errorf("failed to store secret %s in environment %q: %w", slashMergeAppPrivateKeySecret, slashMergeEnvironment, err)
	}

	variable, resp, err := client.Actions.GetEnvVariable(ctx, owner, repo, slashMergeEnvironment, slashMergeAppClientIDVariable)

	switch {
	case err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound):
		return fmt.Errorf("failed to get variable %s of environment %q: %w", slashMergeAppClientIDVariable, slashMergeEnvironment, err)
	case err != nil:
		_, err = client.Actions.CreateEnvVariable(ctx, owner, repo, slashMergeEnvironment, github.ActionsCreateVariableRequest{
			Name:  slashMergeAppClientIDVariable,
			Value: creds.clientID,
		})
	case variable.Value != creds.clientID:
		_, err = client.Actions.UpdateEnvVariable(ctx, owner, repo, slashMergeEnvironment, slashMergeAppClientIDVariable, github.ActionsUpdateVariableRequest{
			Value: new(creds.clientID),
		})
	}

	if err != nil {
		return fmt.Errorf("failed to store variable %s in environment %q: %w", slashMergeAppClientIDVariable, slashMergeEnvironment, err)
	}

	fmt.Printf("GitHub App %q credentials stored in environment %q\n", slug, slashMergeEnvironment)

	return nil
}

// verifySlashMergeApp checks that the credentials are valid, and the app is installed on the repository
// with the required permissions. It returns the app slug.
func (r *Repository) verifySlashMergeApp(client *github.Client, creds *slashMergeAppCredentials) (string, error) {
	ctx := context.Background()

	jwt, err := creds.jwt(time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to sign GitHub App JWT: %w", err)
	}

	appClient, err := github.NewClient(
		github.WithAuthToken(jwt),
		github.WithURLs(new(client.BaseURL()), new(client.UploadURL())),
	)
	if err != nil {
		return "", err
	}

	app, _, err := appClient.Apps.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("failed to authenticate as GitHub App %q, check that %s is the private key of the app: %w",
			creds.clientID, SlashMergeAppPrivateKeyEnv, err)
	}

	installation, resp, err := appClient.Apps.GetRepositoryInstallation(ctx, r.meta.GitHubOrganization, r.meta.GitHubRepository)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("GitHub App %q is not installed on %s/%s, add the repository to the app installation: %s/installations/new",
				app.GetSlug(), r.meta.GitHubOrganization, r.meta.GitHubRepository, app.GetHTMLURL())
		}

		return "", fmt.Errorf("failed to get GitHub App %q installation: %w", app.GetSlug(), err)
	}

	if installation.SuspendedAt != nil {
		return "", fmt.Errorf("GitHub App %q installation is suspended", app.GetSlug())
	}

	// these are the permissions accepted by the installation, which might lag behind the app permissions
	if perms := installation.GetPermissions(); perms.GetContents() != "write" || perms.GetWorkflows() != "write" {
		return "", fmt.Errorf(
			"GitHub App %q installation should have contents: write and workflows: write permissions (got contents: %q, workflows: %q), "+
				"update the app permissions and accept them in the organization settings",
			app.GetSlug(), perms.GetContents(), perms.GetWorkflows(),
		)
	}

	return app.GetSlug(), nil
}

// checkSlashMergeApp checks that the GitHub App credentials are present in the slash-merge environment,
// and prints instructions to provide them if they are not.
func (r *Repository) checkSlashMergeApp(client *github.Client) error {
	ctx := context.Background()
	owner, repo := r.meta.GitHubOrganization, r.meta.GitHubRepository

	var missing []string

	if _, resp, err := client.Actions.GetEnvVariable(ctx, owner, repo, slashMergeEnvironment, slashMergeAppClientIDVariable); err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to get variable %s of environment %q: %w", slashMergeAppClientIDVariable, slashMergeEnvironment, err)
		}

		missing = append(missing, "variable "+slashMergeAppClientIDVariable)
	}

	if _, resp, err := client.Actions.GetEnvSecret(ctx, owner, repo, slashMergeEnvironment, slashMergeAppPrivateKeySecret); err != nil {
		if resp == nil || resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to get secret %s of environment %q: %w", slashMergeAppPrivateKeySecret, slashMergeEnvironment, err)
		}

		missing = append(missing, "secret "+slashMergeAppPrivateKeySecret)
	}

	if len(missing) == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, `WARNING: slash-merge is enabled, but environment %q is missing %s,
so /nm won't work. Re-run kres with the GitHub App credentials to provision them:

  export %s=<app client ID>
  export %s="$(cat <app private key>.pem)"

`, slashMergeEnvironment, strings.Join(missing, " and "), SlashMergeAppClientIDEnv, SlashMergeAppPrivateKeyEnv)

	return nil
}

// encryptSecret encrypts the secret value for the GitHub Actions secrets API.
func encryptSecret(publicKeyBase64, value string) (string, error) {
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode environment public key: %w", err)
	}

	if len(publicKey) != 32 {
		return "", fmt.Errorf("unexpected environment public key length %d", len(publicKey))
	}

	encrypted, err := box.SealAnonymous(nil, []byte(value), (*[32]byte)(publicKey), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
