package service

import (
	"net/url"
	"time"

	"download-accelerator/internal/signer"
)

type LinkService struct {
	signer  *signer.LinkSigner
	baseURL string
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
