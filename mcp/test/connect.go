package test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// Connect connects a server and client using in-memory transports and t.Context().
// It automatically closes the sessions when the test finishes.
func Connect(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := t.Context()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})

	return clientSession
}
