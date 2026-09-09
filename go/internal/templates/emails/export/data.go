package export

// Data is what an export mail is rendered from.
//
// URL points at the screen the mail is about: the download of the archive for a succeeded export,
// and the export screen itself for a failed one, where the reason and a way to try again are shown.
//
// ExpirationHours is how long the download stays available. It is carried in rather than written
// into the templates, so that the wording and the expiry of the link cannot drift apart.
//
// [Ja] Data はエクスポートのメールをレンダリングする元になるデータ。
//
// URL はメールが伝える画面を指す。成功したエクスポートではアーカイブのダウンロードを、失敗した
// エクスポートでは理由とやり直しの導線を示すエクスポート画面自身を指す。
//
// ExpirationHours はダウンロードが可能な期間。テンプレートに書き込まず渡す形にすることで、文面と
// リンクの有効期限がずれないようにする。
type Data struct {
	URL             string
	AppURL          string
	ExpirationHours int
}
