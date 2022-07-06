package interceptors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestMDToOutgoing(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "testV", "1")
	md, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.True(t, md.Get("testV")[0] == "1")

	ctx = metadata.AppendToOutgoingContext(ctx, "testV", "1")
	md, ok = metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.True(t, len(md.Get("testV")) == 2)
	assert.True(t, md.Get("testV")[0] == "1")
}
