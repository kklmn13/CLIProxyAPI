package requestlogctx

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/logging"
	log "github.com/sirupsen/logrus"
)

const (
	APIRequestKey           = "API_REQUEST"
	APIResponseKey          = "API_RESPONSE"
	APIWebsocketTimelineKey = "API_WEBSOCKET_TIMELINE"
	APIResponseTimestampKey = "API_RESPONSE_TIMESTAMP"
	APIResponseErrorKey     = "API_RESPONSE_ERROR"
)

type FileBodySourceFactory interface {
	NewFileBodySource(prefix string) (*logging.FileBodySource, error)
}

func FromContext(ctx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}
	ginCtx, _ := ctx.Value("gin").(*gin.Context)
	return ginCtx
}

func APIRequest(c *gin.Context) []byte {
	return bytesValue(c, APIRequestKey, false)
}

func SetAPIRequest(c *gin.Context, data []byte) {
	setBytes(c, APIRequestKey, data, false)
}

func APIResponse(c *gin.Context) []byte {
	return bytesValue(c, APIResponseKey, false)
}

func SetAPIResponse(c *gin.Context, data []byte) {
	setBytes(c, APIResponseKey, data, true)
}

func AppendAPIResponse(c *gin.Context, data []byte) {
	appendBytes(c, APIResponseKey, data, false)
	MarkAPIResponseTimestamp(c)
}

func APIWebsocketTimeline(c *gin.Context) []byte {
	return bytesValue(c, APIWebsocketTimelineKey, true)
}

func SetAPIWebsocketTimeline(c *gin.Context, data []byte) {
	setBytes(c, APIWebsocketTimelineKey, data, true)
}

func AppendAPIWebsocketTimeline(c *gin.Context, data []byte) {
	if c == nil {
		return
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return
	}
	if source, ok := APIWebsocketTimelineSource(c); ok {
		if errAppend := source.AppendPart(data); errAppend == nil {
			return
		} else {
			log.WithError(errAppend).Warn("failed to append api websocket timeline log part")
		}
	}
	appendBytes(c, APIWebsocketTimelineKey, data, true)
}

func MarkAPIResponseTimestamp(c *gin.Context) {
	if c == nil {
		return
	}
	if _, exists := c.Get(APIResponseTimestampKey); exists {
		return
	}
	c.Set(APIResponseTimestampKey, time.Now())
}

func SetAPIResponseTimestamp(c *gin.Context, timestamp time.Time) {
	if c == nil || timestamp.IsZero() {
		return
	}
	c.Set(APIResponseTimestampKey, timestamp)
}

func APIResponseTimestamp(c *gin.Context) time.Time {
	if c == nil {
		return time.Time{}
	}
	value, exists := c.Get(APIResponseTimestampKey)
	if !exists {
		return time.Time{}
	}
	timestamp, ok := value.(time.Time)
	if !ok {
		return time.Time{}
	}
	return timestamp
}

func AppendAPIResponseError(c *gin.Context, err *interfaces.ErrorMessage) {
	if c == nil || err == nil {
		return
	}
	errors := APIResponseErrors(c)
	errors = append(errors, err)
	c.Set(APIResponseErrorKey, errors)
}

func APIResponseErrors(c *gin.Context) []*interfaces.ErrorMessage {
	if c == nil {
		return nil
	}
	value, exists := c.Get(APIResponseErrorKey)
	if !exists {
		return nil
	}
	errors, ok := value.([]*interfaces.ErrorMessage)
	if !ok {
		return nil
	}
	return errors
}

func SetWebsocketTimelineSource(c *gin.Context, source *logging.FileBodySource) {
	if c == nil || source == nil {
		return
	}
	c.Set(logging.WebsocketTimelineSourceContextKey, source)
}

func WebsocketTimelineSource(c *gin.Context) (*logging.FileBodySource, bool) {
	return fileBodySource(c, logging.WebsocketTimelineSourceContextKey)
}

func SetAPIWebsocketTimelineSource(c *gin.Context, source *logging.FileBodySource) {
	if c == nil || source == nil {
		return
	}
	c.Set(logging.APIWebsocketTimelineSourceContextKey, source)
}

func APIWebsocketTimelineSource(c *gin.Context) (*logging.FileBodySource, bool) {
	return fileBodySource(c, logging.APIWebsocketTimelineSourceContextKey)
}

func AttachResponsesWebsocketSources(c *gin.Context, logger logging.RequestLogger, loggerEnabled bool) {
	if c == nil || !loggerEnabled || !IsResponsesWebsocketUpgrade(c.Request) {
		return
	}
	factory, ok := logger.(FileBodySourceFactory)
	if !ok || factory == nil {
		return
	}
	if source, errSource := factory.NewFileBodySource("websocket-timeline"); errSource == nil {
		SetWebsocketTimelineSource(c, source)
	}
	if source, errSource := factory.NewFileBodySource("api-websocket-timeline"); errSource == nil {
		SetAPIWebsocketTimelineSource(c, source)
	}
}

func IsResponsesWebsocketUpgrade(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	if req.URL.Path != "/v1/responses" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket")
}

func bytesValue(c *gin.Context, key string, clone bool) []byte {
	if c == nil {
		return nil
	}
	value, exists := c.Get(key)
	if !exists {
		return nil
	}
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	if clone {
		return bytes.Clone(data)
	}
	return data
}

func setBytes(c *gin.Context, key string, data []byte, clone bool) {
	if c == nil || len(data) == 0 {
		return
	}
	if clone {
		data = bytes.Clone(data)
	}
	c.Set(key, data)
}

func appendBytes(c *gin.Context, key string, data []byte, blankLine bool) {
	if c == nil {
		return
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return
	}
	existing := bytesValue(c, key, false)
	if len(existing) == 0 {
		c.Set(key, bytes.Clone(data))
		return
	}
	separator := 1
	if blankLine {
		separator = 2
	}
	combined := make([]byte, 0, len(existing)+len(data)+separator)
	combined = append(combined, existing...)
	if !bytes.HasSuffix(existing, []byte("\n")) {
		combined = append(combined, '\n')
	}
	if blankLine {
		combined = append(combined, '\n')
	}
	combined = append(combined, data...)
	c.Set(key, combined)
}

func fileBodySource(c *gin.Context, key string) (*logging.FileBodySource, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}
	source, ok := value.(*logging.FileBodySource)
	return source, ok && source != nil
}
