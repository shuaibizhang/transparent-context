package httpmiddleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shuaibizhang/transparent-context/propagation"
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

func TransparentContextMiddleware(opts ...Option) gin.HandlerFunc {
	options := &propagationInterceptorOptions{
		reqPropagator:  propagation.NewRequestPropagator(),
		respPropagator: propagation.NewResponsePropagator(),
	}
	for _, opt := range opts {
		opt(options)
	}

	return func(c *gin.Context) {
		// 1、inbound，，服务端入站请求：从http请求头中提取元数据
		carrier := propagation.HTTPCarrier(c.Request.Header)
		ctx, err := options.reqPropagator.Extract(c.Request.Context(), carrier)
		if err == nil {
			c.Request = c.Request.WithContext(ctx)
		}

		// Wrap ResponseWriter to inject headers before writing response
		w := &responseWriter{
			ResponseWriter: c.Writer,
			ctx:            ctx,
			propagator:     options.respPropagator,
		}
		c.Writer = w

		// 2、业务逻辑处理
		c.Next()

		// If nothing was written, we might need to inject here?
		// But c.Next() usually handles writing.
		// If handler didn't write, Gin might write 200 OK automatically?
		// No, Gin doesn't auto-write unless you call something.
		// But if we return, and nothing written, we might want to inject headers anyway?
		// Typically, headers are only sent when body is sent.
		// If handler returns without writing, headers are not sent yet.
		// But the wrapper will handle it when Write is called.
	}
}

type responseWriter struct {
	gin.ResponseWriter
	ctx        context.Context
	propagator propagation.Propagator
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.inject()
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.inject()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) WriteHeaderNow() {
	w.inject()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.inject()
	return w.ResponseWriter.WriteString(s)
}

func (w *responseWriter) inject() {
	if w.ResponseWriter.Written() {
		return
	}
	carrier := propagation.HTTPCarrier(w.Header())
	_ = w.propagator.Inject(w.ctx, carrier)
}

// 客户端发起时注入
func InjectToHttpClientHeader(ctx context.Context, header http.Header, opts ...Option) {
	options := &propagationInterceptorOptions{
		reqPropagator:  propagation.NewRequestPropagator(),
		respPropagator: propagation.NewResponsePropagator(),
	}
	for _, opt := range opts {
		opt(options)
	}
	carrier := propagation.HTTPCarrier(header)
	_ = options.reqPropagator.Inject(ctx, carrier)
}
