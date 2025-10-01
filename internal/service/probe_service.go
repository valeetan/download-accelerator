package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type ProbeResult struct {
	OK       bool
	Filename string
	Size     int64
	MimeType string
	ETag     string
	LastMod  string
}

type ProbeService struct {
	client *http.Client
}

func NewProbeService(timeout time.Duration) *ProbeService {
	return &ProbeService{client: &http.Client{Timeout: timeout}}
}

func (p *ProbeService) Probe(ctx context.Context, urlStr string) (*ProbeResult, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, urlStr, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return &ProbeResult{OK: false}, nil
	}
	defer safeClose(resp.Body)
	if resp.StatusCode >= 400 {
		return &ProbeResult{OK: false}, nil
	}
	res := &ProbeResult{OK: true}
	res.Size = getSize(resp)
	res.MimeType = resp.Header.Get("Content-Type")
	res.ETag = resp.Header.Get("ETag")
	res.LastMod = resp.Header.Get("Last-Modified")
	res.Filename = extractFilename(urlStr, resp)
	return res, nil
}

// extractFilename 智能提取文件名，优先级：
// 1. Content-Disposition 头中的 filename
// 2. URL 路径中的文件名
// 3. 根据 MIME 类型生成默认文件名
func extractFilename(urlStr string, resp *http.Response) string {
	// 1. 首先尝试从 Content-Disposition 头获取
	if filename := filenameFromHeaders(resp); filename != "" {
		return cleanFilename(filename)
	}

	// 2. 从 URL 路径中提取文件名
	if filename := filenameFromURL(urlStr); filename != "" {
		return cleanFilename(filename)
	}

	// 3. 根据 MIME 类型生成默认文件名
	return generateDefaultFilename(resp.Header.Get("Content-Type"))
}

// filenameFromHeaders 从 Content-Disposition 头提取文件名
func filenameFromHeaders(resp *http.Response) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return ""
	}

	// 支持 filename="..." 和 filename*=UTF-8''... 格式
	if i := strings.Index(cd, "filename="); i >= 0 {
		v := cd[i+9:]
		// 处理引号
		if strings.HasPrefix(v, "\"") {
			if end := strings.Index(v[1:], "\""); end >= 0 {
				return v[1 : end+1]
			}
		}
		// 处理 filename*=UTF-8''... 格式
		if strings.HasPrefix(v, "UTF-8''") {
			decoded, err := url.QueryUnescape(v[7:])
			if err == nil {
				return decoded
			}
		}
		// 简单处理：去除引号和分号
		v = strings.Trim(v, "\";")
		return v
	}
	return ""
}

// filenameFromURL 从 URL 路径中提取文件名
func filenameFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// 获取路径的最后一部分
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) == 0 {
		return ""
	}

	lastPart := pathParts[len(pathParts)-1]

	// 如果最后一部分包含文件扩展名，返回它
	if path.Ext(lastPart) != "" {
		return lastPart
	}

	// 如果路径以文件名结尾，返回它
	if lastPart != "" && !strings.Contains(lastPart, "?") {
		return lastPart
	}

	return ""
}

// generateDefaultFilename 根据 MIME 类型生成默认文件名
func generateDefaultFilename(mimeType string) string {
	// 移除 charset 等参数
	if i := strings.Index(mimeType, ";"); i >= 0 {
		mimeType = mimeType[:i]
	}
	mimeType = strings.TrimSpace(mimeType)

	// MIME 类型到扩展名的映射
	extMap := map[string]string{
		"application/zip":              ".zip",
		"application/x-zip-compressed": ".zip",
		"application/x-rar-compressed": ".rar",
		"application/x-7z-compressed":  ".7z",
		"application/gzip":             ".gz",
		"application/x-tar":            ".tar",
		"application/x-bzip2":          ".bz2",
		"application/x-xz":             ".xz",
		"application/pdf":              ".pdf",
		"application/msword":           ".doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
		"application/vnd.ms-excel": ".xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         ".xlsx",
		"application/vnd.ms-powerpoint":                                             ".ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
		"text/plain":                    ".txt",
		"text/html":                     ".html",
		"text/css":                      ".css",
		"text/javascript":               ".js",
		"application/json":              ".json",
		"application/xml":               ".xml",
		"image/jpeg":                    ".jpg",
		"image/png":                     ".png",
		"image/gif":                     ".gif",
		"image/webp":                    ".webp",
		"image/svg+xml":                 ".svg",
		"video/mp4":                     ".mp4",
		"video/avi":                     ".avi",
		"video/quicktime":               ".mov",
		"video/x-msvideo":               ".avi",
		"audio/mpeg":                    ".mp3",
		"audio/wav":                     ".wav",
		"audio/ogg":                     ".ogg",
		"application/octet-stream":      ".bin",
		"application/x-executable":      ".exe",
		"application/x-msdownload":      ".exe",
		"application/x-msdos-program":   ".exe",
		"application/x-winexe":          ".exe",
		"application/x-iso9660-image":   ".iso",
		"application/x-apple-diskimage": ".dmg",
		"application/vnd.android.package-archive": ".apk",
		"application/x-debian-package":            ".deb",
		"application/x-rpm":                       ".rpm",
	}

	ext, exists := extMap[mimeType]
	if !exists {
		// 尝试从 MIME 类型中提取通用扩展名
		if strings.HasPrefix(mimeType, "application/") {
			ext = ".bin"
		} else if strings.HasPrefix(mimeType, "text/") {
			ext = ".txt"
		} else if strings.HasPrefix(mimeType, "image/") {
			ext = ".img"
		} else if strings.HasPrefix(mimeType, "video/") {
			ext = ".mp4"
		} else if strings.HasPrefix(mimeType, "audio/") {
			ext = ".mp3"
		} else {
			ext = ".bin"
		}
	}

	// 生成带时间戳的默认文件名
	timestamp := time.Now().Format("20060102-150405")
	return "download-" + timestamp + ext
}

// cleanFilename 清理文件名，移除非法字符
func cleanFilename(filename string) string {
	// 移除或替换非法字符
	illegal := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range illegal {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// 限制长度
	if len(filename) > 200 {
		ext := path.Ext(filename)
		name := filename[:200-len(ext)]
		filename = name + ext
	}

	return filename
}

func getSize(resp *http.Response) int64 {
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if n, err := parseInt64(cl); err == nil {
			return n
		}
	}
	return -1
}

func parseInt64(s string) (int64, error) {
	var n int64
	_, err := io.WriteString(&int64Writer{&n}, s)
	return n, err
}

type int64Writer struct{ n *int64 }

func (w *int64Writer) Write(p []byte) (int, error) {
	var x int64
	for _, b := range p {
		if b < '0' || b > '9' {
			continue
		}
		x = x*10 + int64(b-'0')
	}
	*w.n = x
	return len(p), nil
}

func safeClose(c io.Closer) { _ = c.Close() }
