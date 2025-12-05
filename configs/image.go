package configs

import (
	"os"
)

// GetImagesPath 获取图片/视频保存路径
// 优先使用环境变量 IMAGES_PATH，否则使用 /app/images
func GetImagesPath() string {
	path := os.Getenv("IMAGES_PATH")
	if path == "" {
		path = "/app/images"
	}
	return path
}
