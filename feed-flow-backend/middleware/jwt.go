package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourikka/feed-flow/util"
)

func JWAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		//从Header中获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"status_code": 1, "status_msg": "未登录"})
			c.Abort()
			return
		}

		//解析Token(格式: Bearer token)
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"status_code": 1, "status_msg": "Token格式错误"})
			c.Abort()
			return
		}

		//验证Token
		claims, err := util.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status_code": 1, "status_msg": "Token无效"})
			c.Abort()
			return
		}

		//将用户ID存入上下文
		c.Set("userId", claims.UserID)
		c.Next()
	}
}
