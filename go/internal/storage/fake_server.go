package storage

import (
	"bytes"
	"crypto/md5" //nolint:gosec // Content-MD5の照合に使う。改ざんの検出ではなく、転送の誤りを確かめるためである
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FakeObjectStoragePathPrefixは、NewServedFakeObjectStorageの署名付きURLが指すパスの接頭辞。
// e2eのサーバーは、このパスの下へのリクエストをFakeObjectStorageへ渡す
const FakeObjectStoragePathPrefix = "/_test/storage/"

// ServeHTTPは、FakeObjectStoragePathPrefixの下の署名付きURLへのリクエストを、オブジェクトストレージの代わりに受ける。
// PUTは、署名に含めた種類・大きさ・チェックサムと送られたヘッダーと内容を照らし合わせ、一致すれば保持する。
// GETとHEADは、保持している内容を、署名に含めた応答ヘッダーの指定で返す。
// 本物のストレージと同じく、条件が一致しないアップロードは書き込まずに拒否する
func (f *FakeObjectStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key, ok := strings.CutPrefix(r.URL.Path, FakeObjectStoragePathPrefix)
	if !ok || key == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		f.servePut(w, r, key)
	case http.MethodGet, http.MethodHead:
		f.serveGet(w, r, key)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// servePutは、署名付きURLへのアップロードを受ける
func (f *FakeObjectStorage) servePut(w http.ResponseWriter, r *http.Request, key string) {
	query := r.URL.Query()
	contentType := query.Get("content_type")
	if r.Header.Get("Content-Type") != contentType || r.Header.Get("Content-MD5") != query.Get("content_md5") {
		http.Error(w, "署名に含めたヘッダーと一致しません", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "本文サイズの上限を超えています", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "本文を読めませんでした", http.StatusBadRequest)
		return
	}
	if strconv.Itoa(len(body)) != query.Get("content_length") {
		http.Error(w, "署名に含めた大きさと一致しません", http.StatusForbidden)
		return
	}
	digest := md5.Sum(body) //nolint:gosec // Content-MD5の照合に使う
	if base64.StdEncoding.EncodeToString(digest[:]) != query.Get("content_md5") {
		http.Error(w, "内容がContent-MD5と一致しません", http.StatusBadRequest)
		return
	}

	f.Put(key, body, contentType)
	w.WriteHeader(http.StatusOK)
}

// serveGetは、署名付きURLでの取得に、保持している内容を返す
func (f *FakeObjectStorage) serveGet(w http.ResponseWriter, r *http.Request, key string) {
	body, contentType, ok := f.Object(key)
	if !ok {
		http.NotFound(w, r)
		return
	}

	query := r.URL.Query()
	if v := query.Get("response-content-type"); v != "" {
		contentType = v
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if v := query.Get("response-content-disposition"); v != "" {
		w.Header().Set("Content-Disposition", v)
	}
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
}
