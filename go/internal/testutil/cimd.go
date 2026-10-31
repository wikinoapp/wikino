package testutil

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/wikinoapp/wikino/go/internal/cimd"
)

// CIMDStubは、CIMDの文書の取得の通信を差し替えるスタブ。Documentsに登録したURLには200で本文を、
// それ以外のURLには404を返し、外部へは接続しない
type CIMDStub struct {
	mu        sync.Mutex
	documents map[string]string
	requests  []string
}

// NewCIMDStubは、URLごとの文書の本文documentsを返すCIMDStubを生成します。nilなら常に404を返します
func NewCIMDStub(documents map[string]string) *CIMDStub {
	return &CIMDStub{documents: documents}
}

// Fetcherは、このスタブを通信の経路にしたcimd.Fetcherを返します
func (s *CIMDStub) Fetcher() *cimd.Fetcher {
	return cimd.NewFetcher(&http.Client{Transport: s})
}

// SetDocumentは、URLurlの文書の本文を差し替えます
func (s *CIMDStub) SetDocument(url, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.documents == nil {
		s.documents = map[string]string{}
	}
	s.documents[url] = body
}

// Requestsは、これまでに取得したURLを順に返します
func (s *CIMDStub) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

// RoundTripはhttp.RoundTripperを満たす
func (s *CIMDStub) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req.URL.String())
	body, ok := s.documents[req.URL.String()]
	status := http.StatusOK
	if !ok {
		status = http.StatusNotFound
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}
