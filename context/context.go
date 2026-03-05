package context

import "context"

/* 将透传上下文包装到context中，进行站内传递 */

// 从context中获取透传上下文
func GetTransparentContext(ctx context.Context) TransparentContext {
	if ctx == nil {
		return nil
	}

	if val, ok := ctx.Value(TransParentCtx{}).(TransparentContext); ok {
		return val
	}

	return nil
}

// 将透传上下文设置到context中
func WithTransparentContext(ctx context.Context, tc TransparentContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, TransParentCtx{}, tc)
}

/* 根据metadata，创建新的带透传上下文的context */
func NewContextWithTransparentContextFromReq(reqMetadata map[string]string) context.Context {
	tc := NewTransparentContext()
	tc.LoadFromReqMetadata(reqMetadata)

	bgCtx := context.Background()
	return WithTransparentContext(bgCtx, tc)
}

func NewContextWithTransparentContextFromResp(reqMetadata map[string]string) context.Context {
	tc := NewTransparentContext()
	tc.LoadFromRespMetadata(reqMetadata)

	bgCtx := context.Background()
	return WithTransparentContext(bgCtx, tc)
}
