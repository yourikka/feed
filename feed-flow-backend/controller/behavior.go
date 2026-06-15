package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/service"
	"github.com/yourikka/feed-flow/util"
)

type TrackVideoEventRequest struct {
	ClientID   string `json:"client_id"`
	VideoID    uint   `json:"video_id" binding:"required"`
	EventType  string `json:"event_type" binding:"required"`
	RequestID  string `json:"request_id"`
	SessionID  string `json:"session_id"`
	ProgressMs int64  `json:"progress_ms"`
	DurationMs int64  `json:"duration_ms"`
	PositionMs int64  `json:"position_ms"`
}

func TrackVideoEvent(c *gin.Context) {
	var req TrackVideoEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, "参数错误", nil)
		return
	}

	var userID *uint
	if authHeader := strings.TrimSpace(c.GetHeader("Authorization")); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			if claims, err := util.ParseToken(parts[1]); err == nil && claims.UserID > 0 {
				userID = &claims.UserID
			}
		}
	}

	result, err := service.TrackVideoEvent(c.Request.Context(), service.TrackVideoEventInput{
		UserID:     userID,
		ClientID:   strings.TrimSpace(req.ClientID),
		VideoID:    req.VideoID,
		EventType:  req.EventType,
		RequestID:  req.RequestID,
		SessionID:  req.SessionID,
		ProgressMs: req.ProgressMs,
		DurationMs: req.DurationMs,
		PositionMs: req.PositionMs,
	})
	if err != nil {
		respondError(c, err.Error(), nil)
		return
	}

	respondSuccess(c, "上报成功", gin.H{
		"accepted": result.Accepted,
		"deduped":  result.Deduped,
	})
}
