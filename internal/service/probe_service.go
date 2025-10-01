package service

import (
	"context"
	"io"
	"net/http"
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

func (p *ProbeService) Probe(ctx context.Context, url string) (*ProbeResult, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
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
	res.Filename = filenameFromHeaders(resp)
	return res, nil
}

func filenameFromHeaders(resp *http.Response) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return ""
	}
	// 简单解析 filename="..."
	if i := strings.Index(cd, "filename="); i >= 0 {
		v := cd[i+9:]
		v = strings.Trim(v, "\";")
		return v
	}
	return ""
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
