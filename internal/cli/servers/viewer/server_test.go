package viewer_test

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreofscience/go-bibx/internal/cli/servers/viewer"
	"github.com/coreofscience/go-bibx/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewer_Endpoints(t *testing.T) {
	analysis := &models.Analysis{
		Nodes: []*models.Node{
			{ID: "node1", Category: models.CategoryRoot},
		},
		Links: []*models.Link{
			{Source: "node1", Target: "node1", Weight: 1.0},
		},
	}
	mux := viewer.New(analysis)

	t.Run("GET / returns index html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Contains(t, rec.Body.String(), "<html")
	})

	t.Run("GET /unknown returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GET /collection.json.gz returns compressed analysis", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/collection.json.gz", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

		gzReader, err := gzip.NewReader(rec.Body)
		require.NoError(t, err)
		defer func() {
			_ = gzReader.Close()
		}()

		var decoded models.Analysis
		err = json.NewDecoder(gzReader).Decode(&decoded)
		require.NoError(t, err)
		assert.Len(t, decoded.Nodes, 1)
		assert.Equal(t, "node1", decoded.Nodes[0].ID)
	})
}

func TestViewer_ServeListener_GracefulShutdown(t *testing.T) {
	analysis := &models.Analysis{}
	mux := viewer.New(analysis)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	server := &http.Server{
		Handler: mux,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- viewer.ServeListener(ctx, server, ln)
	}()

	// Verify server responds
	resp, err := http.Get(fmt.Sprintf("http://%s/", addr))
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Trigger graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for graceful shutdown")
	}
}

func TestViewer_Serve_ListenError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = ln.Close()
	}()

	server := &http.Server{
		Addr: ln.Addr().String(),
	}

	err = viewer.Serve(context.Background(), server)
	assert.Error(t, err)
}

func TestViewer_ServeListener_AlreadyCanceled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = viewer.ServeListener(ctx, server, ln)
	assert.NoError(t, err)
}
