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

// NewRequestPropagator 创建请求方向的传播者
func NewRequestPropagator() Propagator {
	return &basePropagator{
		inject: func(tc transparentconetext.TransparentContext) map[string]string {
			return tc.InjectToReqMetadata()
		},
		extract: func(tc transparentconetext.TransparentContext, metaData map[string]string) {
			tc.LoadFromReqMetadata(metaData)
		},
	}
}

// NewResponsePropagator 创建响应方向的传播者
func NewResponsePropagator() Propagator {
	return &basePropagator{
		inject: func(tc transparentconetext.TransparentContext) map[string]string {
			return tc.InjectToRespMetadata()
		},
		extract: func(tc transparentconetext.TransparentContext, metaData map[string]string) {
			tc.LoadFromRespMetadata(metaData)
		},
	}
}

// basePropagator 基础传播者，封装了通用的注入和提取逻辑
type basePropagator struct {
	inject  func(tc transparentconetext.TransparentContext) map[string]string
	extract func(tc transparentconetext.TransparentContext, metaData map[string]string)
}

func (b *basePropagator) Inject(ctx context.Context, carrier MetadataCarrier) error {
	// 从 context 中获取透传上下文
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		return nil
	}

	// 执行注入逻辑，获取元数据 map
	metaDataMap := b.inject(tc)
	if metaDataMap != nil {
		for k, v := range metaDataMap {
			carrier.Set(k, v)
		}
	}
	return nil
}

func (b *basePropagator) Extract(ctx context.Context, carrier MetadataCarrier) (context.Context, error) {
	// 获取或创建透传上下文
	tc := transparentconetext.GetTransparentContext(ctx)
	if tc == nil {
		tc = transparentconetext.NewTransparentContext()
	}

	// 从 carrier 中批量提取元数据
	keys := carrier.Keys()
	if len(keys) > 0 {
		metaDataMap := make(map[string]string, len(keys))
		for _, key := range keys {
			metaDataMap[key] = carrier.Get(key)
		}
		// 执行具体的提取加载逻辑
		b.extract(tc, metaDataMap)
	}

	// 将更新后的透传上下文存回 context
	return transparentconetext.WithTransparentContext(ctx, tc), nil
}
