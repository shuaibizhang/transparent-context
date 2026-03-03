package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	transparentcontext "github.com/shuaibizhang/transparent-context/context"
	_ "github.com/shuaibizhang/transparent-context/demo/pkg/codec"
	"github.com/shuaibizhang/transparent-context/demo/pkg/model"
	"github.com/shuaibizhang/transparent-context/middleware/grpcmiddleware"
	"github.com/shuaibizhang/transparent-context/middleware/httpmiddleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string     `json:"message"`
	Node    model.Node `json:"node"`
}

func main() {
	// Setup gRPC client to Service C
	conn, err := grpc.Dial("localhost:8082",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
		grpc.WithUnaryInterceptor(grpcmiddleware.TransparentContextUnaryClientInterceptor()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	r := gin.Default()
	r.Use(httpmiddleware.TransparentContextMiddleware())

	r.GET("/hello", func(c *gin.Context) {
		ctx := c.Request.Context()
		tc := transparentcontext.GetTransparentContext(ctx)

		// Create B's node metadata
		node := model.Node{
			Service: "ServiceB",
			Metadata: model.Metadata{
				ReqAll:   make(map[string]string),
				ReqOnce:  make(map[string]string),
				RespAll:  make(map[string]string),
				RespOnce: make(map[string]string),
			},
		}

		if tc != nil {
			fmt.Printf("[Service B] Received Req-All-TraceID: %s\n", tc.GetReqAllByKey("TraceID"))
			fmt.Printf("[Service B] Received Req-Once-From: %s\n", tc.GetReqOnceByKey("From"))

			// Capture request metadata
			node.Metadata.ReqAll = tc.GetReqAll()
			node.Metadata.ReqOnce = tc.GetReqOnce()

			// Prepare for next hop
			tc.SetReqOnceByKey("From", "ServiceB")
		} else {
			fmt.Println("[Service B] No TransparentContext found!")
		}

		// Call Service C
		out := new(HelloResponse)
		err := conn.Invoke(ctx, "/hello.HelloService/SayHello", &HelloRequest{Name: "World"}, out)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if tc != nil {
			fmt.Printf("[Service B] Received Resp-All-TraceID from C: %s\n", tc.GetRespAllByKey("TraceID"))
			fmt.Printf("[Service B] Received Resp-Once-From from C: %s\n", tc.GetRespOnceByKey("From"))

			tc.SetRespOnceByKey("From", "ServiceB")

			// Capture response metadata (what we just set)
			node.Metadata.RespAll = tc.GetRespAll()
			node.Metadata.RespOnce = tc.GetRespOnce()
		}

		// Construct chain response
		resp := model.ChainResponse{
			Message: out.Message,
			Chain:   []model.Node{node, out.Node},
		}

		c.JSON(200, resp)
	})

	log.Println("Service B (Gin) listening at :8081")
	r.Run(":8081")
}
