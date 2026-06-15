package mq

import (
	"testing"

	"github.com/rabbitmq/amqp091-go"
)

func TestMarshalAndUnmarshalOutboxHeaders(t *testing.T) {
	headers := amqp091.Table{
		"x-retry-count": int32(2),
		"x-source":      "feed",
	}

	raw, err := marshalOutboxHeaders(headers)
	if err != nil {
		t.Fatalf("marshalOutboxHeaders() error = %v", err)
	}

	got, err := unmarshalOutboxHeaders(raw)
	if err != nil {
		t.Fatalf("unmarshalOutboxHeaders() error = %v", err)
	}
	if got["x-source"] != "feed" {
		t.Fatalf("headers x-source = %v, want feed", got["x-source"])
	}
	if got["x-retry-count"] != float64(2) {
		t.Fatalf("headers x-retry-count = %v, want 2", got["x-retry-count"])
	}
}

func TestBuildVideoPublishOutboxMessage(t *testing.T) {
	message := BuildVideoPublishOutboxMessage(42)
	if message.Exchange != "" {
		t.Fatalf("Exchange = %q, want empty", message.Exchange)
	}
	if message.RoutingKey == "" {
		t.Fatalf("RoutingKey should not be empty")
	}
	if string(message.Publishing.Body) != "42" {
		t.Fatalf("Body = %q, want 42", string(message.Publishing.Body))
	}
	if message.Publishing.ContentType != "text/plain" {
		t.Fatalf("ContentType = %q, want text/plain", message.Publishing.ContentType)
	}
	if message.Publishing.MessageId != "video_publish:42" {
		t.Fatalf("MessageId = %q, want video_publish:42", message.Publishing.MessageId)
	}
}

func TestVideoPublishDedupKey(t *testing.T) {
	got := videoPublishDedupKey("video_publish:42")
	if got != "mq:video_publish:done:video_publish:42" {
		t.Fatalf("videoPublishDedupKey() = %q", got)
	}
}

func TestBuildVideoPublishMessageID(t *testing.T) {
	if got := buildVideoPublishMessageID(7); got != "video_publish:7" {
		t.Fatalf("buildVideoPublishMessageID() = %q", got)
	}
}

func TestCloneHeadersNil(t *testing.T) {
	if got := CloneHeaders(nil); len(got) != 0 {
		t.Fatalf("CloneHeaders(nil) length = %d, want 0", len(got))
	}
}
