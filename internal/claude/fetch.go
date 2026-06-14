package claude

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultAPIBase = "https://api.github.com"
	fetchTimeout   = 30 * time.Second
)

// Fetcher 从 GitHub 按 tag 拉取规范（解析 commit sha + 下载 tarball）。
type Fetcher struct {
	client  *http.Client
	apiBase string
	token   string
}

// FetcherOption 配置 Fetcher。
type FetcherOption func(*Fetcher)

// WithToken 设置私有仓库访问令牌（公开仓库可不设）。
func WithToken(t string) FetcherOption {
	return func(f *Fetcher) {
		if t != "" {
			f.token = t
		}
	}
}

// WithAPIBase 覆盖 GitHub API 基址（主要供测试注入 httptest server）。
func WithAPIBase(u string) FetcherOption {
	return func(f *Fetcher) {
		if u != "" {
			f.apiBase = u
		}
	}
}

// NewFetcher 创建 Fetcher。
func NewFetcher(opts ...FetcherOption) *Fetcher {
	f := &Fetcher{
		client:  &http.Client{Timeout: fetchTimeout},
		apiBase: defaultAPIBase,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// ResolveTag 把精确 tag 解析为 commit sha（作为 lock 的唯一完整性锚点）。
func (f *Fetcher) ResolveTag(ctx context.Context, ref RepoRef, tag string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits/%s", f.apiBase, ref.Owner, ref.Repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("claude: build resolve request: %w", err)
	}
	// 该 Accept 让 GitHub 直接返回 commit sha 纯文本
	req.Header.Set("Accept", "application/vnd.github.sha")
	f.auth(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude: resolve tag %q: %w", tag, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude: resolve tag %q: github status %d", tag, resp.StatusCode)
	}
	sha, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("claude: read sha: %w", err)
	}
	return string(sha), nil
}

// DownloadTarball 返回 tag 对应 tarball 的读取流，调用方负责 Close。
func (f *Fetcher) DownloadTarball(ctx context.Context, ref RepoRef, tag string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tarball/%s", f.apiBase, ref.Owner, ref.Repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("claude: build tarball request: %w", err)
	}
	f.auth(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: download tarball %q: %w", tag, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("claude: download tarball %q: github status %d", tag, resp.StatusCode)
	}
	return resp.Body, nil
}

func (f *Fetcher) auth(req *http.Request) {
	if f.token != "" {
		req.Header.Set("Authorization", "Bearer "+f.token)
	}
}
