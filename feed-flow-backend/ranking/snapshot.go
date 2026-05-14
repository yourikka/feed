package ranking

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const hotSnapshotTTL = 2 * time.Minute

type HotSnapshotCursor struct {
	AggKey  string `json:"agg_key"`
	Offset  int    `json:"offset"`
	Window  int    `json:"window"`
	Created int64  `json:"created"`
}

func BuildHotSnapshotCursor(offset int) string {
	return EncodeHotSnapshotCursor(HotSnapshotCursor{
		AggKey:  buildAggKeyAt(time.Now(), getHotWindowHours()),
		Offset:  max(offset, 0),
		Window:  getHotWindowHours(),
		Created: time.Now().Unix(),
	})
}

func EncodeHotSnapshotCursor(cursor HotSnapshotCursor) string {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func ParseHotSnapshotCursor(raw string) (HotSnapshotCursor, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return HotSnapshotCursor{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return HotSnapshotCursor{}, false
	}
	var cursor HotSnapshotCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return HotSnapshotCursor{}, false
	}
	if cursor.AggKey == "" || cursor.Offset < 0 || cursor.Window <= 0 {
		return HotSnapshotCursor{}, false
	}
	return cursor, true
}

func IsHotSnapshotCursorExpired(cursor HotSnapshotCursor) bool {
	if cursor.Created <= 0 {
		return true
	}
	return time.Since(time.Unix(cursor.Created, 0)) > hotSnapshotTTL
}

func buildAggKeyAt(now time.Time, window int) string {
	nowHour := now.Unix() / 3600
	return hotAggPrefix + strconv.FormatInt(nowHour, 10) + ":" + strconv.Itoa(window)
}
