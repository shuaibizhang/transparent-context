package propagation

import (
	"context"

	transparentconetext "github.com/shuaibizhang/transparent-context/context"
)

// Propagator 传播者接口
type Propagator interface {
	// 将context中的元数据注入元数据载体
	Inject(ctx context.Context, carrier MetadataCarrier) error
	// 从元数据载体中提取元数据，并放入context中
	Extract(ctx context.Context, carrier MetadataCarrier) (context.Context, error)
}

func NewRequestPropagator() Propagator {
	return &requestPropagator{}
}

func NewResponsePropagator() Propagator {
	return &responsePropagator{}
}

type requestPropagator struct{}

func (d requestPropagator) Inject(ctx context.Context, carrier MetadataCarrier) error {
	// 从context中获取透传数据，注入到carrier中
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		return nil
	}

	reqMetaDataMap := tc.InjectToReqMetadata()
	if reqMetaDataMap != nil {
		for k, v := range reqMetaDataMap {
			carrier.Set(k, v)
		}
	}
	return nil
}

func (d requestPropagator) Extract(ctx context.Context, carrier MetadataCarrier) (context.Context, error) {
	// 获取透传上下文，没有创建新的
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		tc = transparentconetext.NewTransparentContext()
	}

	// 从carrier中提取元数据，放入透传上下文中
	metaDataMap := make(map[string]string)
	for _, key := range carrier.Keys() {
		val := carrier.Get(key)
		metaDataMap[key] = val
	}
	tc.LoadFromReqMetadata(metaDataMap)

	// set透传上下文到context中
	ctx = transparentconetext.WithTransparentContext(ctx, tc)
	return ctx, nil
}

type responsePropagator struct{}

func (d responsePropagator) Inject(ctx context.Context, carrier MetadataCarrier) error {
	// 从context中获取透传数据，注入到carrier中
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		return nil
	}

	respMetaDataMap := tc.InjectToRespMetadata()
	if respMetaDataMap != nil {
		for k, v := range respMetaDataMap {
			carrier.Set(k, v)
		}
	}
	return nil
}

func (d responsePropagator) Extract(ctx context.Context, carrier MetadataCarrier) (context.Context, error) {
	// 获取透传上下文，没有创建新的
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		tc = transparentconetext.NewTransparentContext()
	}

	// 从carrier中提取元数据，放入透传上下文中
	metaDataMap := make(map[string]string)
	for _, key := range carrier.Keys() {
		val := carrier.Get(key)
		metaDataMap[key] = val
	}
	tc.LoadFromRespMetadata(metaDataMap)

	// set透传上下文到context中
	ctx = transparentconetext.WithTransparentContext(ctx, tc)
	return ctx, nil
}
