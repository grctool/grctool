package transport_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/grctool/grctool/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggingTransport_Defaults(t *testing.T) {
	t.Parallel()

	lt := transport.NewLoggingTransport(nil, nil)
	require.NotNil(t, lt)
	assert.NotNil(t, lt.Transport, "should default to http.DefaultTransport")
	assert.NotNil(t, lt.Logger, "should create a default logger when nil")
}

func TestNewLoggingTransport_WithCustomTransport(t *testing.T) {
	t.Parallel()

	custom := http.DefaultTransport
	log := testhelpers.NewStubLogger()
	lt := transport.NewLoggingTransport(custom, log)
	assert.Equal(t, custom, lt.Transport)
}

func TestLoggingTransport_RoundTrip_GET(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	log := testhelpers.NewCapturingLogger()
	lt := transport.NewLoggingTransport(http.DefaultTransport, log)

	client := &http.Client{Transport: lt}
	resp, err := client.Get(srv.URL + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `"status":"ok"`)
	// Verify logging occurred
	assert.True(t, log.HasEntry("DEBUG", "http request"))
	assert.True(t, log.HasEntry("DEBUG", "http response"))
}

func TestLoggingTransport_RoundTrip_POST(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(body), "hello")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	log := testhelpers.NewCapturingLogger()
	lt := transport.NewLoggingTransport(http.DefaultTransport, log)

	client := &http.Client{Transport: lt}
	resp, err := client.Post(srv.URL+"/items", "application/json", strings.NewReader(`{"msg":"hello"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Body should still be readable after logging
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"id":1`)
}

func TestLoggingTransport_RoundTrip_Error(t *testing.T) {
	t.Parallel()

	log := testhelpers.NewCapturingLogger()
	lt := transport.NewLoggingTransport(http.DefaultTransport, log)

	client := &http.Client{Transport: lt}
	// Connect to a port that should refuse connections
	_, err := client.Get("http://127.0.0.1:1/fail")
	assert.Error(t, err)
	assert.True(t, log.HasEntry("DEBUG", "http request failed"))
}

func TestLoggingTransport_BodyPreserved(t *testing.T) {
	t.Parallel()

	expected := "response body content here"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expected))
	}))
	defer srv.Close()

	log := testhelpers.NewStubLogger()
	lt := transport.NewLoggingTransport(http.DefaultTransport, log)

	client := &http.Client{Transport: lt}
	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The body must be fully readable even after the transport logged it
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, expected, string(body))
}

func TestLoggingTransport_RequestBodyPreserved(t *testing.T) {
	t.Parallel()

	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	log := testhelpers.NewStubLogger()
	lt := transport.NewLoggingTransport(http.DefaultTransport, log)

	client := &http.Client{Transport: lt}
	_, err := client.Post(srv.URL, "text/plain", strings.NewReader("original body"))
	require.NoError(t, err)

	// The server should have received the body intact
	assert.Equal(t, "original body", receivedBody)
}
