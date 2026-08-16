package templates

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// Path はURLのパスを表す型です
type Path string

// SpacePath はスペースのパスを生成します
func SpacePath(identifier viewmodel.SpaceIdentifier) Path {
	return Path("/s/" + string(identifier))
}

// PaginatedPath appends the offset pagination query to path, and returns path unchanged for the
// first page. It is used for the canonical URL of a paginated screen: each page of the series
// carries different content, so it declares itself rather than the first page as its canonical
// address, while the first page keeps the bare path it is linked by everywhere else. A path that
// already carries a query (SearchPathWithSpaceFilter, for one) gets the parameter appended with
// "&", so that the result stays a valid URL.
//
// [Ja] PaginatedPath はオフセットページネーションのクエリを path に付けて返し、1 ページ目は path を
// そのまま返す。ページネーションされた画面の正規 URL に使う。系列の各ページは内容が異なるため、
// 1 ページ目ではなく自分自身を正規アドレスとして宣言する。1 ページ目は他の箇所からリンクされる
// ときと同じクエリ無しのパスのままにする。既にクエリを持つパス (SearchPathWithSpaceFilter など) には
// "&" で連結し、結果が妥当な URL のままになるようにする。
func PaginatedPath(path Path, page int32) Path {
	if page <= 1 {
		return path
	}

	separator := "?"
	if strings.Contains(string(path), "?") {
		separator = "&"
	}

	return Path(fmt.Sprintf("%s%spage=%d", path, separator, page))
}

// NewSpacePath generates the path to the new space form (currently proxied to the Rails version).
// [Ja] 新規スペース作成フォームのパスを生成します (現状は Rails 版にプロキシされる)。
func NewSpacePath() Path {
	return Path("/spaces/new")
}

// AtomPath generates the path to the space RSS (Atom) feed.
// [Ja] AtomPath はスペースの RSS (Atom) フィードのパスを生成します。
func AtomPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/atom", spaceIdentifier))
}

// TrashPath generates the path to the space trash (currently proxied to the Rails version).
// [Ja] TrashPath はスペースのゴミ箱のパスを生成します (現状は Rails 版にプロキシされる)。
func TrashPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/trash", spaceIdentifier))
}

// SpaceSettingsPath generates the path to the space settings (currently proxied to the Rails version).
// [Ja] SpaceSettingsPath はスペース設定のパスを生成します (現状は Rails 版にプロキシされる)。
func SpaceSettingsPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/settings", spaceIdentifier))
}

// HomePath はホームのパスを生成します
func HomePath() Path {
	return Path("/home")
}

// TopicPath はトピックのパスを生成します
func TopicPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber))
}

// TopicSettingsPath はトピック設定のパスを生成します
func TopicSettingsPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/settings", spaceIdentifier, topicNumber))
}

// NewTopicPath generates the path to the new topic form (currently proxied to the Rails version).
// [Ja] 新規トピック作成フォームのパスを生成します (現状は Rails 版にプロキシされる)。
func NewTopicPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/topics/new", spaceIdentifier))
}

// NewPagePath はページ新規作成のパスを生成します
func NewPagePath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/pages/new", spaceIdentifier, topicNumber))
}

// PagePath はページのパスを生成します
func PagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber viewmodel.PageNumber) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// PageDraftPagePath は下書きページのパスを生成します
func PageDraftPagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page", spaceIdentifier, pageNumber))
}

// PageDraftPageRevisionPath は下書きリビジョン手動保存のパスを生成します
func PageDraftPageRevisionPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revision", spaceIdentifier, pageNumber))
}

// PageDraftPageRevisionShowPath generates the path to one draft revision (the diff fragment).
// [Ja] PageDraftPageRevisionShowPath は下書きリビジョン単件 (差分フラグメント) のパスを生成します。
func PageDraftPageRevisionShowPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, revisionID string) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revisions/%s", spaceIdentifier, pageNumber, revisionID))
}

// PageDraftPageRevisionRestorePath generates the path to restore a draft revision.
// [Ja] PageDraftPageRevisionRestorePath は下書きリビジョン復元のパスを生成します。
func PageDraftPageRevisionRestorePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, revisionID string) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revisions/%s/restore", spaceIdentifier, pageNumber, revisionID))
}

// PagePreviewPath generates the path to the page preview.
// [Ja] PagePreviewPath はページプレビューのパスを生成します。
func PagePreviewPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/preview", spaceIdentifier, pageNumber))
}

// SearchPath は検索のパスを生成します
func SearchPath() Path {
	return Path("/search")
}

// SearchPathWithSpaceFilter はスペースフィルター付きの検索パスを生成します
func SearchPathWithSpaceFilter(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path("/search?q=space:" + string(spaceIdentifier))
}

// SearchPathFor は現在のスペースに応じた検索パスを生成します
// スペース内ならスペースフィルター付き、スペース外なら素の `/search` を返します
func SearchPathFor(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	if spaceIdentifier != "" {
		return SearchPathWithSpaceFilter(spaceIdentifier)
	}
	return SearchPath()
}

// ProfilePath はプロフィールのパスを生成します
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SignInPath はサインインのパスを生成します
func SignInPath() Path {
	return Path("/sign_in")
}

// TopPath generates the path to the public top page.
//
// [Ja] TopPath は公開トップページのパスを生成する。
func TopPath() Path {
	return Path("/")
}

// SignUpPath generates the path to the sign-up form.
//
// [Ja] SignUpPath は新規登録フォームのパスを生成する。
func SignUpPath() Path {
	return Path("/sign_up")
}

// PasswordResetPath generates the path to the password reset request form.
//
// [Ja] PasswordResetPath はパスワードリセット申請フォームのパスを生成する。
func PasswordResetPath() Path {
	return Path("/password/reset")
}

// PageLinkListPath はリンク一覧のパスを生成します
func PageLinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/link_list", spaceIdentifier, pageNumber))
}

// PageBacklinkListPath はバックリンク一覧のパスを生成します
func PageBacklinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, linkedPageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/links/%d/backlink_list", spaceIdentifier, pageNumber, linkedPageNumber))
}

// PageBacklinksPath はページレベルのバックリンク一覧のパスを生成します
func PageBacklinksPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/backlinks", spaceIdentifier, pageNumber))
}

// PageEditPath はページ編集のパスを生成します
func PageEditPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/edit", spaceIdentifier, pageNumber))
}

// PageMovePath はページ移動のパスを生成します
func PageMovePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/move", spaceIdentifier, pageNumber))
}

// PageTrashPath generates the path that moves a page into the trash. It is the POST destination of
// the page actions form, not a screen: the space's trash screen is TrashPath.
//
// [Ja] PageTrashPath はページをゴミ箱へ入れるパスを生成します。画面ではなくページ操作フォームの
// POST 先で、スペースのゴミ箱画面は TrashPath です。
func PageTrashPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/trash", spaceIdentifier, pageNumber))
}

// AttachmentOGImagePath generates the path to the og:image delivery endpoint of an attachment. The
// endpoint re-evaluates on every request whether the referencing pages are public, so the path
// stays valid past the cache lifetime of the HTML that carries it.
//
// [Ja] AttachmentOGImagePath は添付ファイルの og:image 配信エンドポイントのパスを生成します。
// エンドポイントはリクエストのたびに参照元ページが公開かを再評価するため、このパスを載せた HTML の
// キャッシュ寿命を超えても無効化されません。
func AttachmentOGImagePath(attachmentID string) Path {
	return Path(fmt.Sprintf("/attachments/%s/og_image", attachmentID))
}

// SuggestionListPath は編集提案一覧のパスを生成します
func SuggestionListPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
}

// SuggestionListTabPath appends the status tab query to the suggestion list path, and returns the
// path unchanged for the default (open) tab. Each tab lists a different set of suggestions, so it
// is used for the canonical URL as well: the closed tab declares itself rather than the open tab,
// while the open tab keeps the bare path it is linked by everywhere else. The tab is taken as a
// bool rather than the raw query value so that an unknown value, which renders the open tab, does
// not end up in the canonical URL.
//
// [Ja] SuggestionListTabPath はステータスタブのクエリを編集提案一覧のパスに付けて返し、既定
// (オープン) のタブは path をそのまま返す。タブごとに載っている編集提案が異なるため正規 URL にも
// 使う。クローズタブは自分自身を宣言し、オープンタブは他の箇所からリンクされるときと同じクエリ無しの
// パスのままにする。タブを生のクエリ値ではなく bool で受け取るのは、オープンタブを描画する未知の値が
// 正規 URL に載らないようにするためである。
func SuggestionListTabPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32, showClosed bool) Path {
	path := SuggestionListPath(spaceIdentifier, topicNumber)
	if !showClosed {
		return path
	}

	return Path(string(path) + "?tab=closed")
}

// SuggestionShowPath は編集提案詳細のパスを生成します
func SuggestionShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d", spaceIdentifier, suggestionNumber))
}

// SuggestionNewPath は編集提案作成のパスを生成します
func SuggestionNewPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions/new", spaceIdentifier, topicNumber))
}

// SuggestionChangesPath は編集提案の変更差分のパスを生成します
func SuggestionChangesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/changes", spaceIdentifier, suggestionNumber))
}

// SuggestionApplyPath は編集提案反映のパスを生成します
func SuggestionApplyPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/apply", spaceIdentifier, suggestionNumber))
}

// SuggestionClosePath は編集提案クローズのパスを生成します
func SuggestionClosePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/close", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentsPath は編集提案コメント作成のパスを生成します
func SuggestionCommentsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditsPath は編集提案ページの編集開始のパスを生成します
func SuggestionPageEditsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditShowPath は編集提案ページ編集の確認画面のパスを生成します
func SuggestionPageEditShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionEditPath は編集提案編集のパスを生成します
func SuggestionEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/edit", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentPath は編集提案コメントのパスを生成します
func SuggestionCommentPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionCommentEditPath は編集提案コメント編集のパスを生成します
func SuggestionCommentEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d/edit", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionPagePath は編集提案ページのパスを生成します
func SuggestionPagePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionPagesPath は編集提案ページ一覧のパスを生成します
func SuggestionPagesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages", spaceIdentifier, suggestionNumber))
}

// SuggestionPageNewPath は編集提案ページ追加のパスを生成します
func SuggestionPageNewPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/new", spaceIdentifier, suggestionNumber))
}

// DraftsPath は下書き一覧のパスを生成します
func DraftsPath() Path {
	return Path("/drafts")
}
