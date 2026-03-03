package main

import (
	"context"
	"fmt"
	"log"
	"net"

	_ "github.com/shuaibizhang/transparent-context/demo/pkg/codec"
	"github.com/shuaibizhang/transparent-context/demo/pkg/model"

	transparentcontext "github.com/shuaibizhang/transparent-context/context"
	"github.com/shuaibizhang/transparent-context/middleware/grpcmiddleware"
	"google.golang.org/grpc"
)

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string     `json:"message"`
	Node    model.Node `json:"node"`
}

type HelloServiceServer interface {
	SayHello(context.Context, *HelloRequest) (*HelloResponse, error)
}

type server struct{}

func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloResponse, error) {
	node := model.Node{
		Service: "ServiceC",
		Metadata: model.Metadata{
			ReqAll:   make(map[string]string),
			ReqOnce:  make(map[string]string),
			RespAll:  make(map[string]string),
			RespOnce: make(map[string]string),
		},
	}

	tc := transparentcontext.GetTransparentContext(ctx)
	if tc != nil {
		fmt.Printf("[Service C] Received Req-All-TraceID: %s\n", tc.GetReqAllByKey("TraceID"))
		fmt.Printf("[Service C] Received Req-Once-From: %s\n", tc.GetReqOnceByKey("From"))

		// Capture request metadata
		node.Metadata.ReqAll = tc.GetReqAll()
		node.Metadata.ReqOnce = tc.GetReqOnce()

		// Set response headers
		tc.SetRespAllByKey("TraceID", tc.GetReqAllByKey("TraceID"))
		tc.SetRespOnceByKey("From", "ServiceC")

		// Capture response metadata (what we just set)
		node.Metadata.RespAll = tc.GetRespAll()
		node.Metadata.RespOnce = tc.GetRespOnce()
	} else {
		fmt.Println("[Service C] No TransparentContext found!")
	}
	return &HelloResponse{
		Message: "Hello " + in.Name,
		Node:    node,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.TransparentContextUnaryServerInterceptor()),
	)

	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "hello.HelloService",
		HandlerType: (*HelloServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "SayHello",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(HelloRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*server).SayHello(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/hello.HelloService/SayHello",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(*server).SayHello(ctx, req.(*HelloRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "hello.proto",
	}, &server{})

	log.Printf("Service C (gRPC) listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
