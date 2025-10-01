package service

import (
	"context"
	"io"
	"net/http"
	"time"

	"golang.org/x/sync/singleflight"
)

type ProxyService struct {
	client      *http.Client
	group       singleflight.Group
	sem         chan struct{}
	throttleBps int
}

func NewProxyService(timeout time.Duration, maxConcurrent int) *ProxyService {
	if maxConcurrent <= 0 {
		maxConcurrent = 16
	}
	return &ProxyService{
		client: &http.Client{Timeout: timeout},
		sem:    make(chan struct{}, maxConcurrent),
	}
}

func (p *ProxyService) SetThrottle(bps int) { p.throttleBps = bps }

func (p *ProxyService) Proxy(ctx context.Context, source string, rw http.ResponseWriter, req *http.Request) error {
	// 限流：并发信号量
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.sem }()

	// singleflight：同一源在同一时刻只拉一次
	_, err, _ := p.group.Do(source, func() (interface{}, error) {
		return nil, p.doProxy(ctx, source, rw, req)
	})
	return err
}

func (p *ProxyService) ProxyWithFilename(ctx context.Context, source, filename string, rw http.ResponseWriter, req *http.Request) error {
	// 限流：并发信号量
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.sem }()

	// singleflight：同一源在同一时刻只拉一次
	_, err, _ := p.group.Do(source, func() (interface{}, error) {
		return nil, p.doProxyWithFilename(ctx, source, rw, req, filename)
	})
	return err
}

func (p *ProxyService) doProxy(ctx context.Context, source string, rw http.ResponseWriter, req *http.Request) error {
	return p.doProxyWithFilename(ctx, source, rw, req, "")
}

func (p *ProxyService) doProxyWithFilename(ctx context.Context, source string, rw http.ResponseWriter, req *http.Request, filename string) error {
	// 组装下游请求，透传 Range、If-None-Match、If-Modified-Since 等
	outReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	for _, h := range []string{"Range", "If-None-Match", "If-Modified-Since", "User-Agent"} {
		if v := req.Header.Get(h); v != "" {
			outReq.Header.Set(h, v)
		}
	}
	resp, err := p.client.Do(outReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// 选择性透传响应头，利于缓存
	copyHeader(rw.Header(), resp.Header, []string{
		"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges",
		"ETag", "Last-Modified", "Cache-Control",
	})

	// 如果提供了文件名，设置正确的 Content-Disposition 头
	if filename != "" {
		rw.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		// 否则透传原始的 Content-Disposition 头
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			rw.Header().Set("Content-Disposition", cd)
		}
	}

	rw.WriteHeader(resp.StatusCode)
	if p.throttleBps <= 0 {
		_, err = io.Copy(rw, resp.Body)
		return err
	}
	// 限速复制
	bufSize := p.throttleBps / 10
	if bufSize < 1024 {
		bufSize = 1024
	}
	buf := make([]byte, bufSize)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := rw.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
		<-ticker.C
	}
	return nil
}

// ProxyWithCount 与 Proxy 相同，但返回传输的总字节数，便于统计
func (p *ProxyService) ProxyWithCount(ctx context.Context, source string, rw http.ResponseWriter, req *http.Request) (int64, error) {
	// 限流：并发信号量
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	defer func() { <-p.sem }()

	// singleflight：同一源在同一时刻只拉一次
	var total int64
	_, err, _ := p.group.Do(source, func() (interface{}, error) {
		// 组装下游请求
		outReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		for _, h := range []string{"Range", "If-None-Match", "If-Modified-Since", "User-Agent"} {
			if v := req.Header.Get(h); v != "" {
				outReq.Header.Set(h, v)
			}
		}
		resp, err := p.client.Do(outReq)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()

		copyHeader(rw.Header(), resp.Header, []string{
			"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges",
			"ETag", "Last-Modified", "Cache-Control", "Content-Disposition",
		})
		rw.WriteHeader(resp.StatusCode)

		if p.throttleBps <= 0 {
			n, err := io.Copy(rw, resp.Body)
			total = n
			return nil, err
		}
		bufSize := p.throttleBps / 10
		if bufSize < 1024 {
			bufSize = 1024
		}
		buf := make([]byte, bufSize)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := rw.Write(buf[:n]); werr != nil {
					return nil, werr
				}
				total += int64(n)
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, rerr
			}
			<-ticker.C
		}
		return nil, nil
	})
	return total, err
}

func copyHeader(dst, src http.Header, keys []string) {
	for _, k := range keys {
		if v := src.Values(k); len(v) > 0 {
			dst[k] = v
		}
	}
}
