package propagation

import (
	"context"
	"testing"

	transparentcontext "github.com/shuaibizhang/transparent-context/context"
)

// mockCarrier 模拟元数据载体
type mockCarrier struct {
	data map[string]string
}

func (m *mockCarrier) Get(key string) string {
	return m.data[key]
}

func (m *mockCarrier) Set(key string, val string) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = val
}

func (m *mockCarrier) Keys() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func TestRequestPropagator(t *testing.T) {
	p := NewRequestPropagator()
	ctx := context.Background()
	tc := transparentcontext.NewTransparentContext()

	t.Run("Inject", func(t *testing.T) {
		tc.SetReqAllByKey("foo", "bar")
		ctxWithTC := transparentcontext.WithTransparentContext(ctx, tc)
		carrier := &mockCarrier{data: make(map[string]string)}

		err := p.Inject(ctxWithTC, carrier)
		if err != nil {
			t.Fatalf("Inject failed: %v", err)
		}

		// Req-All-Foo 是由 TransparentContext 自动拼接的（假定默认实现）
		// 我们通过 GetReqAll 验证 Inject 行为
		val := carrier.Get("Req-All-Foo")
		if val != "bar" {
			t.Errorf("expected Req-All-Foo=bar, got %s", val)
		}
	})

	t.Run("Extract", func(t *testing.T) {
		carrier := &mockCarrier{data: map[string]string{
			"Req-All-Hello": "World",
		}}

		newCtx, err := p.Extract(ctx, carrier)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		newTC := transparentcontext.GetTransparentContext(newCtx)
		if newTC == nil {
			t.Fatal("TransparentContext not found in extracted context")
		}

		val := newTC.GetReqAllByKey("Hello")
		if val != "World" {
			t.Errorf("expected GetReqAllByKey(Hello)=World, got %s", val)
		}
	})
}

func TestResponsePropagator(t *testing.T) {
	p := NewResponsePropagator()
	ctx := context.Background()
	tc := transparentcontext.NewTransparentContext()

	t.Run("Inject", func(t *testing.T) {
		tc.SetRespAllByKey("status", "ok")
		ctxWithTC := transparentcontext.WithTransparentContext(ctx, tc)
		carrier := &mockCarrier{data: make(map[string]string)}

		err := p.Inject(ctxWithTC, carrier)
		if err != nil {
			t.Fatalf("Inject failed: %v", err)
		}

		val := carrier.Get("Resp-All-Status")
		if val != "ok" {
			t.Errorf("expected Resp-All-Status=ok, got %s", val)
		}
	})

	t.Run("Extract", func(t *testing.T) {
		carrier := &mockCarrier{data: map[string]string{
			"Resp-All-Code": "200",
		}}

		newCtx, err := p.Extract(ctx, carrier)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		newTC := transparentcontext.GetTransparentContext(newCtx)
		if newTC == nil {
			t.Fatal("TransparentContext not found in extracted context")
		}

		val := newTC.GetRespAllByKey("Code")
		if val != "200" {
			t.Errorf("expected GetRespAllByKey(Code)=200, got %s", val)
		}
	})
}

func TestPropagatorEdgeCases(t *testing.T) {
	p := NewRequestPropagator()

	t.Run("InjectWithNoTC", func(t *testing.T) {
		ctx := context.Background()
		carrier := &mockCarrier{data: make(map[string]string)}
		err := p.Inject(ctx, carrier)
		if err != nil {
			t.Errorf("Inject with no TC should not return error, got %v", err)
		}
		if len(carrier.data) != 0 {
			t.Errorf("carrier should be empty, got %v", carrier.data)
		}
	})

	t.Run("ExtractWithEmptyCarrier", func(t *testing.T) {
		ctx := context.Background()
		carrier := &mockCarrier{data: make(map[string]string)}
		newCtx, err := p.Extract(ctx, carrier)
		if err != nil {
			t.Errorf("Extract with empty carrier should not return error, got %v", err)
		}
		newTC := transparentcontext.GetTransparentContext(newCtx)
		if newTC == nil {
			t.Fatal("TransparentContext should still be created")
		}
	})
}
