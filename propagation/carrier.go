package propagation

import (
	"net/http"

	"google.golang.org/grpc/metadata"
)

// MetadataCarrier 定义了元数据载体的接口
type MetadataCarrier interface {
	Get(key string) string
	Set(key string, val string)
	Keys() []string
}

// HTTPCarrier 实现了 MetadataCarrier 接口，用于 HTTP 头部
type HTTPCarrier http.Header

// Get 获取 HTTP 头部中指定键的值
func (h HTTPCarrier) Get(key string) string {
	return http.Header(h).Get(key)
}

// Set 设置 HTTP 头部中指定键的值
func (h HTTPCarrier) Set(key string, val string) {
	http.Header(h).Set(key, val)
}

// Keys 返回 HTTP 头部中所有的键
func (h HTTPCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// GRPCCarrier 实现了 MetadataCarrier 接口，用于 gRPC 元数据
type GRPCCarrier metadata.MD

// Get 获取 gRPC 元数据中指定键的值
func (g GRPCCarrier) Get(key string) string {
	vals := metadata.MD(g).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set 设置 gRPC 元数据中指定键的值
func (g GRPCCarrier) Set(key string, val string) {
	// 使用 Set 而不是 Append，以避免重复键
	metadata.MD(g).Set(key, val)
}

// Keys 返回 gRPC 元数据中所有的键
func (g GRPCCarrier) Keys() []string {
	keys := make([]string, 0, len(g))
	for k := range g {
		keys = append(keys, k)
	}
	return keys
}
