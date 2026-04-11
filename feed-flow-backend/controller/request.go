package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func respondInvalidQuery(c *gin.Context, key string) {
	c.JSON(http.StatusOK, gin.H{
		"status_code": 1,
		"status_msg":  key + " 参数错误",
	})
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
