package topic_feed

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/httperror"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// cacheControlは200と304に付けるCache-Control。
// 期限を持たせず、取得のたびにETagで確かめさせる。トピックを非公開にしたときに、
// キャッシュから古い内容が出続けないようにするためである。
const cacheControl = "no-cache"

// ShowはGETまたはHEAD /s/{space_identifier}/topics/{topic_number}/atom - トピックのフィードを返す
//
// 処理の流れ:
//  1. UseCaseで、公開トピックのページと、絶対URLで描画した本文を取得する
//  2. ViewModelでAtomの文書を組み立て、XMLに書き出す
//  3. 書き出したXMLのダイジェストをETagにし、If-None-Matchが一致すれば304を返す
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	// トピックを公開した後や障害の解消後に、キャッシュが古いエラーを返し続けないようにする。
	w.Header().Set("Cache-Control", "private, no-store")

	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	topicNumber, err := strconv.ParseInt(chi.URLParam(r, "topic_number"), 10, 32)
	if err != nil {
		httperror.NotFound(w, r)
		return
	}

	output, err := h.getTopicFeedUC.Execute(ctx, usecase.GetTopicFeedInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
	})
	if err != nil {
		var ae *model.AppError
		if errors.As(err, &ae) {
			if ae.Code == model.AppErrCodeResourceNotFound {
				httperror.NotFound(w, r)
				return
			}
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		slog.ErrorContext(ctx, "トピックのフィードの取得に失敗", "error", err, "space_identifier", string(spaceIdentifier), "topic_number", topicNumber)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	vmSpaceIdentifier := viewmodel.NewSpaceIdentifier(output.Space.Identifier)
	feed := viewmodel.NewTopicAtomFeed(h.cfg, output.Topic, viewmodel.AtomFeedSource{
		Space:               output.Space,
		Pages:               output.Pages,
		ContentHTMLByPageID: output.ContentHTMLByPageID,
		AlternatePath:       string(templates.TopicPath(vmSpaceIdentifier, output.Topic.Number)),
		SelfPath:            string(templates.TopicAtomPath(vmSpaceIdentifier, output.Topic.Number)),
	})

	body, err := xml.Marshal(feed)
	if err != nil {
		slog.ErrorContext(ctx, "トピックのフィードの書き出しに失敗", "error", err, "space_identifier", string(output.Space.Identifier), "topic_number", output.Topic.Number)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	body = append([]byte(xml.Header), body...)

	sum := sha256.Sum256(body)
	etag := `"` + base64.RawURLEncoding.EncodeToString(sum[:]) + `"`

	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", etag)
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(body); err != nil {
		slog.WarnContext(ctx, "トピックのフィードの書き込みに失敗", "error", err)
	}
}

// etagMatchesはIf-None-Matchの値がetagと一致するかを返す
//
// If-None-Matchの比較は弱い比較 (RFC 9110 13.1.2) のため、`W/` の有無は区別しない。
// 複数のETagの列挙と、任意の表現に一致する `*` にも対応する。
func etagMatches(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" {
		return false
	}
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
