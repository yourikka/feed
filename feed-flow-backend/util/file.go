package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 允许的文件类型和大小限制
const (
	// 图片类型
	AllowImageType = ".jpg,.jpeg,.png,.gif"
	MaxImageSize   = 5 * 1024 * 1024 // 5MB
	// 视频类型
	AllowVideoType = ".mp4,.avi,.mov,.mkv"
	MaxVideoSize   = 1024 * 1024 * 1024 // 1GB
)

// SaveUploadedFile 保存上传的文件到本地
// 参数：c Gin上下文、formKey 表单字段名、saveDir 保存目录（如 "avatar"）、allowType 允许的文件类型、maxSize 最大大小
// 返回：文件访问URL（如 "/uploads/avatar/xxx.jpg"）、错误
func SaveUploadedFile(c *gin.Context, formKey, saveDir, allowType string, maxSize int64) (string, error) {
	// 1. 从请求中获取文件
	file, err := c.FormFile(formKey)
	if err != nil {
		return "", fmt.Errorf("未获取到文件")
	}

	// 2. 校验文件大小
	if file.Size > maxSize {
		return "", fmt.Errorf("文件大小超过限制")
	}

	// 3. 校验文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !strings.Contains(allowType, ext) {
		return "", fmt.Errorf("不支持的文件类型，仅允许：%s", allowType)
	}

	// 4. 生成唯一文件名（UUID + 原后缀），避免重名
	fileName := uuid.New().String() + ext
	// 拼接保存路径（相对路径：uploads/saveDir/fileName）
	savePath := filepath.Join("uploads", saveDir, fileName)

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败")
	}

	// 5. 保存文件到本地
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		return "", fmt.Errorf("文件保存失败")
	}

	// 6. 返回文件访问URL（供前端访问）
	fileUrl := fmt.Sprintf("/uploads/%s/%s", saveDir, fileName)
	return fileUrl, nil
}
