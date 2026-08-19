package clientutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
	"github.com/stackitcloud/stackit-sdk-go/core/auth"
	"github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestResponseLoggingMiddlewareLogsTraceID(t *testing.T) {
	const traceID = "test-trace-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-trace-id", traceID)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	logPath := filepath.Join(t.TempDir(), "terraform.log")
	t.Setenv("TF_LOG", "JSON")
	t.Setenv("TF_LOG_PATH", logPath)

	ctx := tfsdklog.ContextWithTestLogging(context.Background(), t.Name())
	ctx = tfsdklog.NewRootProviderLogger(ctx)

	rt, err := auth.NoAuth()
	if err != nil {
		t.Fatal(err)
	}
	client := (&DefaultClientFactory{}).NewServiceEnablementV2Client(ctx, &core.ProviderData{
		RoundTripper:                    rt,
		ServiceEnablementCustomEndpoint: server.URL,
	}, &diag.Diagnostics{})
	if client == nil {
		t.Fatal("NewServiceEnablementV2Client() returned nil")
	}

	if _, err := client.ListServiceStatusRegional(ctx, "eu01", "project-id").Execute(); err != nil {
		t.Fatalf("ListServiceStatusRegional().Execute() error = %v", err)
	}

	logOutput, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log output: %v", err)
	}
	if !strings.Contains(string(logOutput), "response data") {
		t.Errorf("log output does not contain response data: %s", logOutput)
	}
	if !strings.Contains(string(logOutput), traceID) {
		t.Errorf("log output does not contain trace ID %q: %s", traceID, logOutput)
	}
}

func TestDefaultClientFactory_NewServiceEnablementV2Client(t *testing.T) {
	type args struct {
		ctx          context.Context
		providerData *core.ProviderData
		diags        *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want v2api.DefaultAPI
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &DefaultClientFactory{}
			if got := f.NewServiceEnablementV2Client(tt.args.ctx, tt.args.providerData, tt.args.diags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewServiceEnablementV2Client() = %v, want %v", got, tt.want)
			}
		})
	}
}
