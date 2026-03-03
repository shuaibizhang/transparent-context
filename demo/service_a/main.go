package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	transparentcontext "github.com/shuaibizhang/transparent-context/context"
	"github.com/shuaibizhang/transparent-context/middleware/httpmiddleware"
)

func main() {
	// Give services time to start
	time.Sleep(2 * time.Second)

	fmt.Println("--------------------------------------------------")
	fmt.Println("Service A: Starting request...")

	// Create context with initial transparent context
	// We manually simulate LoadFromReqMetadata because we are the initiator
	tc := transparentcontext.NewTransparentContext()
	tc.SetReqAllByKey("TraceID", "1234567890")
	tc.SetReqOnceByKey("From", "ServiceA")
	ctx := transparentcontext.WithTransparentContext(context.Background(), tc)

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/hello", nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}

	// Inject headers
	httpmiddleware.InjectToHttpClientHeader(ctx, req.Header)

	fmt.Printf("[Service A] Sending Req-All-TraceID: %s\n", req.Header.Get("Req-All-TraceID"))
	fmt.Printf("[Service A] Sending Req-Once-From: %s\n", req.Header.Get("Req-Once-From"))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[Service A] Response Body: %s\n", string(body))

	// Check response headers
	fmt.Println("[Service A] Response Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Printf("[Service A] Received Resp-All-TraceID: %s\n", resp.Header.Get("Resp-All-TraceID"))
	fmt.Printf("[Service A] Received Resp-Once-From: %s\n", resp.Header.Get("Resp-Once-From"))
	fmt.Println("--------------------------------------------------")
}
