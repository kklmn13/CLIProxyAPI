package requestlogctx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
)

func TestFromContextReturnsGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := context.WithValue(context.Background(), "gin", ginCtx)

	if got := FromContext(ctx); got != ginCtx {
		t.Fatalf("FromContext() = %p, want %p", got, ginCtx)
	}
}

func TestAppendAPIResponseMarksTimestampAndUsesSingleNewline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())

	AppendAPIResponse(ginCtx, []byte("first"))
	firstTimestamp := APIResponseTimestamp(ginCtx)
	if firstTimestamp.IsZero() {
		t.Fatalf("timestamp was not set")
	}
	AppendAPIResponse(ginCtx, []byte("second"))

	if got := string(APIResponse(ginCtx)); got != "first\nsecond" {
		t.Fatalf("APIResponse() = %q, want first\\nsecond", got)
	}
	if !APIResponseTimestamp(ginCtx).Equal(firstTimestamp) {
		t.Fatalf("timestamp was overwritten")
	}
}

func TestAppendAPIWebsocketTimelineUsesBlankLineSeparator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())

	AppendAPIWebsocketTimeline(ginCtx, []byte("event-1"))
	AppendAPIWebsocketTimeline(ginCtx, []byte("event-2"))

	if got := string(APIWebsocketTimeline(ginCtx)); got != "event-1\n\nevent-2" {
		t.Fatalf("APIWebsocketTimeline() = %q", got)
	}
}

func TestAppendAPIWebsocketTimelineUsesFileSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	source, err := logging.NewFileBodySourceInDir(t.TempDir(), "api-websocket-timeline")
	if err != nil {
		t.Fatalf("NewFileBodySourceInDir() error = %v", err)
	}
	defer source.Cleanup()
	SetAPIWebsocketTimelineSource(ginCtx, source)

	AppendAPIWebsocketTimeline(ginCtx, []byte("event"))

	if got := APIWebsocketTimeline(ginCtx); got != nil {
		t.Fatalf("in-memory APIWebsocketTimeline() = %q, want nil", string(got))
	}
	body, err := source.Bytes()
	if err != nil {
		t.Fatalf("source Bytes() error = %v", err)
	}
	if string(body) != "event\n" {
		t.Fatalf("source body = %q, want event with newline", string(body))
	}
}

func TestAPIResponseTimestampSetOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	first := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	SetAPIResponseTimestamp(ginCtx, first)
	MarkAPIResponseTimestamp(ginCtx)

	if got := APIResponseTimestamp(ginCtx); !got.Equal(first) {
		t.Fatalf("APIResponseTimestamp() = %v, want %v", got, first)
	}
}

func TestAppendAPIResponseError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	first := &interfaces.ErrorMessage{StatusCode: http.StatusBadGateway, Error: errors.New("first")}
	second := &interfaces.ErrorMessage{StatusCode: http.StatusGatewayTimeout, Error: errors.New("second")}

	AppendAPIResponseError(ginCtx, first)
	AppendAPIResponseError(ginCtx, second)

	errors := APIResponseErrors(ginCtx)
	if len(errors) != 2 || errors[0] != first || errors[1] != second {
		t.Fatalf("APIResponseErrors() = %+v", errors)
	}
}

func TestAttachResponsesWebsocketSources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	ginCtx.Request.Header.Set("Upgrade", "websocket")
	logger := &testSourceFactory{dir: t.TempDir()}

	AttachResponsesWebsocketSources(ginCtx, logger, true)

	if _, ok := WebsocketTimelineSource(ginCtx); !ok {
		t.Fatalf("websocket timeline source was not attached")
	}
	if _, ok := APIWebsocketTimelineSource(ginCtx); !ok {
		t.Fatalf("api websocket timeline source was not attached")
	}
}

type testSourceFactory struct {
	dir string
}

func (f *testSourceFactory) LogRequest(string, string, map[string][]string, []byte, int, map[string][]string, []byte, []byte, []byte, []byte, []byte, []*interfaces.ErrorMessage, string, time.Time, time.Time) error {
	return nil
}

func (f *testSourceFactory) LogStreamingRequest(string, string, map[string][]string, []byte, string) (logging.StreamingLogWriter, error) {
	return nil, nil
}

func (f *testSourceFactory) IsEnabled() bool { return true }

func (f *testSourceFactory) NewFileBodySource(prefix string) (*logging.FileBodySource, error) {
	return logging.NewFileBodySourceInDir(filepath.Join(f.dir, prefix), prefix)
}
