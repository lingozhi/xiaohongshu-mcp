package configs

import (
	"os"
)

// GetImagesPath 获取图片/视频保存路径
// 优先使用环境变量 IMAGES_PATH，否则使用 /data/images（持久化卷）
func GetImagesPath() string {
	path := os.Getenv("IMAGES_PATH")
	if path == "" {
		path = "/data/images"
	}
	return path
}
