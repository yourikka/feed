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
			got := getRetryCount(tt.headers)
			if got != tt.want {
				t.Fatalf("getRetryCount()=%d, want=%d", got, tt.want)
			}
		})
	}
}
