package ranking

import "testing"

func TestHotSnapshotCursorRoundTrip(t *testing.T) {
	token := EncodeHotSnapshotCursor(HotSnapshotCursor{
		AggKey:  "feed:hot:agg:test",
		Offset:  20,
		Window:  24,
		Created: 100,
	})
	if token == "" {
		t.Fatalf("EncodeHotSnapshotCursor() returned empty token")
	}

	cursor, ok := ParseHotSnapshotCursor(token)
	if !ok {
		t.Fatalf("ParseHotSnapshotCursor() should succeed")
	}
	if cursor.AggKey != "feed:hot:agg:test" || cursor.Offset != 20 || cursor.Window != 24 || cursor.Created != 100 {
		t.Fatalf("unexpected cursor round trip result: %+v", cursor)
	}
}
