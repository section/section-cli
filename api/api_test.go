package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func helperLoadBytes(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	return bytes
}

func TestAPIClientSetsUserAgent(t *testing.T) {
	assert := assert.New(t)

	// Setup
	keyring.MockInit()

	var userAgent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header["User-Agent"][0]
		w.WriteHeader(http.StatusOK)
	}))

	u, err := url.Parse(ts.URL)
	assert.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	// Invoke
	_, err = request(ctx, http.MethodGet, *u, nil)
	assert.NoError(err)

	// Test
	assert.Regexp("^sectionctl (.+)$", userAgent)
}

func TestPrettyTxIDErrorPrintsApertureTxID(t *testing.T) {
	assert := assert.New(t)

	// Invoke
	resp := http.Response{
		Status: "500 Internal Server Error",
		Header: map[string][]string{"Aperture-Tx-Id": []string{"12345"}},
	}
	err := prettyTxIDError(&resp)

	// Test
	assert.Error(err)
	assert.Regexp("Section Transaction ID", err)
	assert.Regexp(resp.Header["Aperture-Tx-Id"][0], err)
}

func TestPrettyTxIDErrorHandlesNoApertureTxIDHeader(t *testing.T) {
	assert := assert.New(t)

	// Invoke
	resp := http.Response{Status: "500 Internal Server Error"}
	assert.NotPanics(func() { prettyTxIDError(&resp) })
	err := prettyTxIDError(&resp)

	// Test
	assert.Error(err)
	assert.NotRegexp("Section Transaction ID", err)
}

func TestPrettyTxIDErrorHandlesRateLimiting(t *testing.T) {
	assert := assert.New(t)

	// Invoke
	resp := http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     map[string][]string{"Aperture-Tx-Id": []string{"12345"}},
	}
	assert.NotPanics(func() { prettyTxIDError(&resp) })
	err := prettyTxIDError(&resp)

	// Test
	assert.Error(err)
	assert.Regexp(resp.Header["Aperture-Tx-Id"][0], err)
	assert.Regexp("Please wait a few minutes", err)
}

func TestAPIClientUsesCredentialsIfSpecified(t *testing.T) {
	assert := assert.New(t)

	// Setup
	var token string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = r.Header.Get("section-token")
		if assert.NotEmpty(token) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	Token = "s3cr3t"

	u, err := url.Parse(ts.URL)
	assert.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	// Invoke
	resp, err := request(ctx, http.MethodGet, *u, nil)
	assert.NoError(err)

	// Test
	assert.Equal(resp.StatusCode, http.StatusOK)
	assert.Equal(Token, token)

	// Teardown
	Token = ""
}

func TestAPIrequestSendsHeaderArguments(t *testing.T) {
	assert := assert.New(t)

	// Setup
	headers := []http.Header{
		http.Header{"filepath": []string{"/etc/passwd"}},
		http.Header{"Hello": []string{"world"}},
		http.Header{"foo": []string{"bar"}},
	}
	// Test
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, hs := range headers {
			for k, v := range hs {
				assert.Contains(r.Header.Get(k), v[0])
			}
		}
		w.WriteHeader(http.StatusOK)
	}))

	u, err := url.Parse(ts.URL)
	assert.NoError(err)
	Token = "s3cr3t"

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	// Invoke
	_, err = request(ctx, http.MethodGet, *u, nil, headers...)
	assert.NoError(err)
}

func TestAPIrequestReturnsErrorsOnTimeouts(t *testing.T) {
	assert := assert.New(t)

	// Setup
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))

	u, err := url.Parse(ts.URL)
	assert.NoError(err)
	Token = "s3cr3t"

	ctx, cancel := context.WithTimeout(context.Background(), 0*time.Second)
	defer cancel()

	// Invoke
	_, err = request(ctx, http.MethodGet, *u, nil)

	// Test
	assert.Error(err)
	assert.Regexp("context deadline exceeded", err)
}
