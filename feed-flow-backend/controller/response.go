package controller

import "github.com/gin-gonic/gin"

func respond(c *gin.Context, statusCode int, statusMsg string, data gin.H) {
	resp := gin.H{
		"status_code": statusCode,
		"status_msg":  statusMsg,
	}
	for k, v := range data {
		resp[k] = v
	}
	c.JSON(200, resp)
}

func respondSuccess(c *gin.Context, statusMsg string, data gin.H) {
	respond(c, 0, statusMsg, data)
}

func respondError(c *gin.Context, statusMsg string, data gin.H) {
	respond(c, 1, statusMsg, data)
}
