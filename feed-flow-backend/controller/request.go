package controller

import (
	"errors"
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

func parseUintCSV(raw string) ([]uint, error) {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	result := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil || parsed == 0 {
			return nil, errors.New("invalid uint csv")
		}
		result = append(result, uint(parsed))
	}
	return result, nil
}
