package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func respondInvalidQuery(c *gin.Context, key string) {
	respondError(c, key+" 参数错误", nil)
}

func parsePositiveUintQuery(c *gin.Context, key string) (uint, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		respondInvalidQuery(c, key)
		return 0, false
	}

	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		respondInvalidQuery(c, key)
		return 0, false
	}
	return uint(parsed), true
}

func parseOptionalUintQuery(c *gin.Context, key string, defaultVal uint) (uint, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultVal, true
	}

	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		respondInvalidQuery(c, key)
		return 0, false
	}
	return uint(parsed), true
}

func parseOptionalPositiveIntQuery(c *gin.Context, key string, defaultVal int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultVal, true
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		respondInvalidQuery(c, key)
		return 0, false
	}
	return parsed, true
}
