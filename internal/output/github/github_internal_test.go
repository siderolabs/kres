// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSOHintTransport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Github-Sso", "required; url=https://github.com/orgs/org/sso?authorization_request=abc")
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: &ssoHintTransport{base: http.DefaultTransport}}

	// the response is passed through unchanged, the hint is only printed
	for range 2 {
		resp, err := client.Get(srv.URL) //nolint:noctx
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}
