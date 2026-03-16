package context

import (
	"context"
	"testing"
)

func TestWithTransparentContext(t *testing.T) {
	ctx := context.Background()
	tc := NewTransparentContext()
	tc.SetReqAllByKey("Key", "value")

	newCtx := WithTransparentContext(ctx, tc)

	got := GetTransparentContext(newCtx)
	if got.GetReqAllByKey("Key") != "value" {
		t.Errorf("expected value, got %v", got.GetReqAllByKey("Key"))
	}
}
