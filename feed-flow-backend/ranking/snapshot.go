package ranking

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/yourikka/feed-flow/util"
)

const hotSnapshotTTL = 2 * time.Minute

const hotSnapshotSeparator = "."

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
	secret, err := util.GetSnapshotSecret()
	if err != nil {
		return ""
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := util.SignPayload(secret, payload)
	return encodedPayload + hotSnapshotSeparator + signature
}

func ParseHotSnapshotCursor(raw string) (HotSnapshotCursor, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return HotSnapshotCursor{}, false
	}
	parts := strings.Split(raw, hotSnapshotSeparator)
	if len(parts) != 2 {
		return HotSnapshotCursor{}, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return HotSnapshotCursor{}, false
	}
	secret, err := util.GetSnapshotSecret()
	if err != nil {
		return HotSnapshotCursor{}, false
	}
	if err := util.VerifyPayloadSignature(secret, decoded, parts[1]); err != nil {
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
