package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

func buildAllowedExtSet(allowType string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, ext := range strings.Split(strings.ToLower(allowType), ",") {
		normalizedExt := strings.TrimSpace(ext)
		if normalizedExt == "" {
			continue
		}
		set[normalizedExt] = struct{}{}
	}
	return set
}

func normalizeMIMEType(mimeType string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
}

func detectFileMIMEType(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("文件内容为空")
	}
	return normalizeMIMEType(http.DetectContentType(buffer[:n])), nil
}

func isMIMETypeAllowed(ext, mimeType, allowType string) bool {
	mimeType = normalizeMIMEType(mimeType)
	switch allowType {
	case AllowImageType:
		return strings.HasPrefix(mimeType, "image/")
	case AllowVideoType:
		// 某些 mkv 文件在 sniff 时会落到 octet-stream，保留兼容。
		return strings.HasPrefix(mimeType, "video/") || (ext == ".mkv" && mimeType == "application/octet-stream")
	default:
		return false
	}
}

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
	allowExtSet := buildAllowedExtSet(allowType)
	if _, ok := allowExtSet[ext]; !ok {
		return "", fmt.Errorf("不支持的文件类型，仅允许：%s", allowType)
	}
	mimeType, err := detectFileMIMEType(file)
	if err != nil {
		return "", fmt.Errorf("文件内容校验失败")
	}
	if !isMIMETypeAllowed(ext, mimeType, allowType) {
		return "", fmt.Errorf("文件内容与扩展名不匹配")
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

// DeleteUploadedFile 删除 uploads 目录中的本地文件
func DeleteUploadedFile(fileURL string) error {
	if fileURL == "" {
		return nil
	}

	cleanPath := filepath.Clean(strings.TrimPrefix(fileURL, "/"))
	if cleanPath == "." || cleanPath == "uploads" {
		return fmt.Errorf("无效的文件路径")
	}
	if !strings.HasPrefix(cleanPath, "uploads"+string(filepath.Separator)) && cleanPath != "uploads" {
		return fmt.Errorf("仅支持删除 uploads 目录中的文件")
	}

	if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
