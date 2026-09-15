package export

// Dataはエクスポートのメールをレンダリングする元になるデータ。
//
// URLはメールが伝える画面を指す。成功したエクスポートではアーカイブのダウンロードを、失敗した
// エクスポートでは理由とやり直しの導線を示すエクスポート画面自身を指す。
//
// ExpirationHoursはダウンロードが可能な期間。テンプレートに書き込まず渡す形にすることで、文面と
// リンクの有効期限がずれないようにする。
type Data struct {
	URL             string
	AppURL          string
	ExpirationHours int
}
