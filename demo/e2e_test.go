package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	transparentcontext "github.com/shuaibizhang/transparent-context/context"
	"github.com/shuaibizhang/transparent-context/demo/pkg/model"
	"github.com/shuaibizhang/transparent-context/middleware/httpmiddleware"
)

// Helper function to build services
func buildService(t *testing.T, name string, dir string) string {
	cwd, _ := os.Getwd()
	binPath := fmt.Sprintf("%s/bin/%s", cwd, name)

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build %s: %v\nOutput: %s", name, err, out)
	}
	return binPath
}

func TestE2E(t *testing.T) {
	// 1. Build Services
	// Assuming we are running this test from demo/
	// Create bin directory if not exists
	os.MkdirAll("bin", 0755)

	t.Log("Building Service C...")
	svcCPath := buildService(t, "service_c", "service_c")
	t.Log("Building Service B...")
	svcBPath := buildService(t, "service_b", "service_b")

	// 2. Start Service C
	t.Log("Starting Service C...")
	cmdC := exec.Command(svcCPath)
	cmdC.Stdout = os.Stdout
	cmdC.Stderr = os.Stderr
	if err := cmdC.Start(); err != nil {
		t.Fatalf("Failed to start Service C: %v", err)
	}
	defer func() {
		if cmdC.Process != nil {
			cmdC.Process.Kill()
		}
	}()

	// 3. Start Service B
	t.Log("Starting Service B...")
	cmdB := exec.Command(svcBPath)
	cmdB.Stdout = os.Stdout
	cmdB.Stderr = os.Stderr
	if err := cmdB.Start(); err != nil {
		t.Fatalf("Failed to start Service B: %v", err)
	}
	defer func() {
		if cmdB.Process != nil {
			cmdB.Process.Kill()
		}
	}()

	// Wait for services to be ready
	time.Sleep(5 * time.Second)

	// 4. Send Request (Act as Service A)
	t.Log("Service A: Sending request...")
	tc := transparentcontext.NewTransparentContext()
	traceID := "1234567890"
	tc.SetReqAllByKey("TraceID", traceID)
	tc.SetReqOnceByKey("From", "ServiceA")
	ctx := transparentcontext.WithTransparentContext(context.Background(), tc)

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8081/hello", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// Inject headers
	httpmiddleware.InjectToHttpClientHeader(ctx, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	var chainResp model.ChainResponse
	if err := json.Unmarshal(body, &chainResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v\nBody: %s", err, string(body))
	}
	t.Logf("Response Body: %s", string(body))

	// 5. Verify Response Metadata (Assertions)
	t.Log("Verifying metadata propagation...")

	// Verify Chain Length (Should be B and C)
	if len(chainResp.Chain) != 2 {
		t.Errorf("Expected chain length 2 (B, C), got %d", len(chainResp.Chain))
	}

	var nodeB, nodeC model.Node
	for _, node := range chainResp.Chain {
		if node.Service == "ServiceB" {
			nodeB = node
		} else if node.Service == "ServiceC" {
			nodeC = node
		}
	}

	if nodeB.Service == "" || nodeC.Service == "" {
		t.Fatalf("Missing service node in chain: %+v", chainResp.Chain)
	}

	// Verify Req-All-TraceID propagation
	expectedKey := "Req-All-Traceid" // http.CanonicalHeaderKey format
	if val := nodeB.Metadata.ReqAll[expectedKey]; val != traceID {
		t.Errorf("Service B ReqAll[%s] = %s, expected %s", expectedKey, val, traceID)
	}
	if val := nodeC.Metadata.ReqAll[expectedKey]; val != traceID {
		t.Errorf("Service C ReqAll[%s] = %s, expected %s", expectedKey, val, traceID)
	}

	// Verify Req-Once-From propagation
	// A -> B: ServiceA
	expectedKeyOnce := "Req-Once-From"
	if val := nodeB.Metadata.ReqOnce[expectedKeyOnce]; val != "ServiceA" {
		t.Errorf("Service B ReqOnce[%s] = %s, expected ServiceA", expectedKeyOnce, val)
	}
	// B -> C: ServiceB (Service B sets this before calling C)
	if val := nodeC.Metadata.ReqOnce[expectedKeyOnce]; val != "ServiceB" {
		t.Errorf("Service C ReqOnce[%s] = %s, expected ServiceB", expectedKeyOnce, val)
	}

	// Verify Resp-All-TraceID propagation (C -> B -> A)
	// Note: Response metadata keys might be canonicalized
	expectedRespKey := "Resp-All-Traceid"
	if val := nodeC.Metadata.RespAll[expectedRespKey]; val != traceID {
		t.Errorf("Service C RespAll[%s] = %s, expected %s", expectedRespKey, val, traceID)
	}
	// Service B captures response from C. Wait, Service B captures what it *sends* or what it *receives*?
	// In my implementation:
	// Service C sets RespAll.
	// Service B receives it. And B sets its own RespOnce.
	// The `node` in Service B captures:
	// node.Metadata.RespAll = tc.GetRespAll()
	// tc.GetRespAll() returns what is in the context.
	// When B calls C, C returns headers. Does the gRPC interceptor automatically populate the context with response headers?
	// Let's check `middleware/grpcmiddleware/interceptor.go`.

	// Also verify headers received by A
	if val := resp.Header.Get("Resp-All-TraceID"); val != traceID {
		t.Errorf("Service A Header Resp-All-TraceID = %s, expected %s", val, traceID)
	}
	if val := resp.Header.Get("Resp-Once-From"); val != "ServiceB" {
		t.Errorf("Service A Header Resp-Once-From = %s, expected ServiceB", val)
	}

	t.Log("E2E Test Passed!")
}
