package service

import (
	"context"
	"testing"
	"time"
)

func TestBuildViewerKey(t *testing.T) {
	t.Run("prefer user id", func(t *testing.T) {
		userID := uint(42)
		got := BuildViewerKey(&userID, "device-1")
		if got != "u:42" {
			t.Fatalf("BuildViewerKey() = %q, want %q", got, "u:42")
		}
	})

	t.Run("fallback to client id", func(t *testing.T) {
		got := BuildViewerKey(nil, " device-1 ")
		if got != "c:device-1" {
			t.Fatalf("BuildViewerKey() = %q, want %q", got, "c:device-1")
		}
	})

	t.Run("empty when missing all ids", func(t *testing.T) {
		if got := BuildViewerKey(nil, "   "); got != "" {
			t.Fatalf("BuildViewerKey() = %q, want empty string", got)
		}
	})
}

func TestScoreDeltaForEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventType  string
		progressMs int64
		durationMs int64
		positionMs int64
		want       float64
	}{
		{name: "exposure", eventType: EventExposure, want: 0.2},
		{name: "finish", eventType: EventPlayEnd, want: 2.6},
		{name: "progress high ratio", eventType: EventProgress, progressMs: 9000, durationMs: 10000, want: 1.6},
		{name: "skip early", eventType: EventSkip, positionMs: 1200, want: -0.6},
		{name: "skip normal", eventType: EventSkip, positionMs: 4000, want: -0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scoreDeltaForEvent(tt.eventType, tt.progressMs, tt.durationMs, tt.positionMs)
			if got != tt.want {
				t.Fatalf("scoreDeltaForEvent(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestFinalizeAcceptedVideoEventNoopWithoutRedis(t *testing.T) {
	t.Parallel()

	if err := finalizeAcceptedVideoEvent(context.Background(), "req-1", EventExposure, "u:1", 42, time.Now()); err != nil {
		t.Fatalf("finalizeAcceptedVideoEvent() error = %v", err)
	}
}

func TestBuildBehaviorEventID(t *testing.T) {
	t.Parallel()

	reqEvent := queuedVideoEvent{
		RequestID: "req-42",
		ViewerKey: "u:1",
		VideoID:   9,
		EventType: EventPlayStart,
	}
	if got := buildBehaviorEventID(reqEvent); got != "req:req-42" {
		t.Fatalf("buildBehaviorEventID(request) = %q", got)
	}

	noReqEvent := queuedVideoEvent{
		UserID:     7,
		ViewerKey:  "u:7",
		VideoID:    9,
		EventType:  EventProgress,
		ProgressMs: 1000,
		DurationMs: 5000,
		PositionMs: 1000,
	}
	if got := buildBehaviorEventID(noReqEvent); got == "" {
		t.Fatal("buildBehaviorEventID(without request id) should not be empty")
	}
}
