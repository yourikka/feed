package mq

import (
	"testing"

	"github.com/rabbitmq/amqp091-go"
)

func TestGetRetryCount(t *testing.T) {
	tests := []struct {
		name    string
		headers amqp091.Table
		want    int
	}{
		{
			name:    "nil headers",
			headers: nil,
			want:    0,
		},
		{
			name: "missing key",
			headers: amqp091.Table{
				"foo": "bar",
			},
			want: 0,
		},
		{
			name: "int32",
			headers: amqp091.Table{
				"x-retry-count": int32(2),
			},
			want: 2,
		},
		{
			name: "string",
			headers: amqp091.Table{
				"x-retry-count": "3",
			},
			want: 3,
		},
		{
			name: "negative",
			headers: amqp091.Table{
				"x-retry-count": -1,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRetryCount(tt.headers)
			if got != tt.want {
				t.Fatalf("GetRetryCount()=%d, want=%d", got, tt.want)
			}
		})
	}
}

func TestClosePublisherChannelResetsState(t *testing.T) {
	publisherState.mu.Lock()
	publisherState.conn = nil
	publisherState.ch = nil
	publisherState.confirmCh = make(chan amqp091.Confirmation, 1)
	publisherState.mu.Unlock()

	publisherState.mu.Lock()
	closePublisherChannel()
	isNil := publisherState.conn == nil && publisherState.ch == nil && publisherState.confirmCh == nil
	publisherState.mu.Unlock()

	if !isNil {
		t.Fatalf("publisher state should be reset to nil")
	}
}
