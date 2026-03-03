package grpcmiddleware

import (
	"context"

	transparentconetext "github.com/shuaibizhang/transparent-context/context"
	"github.com/shuaibizhang/transparent-context/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Option func(*propagationInterceptorOptions)

// 拦截器选项
type propagationInterceptorOptions struct {
	reqPropagator  propagation.Propagator
	respPropagator propagation.Propagator
}

// 传播器选项
func WithPropagators(reqPropagator, respPropagator propagation.Propagator) Option {
	return func(o *propagationInterceptorOptions) {
		if reqPropagator != nil {
			o.reqPropagator = reqPropagator
		}
		if respPropagator != nil {
			o.respPropagator = respPropagator
		}
	}
}

// 服务器端拦截器实现
func TransparentContextUnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	options := propagationInterceptorOptions{
		reqPropagator:  propagation.NewRequestPropagator(),
		respPropagator: propagation.NewResponsePropagator(),
	}
	for _, o := range opts {
		o(&options)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// 1、inbound，服务端入站请求：从grpc元数据中提取
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			// 从grpc md中提取req透传上下文放入ctx中透传
			carrier := propagation.GRPCCarrier(md)
			ctx, err = options.reqPropagator.Extract(ctx, carrier)
			if err != nil {
				// TODO：错误处理
			}
		}

		// 2、处理业务逻辑
		resp, err = handler(ctx, req)
		if err != nil {
			return resp, err
		}

		// 3、outbound，服务端出站响应：将透传的元数据放入grpc元数据中返回
		if transparentconetext.GetTransparentContext(ctx) != nil {
			// 准备好载体
			respMD := metadata.MD{}
			carrier := propagation.GRPCCarrier(respMD)
			// 注入resp透传上下文到载体中
			_ = options.respPropagator.Inject(ctx, carrier)

			if len(respMD) > 0 {
				// 将载体中的透传上下文放入grpc元数据中返回
				grpc.SetHeader(ctx, respMD)
			}
		}

		return resp, err
	}
}

// 客户端拦截器实现
func TransparentContextUnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	options := propagationInterceptorOptions{
		reqPropagator:  propagation.NewRequestPropagator(),
		respPropagator: propagation.NewResponsePropagator(),
	}
	for _, o := range opts {
		o(&options)
	}

	return func(ctx context.Context, method string, req, resp any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 1、outbound，客户端出站请求，将透传上下文注入到grpc元数据中
		// 从grpc metadata中获取数据
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		} else {
			md = md.Copy()
		}

		// 将透传上下文注入到grpc元数据中
		if transparentconetext.GetTransparentContext(ctx) != nil {
			carrier := propagation.GRPCCarrier(md)
			_ = options.reqPropagator.Inject(ctx, carrier)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		// 2、执行业务逻辑
		var headMD metadata.MD
		opts = append(opts, grpc.Header(&headMD))
		err := invoker(ctx, method, req, resp, cc, opts...)
		if err != nil {
			return err
		}

		// 3、inbound，客户端入站响应，将grpc元数据中的透传上下文取出
		if len(headMD) > 0 {
			carrier := propagation.GRPCCarrier(headMD)
			_, _ = options.respPropagator.Extract(ctx, carrier)
		}
		return err
	}
}
