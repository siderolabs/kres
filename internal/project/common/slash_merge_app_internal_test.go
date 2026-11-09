// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v92/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/nacl/box"

	"github.com/siderolabs/kres/internal/project/meta"
)

const (
	testClientID   = "Iv23test"
	testAdminToken = "admin-token"
)

// fakeAppAPI is a minimal fake of the GitHub API parts used to provision the slash-merge GitHub App credentials.
type fakeAppAPI struct {
	t *testing.T

	appKey *rsa.PublicKey // key the JWT must be signed with

	envPublicKey, envPrivateKey *[32]byte

	installation map[string]any // nil if the app is not installed

	variable *string // nil if missing
	secret   *string // decrypted value, nil if missing

	calls []string

	mu sync.Mutex
}

//nolint:gocyclo,cyclop
func (f *fakeAppAPI) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	const envPrefix = "/repos/org/repo/environments/" + slashMergeEnvironment

	path := strings.TrimPrefix(req.URL.Path, "/api/v3")
	f.calls = append(f.calls, req.Method+" "+path)

	writeJSON := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v) //nolint:errcheck,errchkjson
	}

	token := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")

	// app endpoints accept only the app JWT, everything else only the admin token
	if path == "/app" || path == "/repos/org/repo/installation" {
		if !f.validJWT(token) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}
	} else if token != testAdminToken {
		f.t.Errorf("unexpected token for %s %s", req.Method, path)
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	switch {
	case req.Method == http.MethodGet && path == "/app":
		writeJSON(map[string]any{"slug": "merge-bot", "html_url": "https://github.com/apps/merge-bot"})
	case req.Method == http.MethodGet && path == "/repos/org/repo/installation":
		if f.installation == nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		writeJSON(f.installation)
	case req.Method == http.MethodGet && path == envPrefix+"/secrets/public-key":
		writeJSON(map[string]any{"key_id": "key-1", "key": base64.StdEncoding.EncodeToString(f.envPublicKey[:])})
	case req.Method == http.MethodGet && path == envPrefix+"/secrets/"+slashMergeAppPrivateKeySecret:
		if f.secret == nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		writeJSON(map[string]any{"name": slashMergeAppPrivateKeySecret})
	case req.Method == http.MethodPut && path == envPrefix+"/secrets/"+slashMergeAppPrivateKeySecret:
		var body github.SecretRequest

		require.NoError(f.t, json.NewDecoder(req.Body).Decode(&body))
		assert.Equal(f.t, "key-1", body.KeyID)

		encrypted, err := base64.StdEncoding.DecodeString(body.EncryptedValue)
		require.NoError(f.t, err)

		decrypted, ok := box.OpenAnonymous(nil, encrypted, f.envPublicKey, f.envPrivateKey)
		require.True(f.t, ok, "secret must be encrypted with the environment public key")

		f.secret = new(string(decrypted))

		w.WriteHeader(http.StatusCreated)
	case req.Method == http.MethodGet && path == envPrefix+"/variables/"+slashMergeAppClientIDVariable:
		if f.variable == nil {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		writeJSON(map[string]any{"name": slashMergeAppClientIDVariable, "value": *f.variable})
	case req.Method == http.MethodPost && path == envPrefix+"/variables",
		req.Method == http.MethodPatch && path == envPrefix+"/variables/"+slashMergeAppClientIDVariable:
		var body map[string]string

		require.NoError(f.t, json.NewDecoder(req.Body).Decode(&body))

		f.variable = new(body["value"])

		w.WriteHeader(http.StatusCreated)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (f *fakeAppAPI) validJWT(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}

	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))

	if rsa.VerifyPKCS1v15(f.appKey, crypto.SHA256, digest[:], signature) != nil {
		return false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	var claims struct {
		Iss string `json:"iss"`
		Iat int64  `json:"iat"`
		Exp int64  `json:"exp"`
	}

	if json.Unmarshal(payload, &claims) != nil {
		return false
	}

	now := time.Now().Unix()

	return claims.Iss == testClientID && claims.Iat <= now && claims.Exp > now && claims.Exp-claims.Iat <= 600
}

func generateAppKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return key, string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

//nolint:gocognit
func TestProvisionSlashMergeApp(t *testing.T) {
	t.Parallel()

	appKey, appKeyPEM := generateAppKey(t)
	_, otherKeyPEM := generateAppKey(t)

	goodInstallation := map[string]any{"id": 1, "permissions": map[string]string{"contents": "write", "workflows": "write", "metadata": "read"}}

	for _, test := range []struct { //nolint:govet
		name string

		keyPEM       string // empty: no credentials provided
		installation map[string]any
		variable     *string
		secret       *string

		expectedErr      string
		expectedVariable *string
		expectedSecret   *string
	}{
		{
			name:             "provision",
			keyPEM:           appKeyPEM,
			installation:     goodInstallation,
			expectedVariable: new(testClientID),
			expectedSecret:   new(appKeyPEM),
		},
		{
			name:             "rotate",
			keyPEM:           appKeyPEM,
			installation:     goodInstallation,
			variable:         new("Iv23old"),
			secret:           new("old key"),
			expectedVariable: new(testClientID),
			expectedSecret:   new(appKeyPEM),
		},
		{
			name:         "wrong key",
			keyPEM:       otherKeyPEM,
			installation: goodInstallation,
			expectedErr:  "failed to authenticate as GitHub App",
		},
		{
			name:        "not installed",
			keyPEM:      appKeyPEM,
			expectedErr: "add the repository to the app installation: https://github.com/apps/merge-bot/installations/new",
		},
		{
			name:         "missing permissions",
			keyPEM:       appKeyPEM,
			installation: map[string]any{"id": 1, "permissions": map[string]string{"contents": "write", "workflows": "read"}},
			expectedErr:  "should have contents: write and workflows: write permissions",
		},
		{
			name:   "suspended",
			keyPEM: appKeyPEM,
			installation: map[string]any{
				"id": 1, "suspended_at": "2026-01-01T00:00:00Z",
				"permissions": map[string]string{"contents": "write", "workflows": "write"},
			},
			expectedErr: "installation is suspended",
		},
		{
			name:             "check present",
			variable:         new(testClientID),
			secret:           new(appKeyPEM),
			expectedVariable: new(testClientID),
			expectedSecret:   new(appKeyPEM),
		},
		{
			name: "check missing", // only a warning
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			envPublicKey, envPrivateKey, err := box.GenerateKey(rand.Reader)
			require.NoError(t, err)

			api := &fakeAppAPI{
				t:             t,
				appKey:        &appKey.PublicKey,
				envPublicKey:  envPublicKey,
				envPrivateKey: envPrivateKey,
				installation:  test.installation,
				variable:      test.variable,
				secret:        test.secret,
			}

			srv := httptest.NewServer(api)
			t.Cleanup(srv.Close)

			client, err := github.NewClient(github.WithAuthToken(testAdminToken), github.WithURLs(new(srv.URL+"/"), new(srv.URL+"/")))
			require.NoError(t, err)

			r := NewRepository(&meta.Options{
				MainBranch:         "main",
				GitHubOrganization: "org",
				GitHubRepository:   "repo",
			})

			var creds *slashMergeAppCredentials

			if test.keyPEM != "" {
				creds, err = parseSlashMergeAppCredentials(testClientID, test.keyPEM)
				require.NoError(t, err)
			}

			err = r.provisionSlashMergeApp(client, creds)

			if test.expectedErr != "" {
				require.ErrorContains(t, err, test.expectedErr)

				// nothing is stored if the credentials don't check out
				for _, call := range api.calls {
					assert.NotContains(t, call, "PUT ")
					assert.NotContains(t, call, "POST ")
					assert.NotContains(t, call, "PATCH ")
				}

				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectedVariable, api.variable)
			assert.Equal(t, test.expectedSecret, api.secret)
		})
	}
}

func TestSlashMergeAppCredentialsFromEnv(t *testing.T) {
	_, keyPEM := generateAppKey(t)

	t.Setenv(SlashMergeAppClientIDEnv, "")
	t.Setenv(SlashMergeAppPrivateKeyEnv, "")

	creds, err := slashMergeAppCredentialsFromEnv()
	require.NoError(t, err)
	assert.Nil(t, creds)

	t.Setenv(SlashMergeAppClientIDEnv, testClientID)

	_, err = slashMergeAppCredentialsFromEnv()
	require.ErrorContains(t, err, "should be set")

	t.Setenv(SlashMergeAppPrivateKeyEnv, "not a key")

	_, err = slashMergeAppCredentialsFromEnv()
	require.ErrorContains(t, err, "doesn't contain a PEM-encoded private key")

	t.Setenv(SlashMergeAppPrivateKeyEnv, keyPEM)

	creds, err = slashMergeAppCredentialsFromEnv()
	require.NoError(t, err)
	assert.Equal(t, testClientID, creds.clientID)
}
