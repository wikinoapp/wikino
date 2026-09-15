package templates

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// PathはURLのパスを表す型です
type Path string

// SpacePathはスペースのパスを生成します
func SpacePath(identifier viewmodel.SpaceIdentifier) Path {
	return Path("/s/" + string(identifier))
}

// PaginatedPathはオフセットページネーションのクエリをpathに付けて返し、1ページ目はpathを
// そのまま返す。ページネーションされた画面の正規URLに使う。系列の各ページは内容が異なるため、
// 1ページ目ではなく自分自身を正規アドレスとして宣言する。1ページ目は他の箇所からリンクされる
// ときと同じクエリ無しのパスのままにする。既にクエリを持つパス (SearchPathWithSpaceFilterなど) には
// "&" で連結し、結果が妥当なURLのままになるようにする。
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

// 新規スペース作成フォームのパスを生成します (現状はRails版にプロキシされる)。
func NewSpacePath() Path {
	return Path("/spaces/new")
}

// AtomPathはスペースのRSS (Atom) フィードのパスを生成します。
func AtomPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/atom", spaceIdentifier))
}

// TrashPathはスペースのゴミ箱のパスを生成します (現状はRails版にプロキシされる)。
func TrashPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/trash", spaceIdentifier))
}

// SpaceSettingsPathはスペース設定のパスを生成します (現状はRails版にプロキシされる)。
func SpaceSettingsPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/settings", spaceIdentifier))
}

// SpaceSettingsExportsPathはスペースのエクスポートのパスを生成します。エクスポートは、この
// パスへのPOSTで開始します。
func SpaceSettingsExportsPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/settings/exports", spaceIdentifier))
}

// NewSpaceSettingsExportPathはエクスポートを開始する画面のパスを生成します。
func NewSpaceSettingsExportPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/settings/exports/new", spaceIdentifier))
}

// SpaceSettingsExportPathはエクスポート1件の経過を追う画面のパスを生成します。
func SpaceSettingsExportPath(spaceIdentifier viewmodel.SpaceIdentifier, exportID string) Path {
	return Path(fmt.Sprintf("/s/%s/settings/exports/%s", spaceIdentifier, url.PathEscape(exportID)))
}

// SpaceSettingsExportDownloadPathはエクスポートのアーカイブをダウンロードするパスを生成します。
func SpaceSettingsExportDownloadPath(spaceIdentifier viewmodel.SpaceIdentifier, exportID string) Path {
	return SpaceSettingsExportPath(spaceIdentifier, exportID) + "/download"
}

// HomePathはホームのパスを生成します
func HomePath() Path {
	return Path("/home")
}

// TopicPathはトピックのパスを生成します
func TopicPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d", spaceIdentifier, topicNumber))
}

// TopicSettingsPathはトピック設定のパスを生成します
func TopicSettingsPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/settings", spaceIdentifier, topicNumber))
}

// TopicSettingsGeneralPathはトピックの一般設定のパスを生成します。設定はこのパスへの送信で
// 保存します。
func TopicSettingsGeneralPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return TopicSettingsPath(spaceIdentifier, topicNumber) + "/general"
}

// NewTopicPathは新規トピック作成フォームのパスを生成します。
func NewTopicPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/topics/new", spaceIdentifier))
}

// TopicListPathはスペースのトピックのパスを生成します。トピックはこのパスへのPOSTで
// 作成します。
func TopicListPath(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path(fmt.Sprintf("/s/%s/topics", spaceIdentifier))
}

// NewPagePathはページ新規作成のパスを生成します
func NewPagePath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/pages/new", spaceIdentifier, topicNumber))
}

// PagePathはページのパスを生成します
func PagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber viewmodel.PageNumber) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d", spaceIdentifier, pageNumber))
}

// PageDraftPagePathは下書きページのパスを生成します
func PageDraftPagePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page", spaceIdentifier, pageNumber))
}

// PageDraftPageRevisionPathは下書きリビジョン手動保存のパスを生成します
func PageDraftPageRevisionPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revision", spaceIdentifier, pageNumber))
}

// PageDraftPageRevisionShowPathは下書きリビジョン単件 (差分フラグメント) のパスを生成します。
func PageDraftPageRevisionShowPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, revisionID string) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revisions/%s", spaceIdentifier, pageNumber, revisionID))
}

// PageDraftPageRevisionRestorePathは下書きリビジョン復元のパスを生成します。
func PageDraftPageRevisionRestorePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, revisionID string) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/draft_page_revisions/%s/restore", spaceIdentifier, pageNumber, revisionID))
}

// PagePreviewPathはページプレビューのパスを生成します。
func PagePreviewPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/preview", spaceIdentifier, pageNumber))
}

// SearchPathは検索のパスを生成します
func SearchPath() Path {
	return Path("/search")
}

// SearchPathWithSpaceFilterはスペースフィルター付きの検索パスを生成します
func SearchPathWithSpaceFilter(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	return Path("/search?q=space:" + string(spaceIdentifier))
}

// SearchPathForは現在のスペースに応じた検索パスを生成します
// スペース内ならスペースフィルター付き、スペース外なら素の `/search` を返します
func SearchPathFor(spaceIdentifier viewmodel.SpaceIdentifier) Path {
	if spaceIdentifier != "" {
		return SearchPathWithSpaceFilter(spaceIdentifier)
	}
	return SearchPath()
}

// ProfilePathはプロフィールのパスを生成します
func ProfilePath(atname string) Path {
	return Path("/@" + atname)
}

// SignInPathはサインインのパスを生成します
func SignInPath() Path {
	return Path("/sign_in")
}

// SignInTwoFactorNewPathは二要素認証コード入力のパスを生成します。backURLが空でなければ
// クエリに載せ、二要素認証を経由してもサインイン後の戻り先が失われないようにします。
func SignInTwoFactorNewPath(backURL string) Path {
	return pathWithBack(Path("/sign_in/two_factor/new"), backURL)
}

// SignInTwoFactorRecoveryNewPathはリカバリーコード入力のパスを生成します。backURLの扱いは
// SignInTwoFactorNewPathと同じです。
func SignInTwoFactorRecoveryNewPath(backURL string) Path {
	return pathWithBack(Path("/sign_in/two_factor/recovery/new"), backURL)
}

// pathWithBackはbackURLをbackクエリパラメータとしてpathに付けて返し、backURLが空の
// ときはpathをそのまま返します。backURLがリダイレクト先として安全かどうかは、最終的に
// リダイレクトするハンドラー (redirect.GetSafeRedirectURL) が判断し、ここでは判断しません。
func pathWithBack(path Path, backURL string) Path {
	if backURL == "" {
		return path
	}

	return Path(string(path) + "?back=" + url.QueryEscape(backURL))
}

// TopPathは公開トップページのパスを生成する。
func TopPath() Path {
	return Path("/")
}

// SignUpPathは新規登録フォームのパスを生成する。
func SignUpPath() Path {
	return Path("/sign_up")
}

// PasswordResetPathはパスワードリセット申請フォームのパスを生成する。
func PasswordResetPath() Path {
	return Path("/password/reset")
}

// PageLinkListPathはリンク一覧のパスを生成します
func PageLinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/link_list", spaceIdentifier, pageNumber))
}

// PageBacklinkListPathはバックリンク一覧のパスを生成します
func PageBacklinkListPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32, linkedPageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/links/%d/backlink_list", spaceIdentifier, pageNumber, linkedPageNumber))
}

// PageBacklinksPathはページレベルのバックリンク一覧のパスを生成します
func PageBacklinksPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/backlinks", spaceIdentifier, pageNumber))
}

// PageEditPathはページ編集のパスを生成します
func PageEditPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/edit", spaceIdentifier, pageNumber))
}

// PageMovePathはページ移動のパスを生成します
func PageMovePath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/move", spaceIdentifier, pageNumber))
}

// PageTrashPathはページをゴミ箱へ入れるパスを生成します。画面ではなくページ操作フォームの
// POST先で、スペースのゴミ箱画面はTrashPathです。
func PageTrashPath(spaceIdentifier viewmodel.SpaceIdentifier, pageNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/pages/%d/trash", spaceIdentifier, pageNumber))
}

// AttachmentOGImagePathは添付ファイルのog:image配信エンドポイントのパスを生成します。
// エンドポイントはリクエストのたびに参照元ページが公開かを再評価するため、このパスを載せたHTMLの
// キャッシュ寿命を超えても無効化されません。
func AttachmentOGImagePath(attachmentID string) Path {
	return Path(fmt.Sprintf("/attachments/%s/og_image", attachmentID))
}

// SuggestionListPathは編集提案一覧のパスを生成します
func SuggestionListPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
}

// SuggestionListTabPathはステータスタブのクエリを編集提案一覧のパスに付けて返し、既定
// (オープン) のタブはpathをそのまま返す。タブごとに載っている編集提案が異なるため正規URLにも
// 使う。クローズタブは自分自身を宣言し、オープンタブは他の箇所からリンクされるときと同じクエリ無しの
// パスのままにする。タブを生のクエリ値ではなくboolで受け取るのは、オープンタブを描画する未知の値が
// 正規URLに載らないようにするためである。
func SuggestionListTabPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32, showClosed bool) Path {
	path := SuggestionListPath(spaceIdentifier, topicNumber)
	if !showClosed {
		return path
	}

	return Path(string(path) + "?tab=closed")
}

// SuggestionShowPathは編集提案詳細のパスを生成します
func SuggestionShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d", spaceIdentifier, suggestionNumber))
}

// SuggestionNewPathは編集提案作成のパスを生成します
func SuggestionNewPath(spaceIdentifier viewmodel.SpaceIdentifier, topicNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions/new", spaceIdentifier, topicNumber))
}

// SuggestionChangesPathは編集提案の変更差分のパスを生成します
func SuggestionChangesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/changes", spaceIdentifier, suggestionNumber))
}

// SuggestionApplyPathは編集提案反映のパスを生成します
func SuggestionApplyPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/apply", spaceIdentifier, suggestionNumber))
}

// SuggestionClosePathは編集提案クローズのパスを生成します
func SuggestionClosePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/close", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentsPathは編集提案コメント作成のパスを生成します
func SuggestionCommentsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditsPathは編集提案ページの編集開始のパスを生成します
func SuggestionPageEditsPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits", spaceIdentifier, suggestionNumber))
}

// SuggestionPageEditShowPathは編集提案ページ編集の確認画面のパスを生成します
func SuggestionPageEditShowPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionEditPathは編集提案編集のパスを生成します
func SuggestionEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/edit", spaceIdentifier, suggestionNumber))
}

// SuggestionCommentPathは編集提案コメントのパスを生成します
func SuggestionCommentPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionCommentEditPathは編集提案コメント編集のパスを生成します
func SuggestionCommentEditPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, commentNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/comments/%d/edit", spaceIdentifier, suggestionNumber, commentNumber))
}

// SuggestionPagePathは編集提案ページのパスを生成します
func SuggestionPagePath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32, suggestionPageID string) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}

// SuggestionPagesPathは編集提案ページ一覧のパスを生成します
func SuggestionPagesPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages", spaceIdentifier, suggestionNumber))
}

// SuggestionPageNewPathは編集提案ページ追加のパスを生成します
func SuggestionPageNewPath(spaceIdentifier viewmodel.SpaceIdentifier, suggestionNumber int32) Path {
	return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/new", spaceIdentifier, suggestionNumber))
}

// DraftsPathは下書き一覧のパスを生成します
func DraftsPath() Path {
	return Path("/drafts")
}
