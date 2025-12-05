package downloader

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/configs"
)

// VideoDownloader 视频下载器
type VideoDownloader struct {
	savePath   string
	httpClient *http.Client
}

// NewVideoDownloader 创建视频下载器
func NewVideoDownloader() *VideoDownloader {
	savePath := configs.GetImagesPath()
	logrus.Infof("视频保存路径: %s", savePath)
	if err := os.MkdirAll(savePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create save path: %v", err))
	}

	return &VideoDownloader{
		savePath: savePath,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ProcessVideo 处理视频路径，支持 URL 下载或本地路径
func (d *VideoDownloader) ProcessVideo(videoPath string) (string, error) {
	if IsVideoURL(videoPath) {
		logrus.Infof("检测到视频 URL，开始下载: %s", videoPath)
		localPath, err := d.DownloadVideo(videoPath)
		if err != nil {
			return "", err
		}
		logrus.Infof("视频下载完成: %s", localPath)
		return localPath, nil
	}
	// 本地路径直接返回
	if _, err := os.Stat(videoPath); err != nil {
		return "", errors.Wrapf(err, "视频文件不存在: %s", videoPath)
	}
	logrus.Infof("使用本地视频: %s", videoPath)
	return videoPath, nil
}

// DownloadVideo 下载视频到本地
func (d *VideoDownloader) DownloadVideo(videoURL string) (string, error) {
	if !d.isValidURL(videoURL) {
		return "", errors.New("invalid video URL format")
	}

	resp, err := d.httpClient.Get(videoURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to download video")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// 生成文件名
	ext := d.getExtension(videoURL, resp.Header.Get("Content-Type"))
	fileName := d.generateFileName(videoURL, ext)
	filePath := filepath.Join(d.savePath, fileName)

	// 如果文件已存在，直接返回
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to create video file")
	}
	defer file.Close()

	// 流式写入，避免大文件占用过多内存
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(filePath) // 删除不完整的文件
		return "", errors.Wrap(err, "failed to save video")
	}

	return filePath, nil
}

func (d *VideoDownloader) isValidURL(rawURL string) bool {
	if !strings.HasPrefix(strings.ToLower(rawURL), "http://") &&
		!strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return false
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsedURL.Scheme != "" && parsedURL.Host != ""
}

func (d *VideoDownloader) getExtension(videoURL, contentType string) string {
	// 尝试从 Content-Type 获取
	switch contentType {
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	case "video/webm":
		return "webm"
	}

	// 尝试从 URL 获取
	parsedURL, _ := url.Parse(videoURL)
	if parsedURL != nil {
		ext := filepath.Ext(parsedURL.Path)
		if ext != "" {
			return strings.TrimPrefix(ext, ".")
		}
	}

	// 默认 mp4
	return "mp4"
}

func (d *VideoDownloader) generateFileName(videoURL, extension string) string {
	hash := sha256.Sum256([]byte(videoURL))
	hashStr := fmt.Sprintf("%x", hash)
	shortHash := hashStr[:16]
	timestamp := time.Now().Unix()
	return fmt.Sprintf("video_%s_%d.%s", shortHash, timestamp, extension)
}

// IsVideoURL 判断是否为视频 URL
func IsVideoURL(path string) bool {
	return strings.HasPrefix(strings.ToLower(path), "http://") ||
		strings.HasPrefix(strings.ToLower(path), "https://")
}
