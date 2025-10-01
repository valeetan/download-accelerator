package service

import (
	"context"
	"net/url"
	"time"

	"download-accelerator/internal/dao"
	"download-accelerator/internal/signer"
)

type LinkService struct {
	signer  *signer.LinkSigner
	baseURL string
	db      *dao.DB
}

func NewLinkService(signingKey, baseURL string) *LinkService {
	return &LinkService{
		signer:  signer.NewLinkSigner(signingKey),
		baseURL: baseURL,
	}
}

func (s *LinkService) GenerateSignedLink(source string, exp time.Time) (string, error) {
	path, err := s.signer.SignURL(source, exp)
	if err != nil {
		return "", err
	}
	return signer.BuildAbsolute(s.baseURL, path), nil
}

func (s *LinkService) VerifyQuery(values url.Values) bool {
	return s.signer.Verify(values)
}

func (s *LinkService) SaveMeta(ctx context.Context, m *dao.FileMeta) error {
	if s.db == nil {
		return nil
	}
	return s.db.UpsertFileMeta(ctx, m)
}

func (s *LinkService) AttachDB(db *dao.DB) { s.db = db }

func (s *LinkService) RecordLink(ctx context.Context, sourceURL, absolute, fingerprint string, exp time.Time) error {
	if s.db == nil {
		return nil
	}
	// absolute 以 https://host/d?... 形式，取路径与查询作为 signedPath 键
	u, err := url.Parse(absolute)
	if err != nil {
		return nil
	}
	return s.db.InsertLink(ctx, sourceURL, u.Path+"?"+u.RawQuery, fingerprint, exp)
}

func (s *LinkService) DB() *dao.DB { return s.db }
