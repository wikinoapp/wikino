package model

import (
	"time"
)

// ExportStatusはエクスポートがどの状態にあるかを表す。
type ExportStatus int32

const (
	// ExportStatusQueuedはジョブの実行待ちのエクスポートの状態
	ExportStatusQueued ExportStatus = 0

	// ExportStatusStartedはワーカーが生成中のエクスポートの状態
	ExportStatusStarted ExportStatus = 1

	// ExportStatusSucceededはZIPがバケットに置かれたエクスポートの状態
	ExportStatusSucceeded ExportStatus = 2

	// ExportStatusFailedは最後の試行が諦めたエクスポートの状態
	ExportStatusFailed ExportStatus = 3
)

// Exportはスペースのエクスポートのドメインモデル。スペース全体をMarkdownファイルと添付
// ファイルのZIPとして書き出す1回の試行を表す。
//
// HeartbeatAtはワーカーが処理中に更新する値。プロセスごと停止させられたエクスポートは
// ExportStatusStartedのまま残り続けるため、heartbeatが古いことが、遅いエクスポートと止まった
// エクスポートを見分ける手がかりになる。
//
// ObjectKeyはオブジェクトストレージ上のZIPを指し、エクスポートの成功時に設定される。Railsが
// 書いたエクスポートでは空のままで、それらのZIPは代わりにActiveStorageのblobのキーの下にある。
type Export struct {
	ID              ExportID
	SpaceID         SpaceID
	QueuedByID      SpaceMemberID
	Status          ExportStatus
	StatusChangedAt time.Time
	HeartbeatAt     *time.Time
	ObjectKey       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// エクスポートの時間に関する5つの値は互いに関係して決まるため、それぞれを読むコードの
// そばではなく1か所に置く。
//
// ExportAttemptTimeoutは1回の試行の上限。Riverはジョブに与えるタイムアウトより
// RescueStuckJobsAfterを長くすることを要求するため、ワーカークライアントは両方をここから決める。
//
// ExportHeartbeatStaleAfterは、startedのエクスポートが実行中と見なされなくなる時間。
// ExportHeartbeatIntervalより十分大きいため、次のheartbeatが遅れているだけのワーカーを
// 止まったものと見なさない。ExportAttemptTimeoutより十分小さいため、止まったエクスポートを
// 置き換えるのに、終わることのない試行の上限を待たずに済む。
//
// ExportQueuedStaleAfterは、queuedのエクスポートが実行中と見なされなくなる時間。heartbeatでは
// 答えられない場合のためにある。ワーカーへ届かなかったジョブのエクスポートは打つべきheartbeatを
// 持たず、それ自身の上限が無ければ、そのスペースが二度とエクスポートできなくなる。
// ExportHeartbeatStaleAfterより大きいのは、ワーカーを待つ時間が、既に動いているワーカーからの
// 鼓動を待つ時間より長いためである。
//
// ExportDownloadExpirationは完了メールのリンクが使える期間。メールの文面はこの値から組み立てる
// ので、両者がずれることはない。
const (
	ExportHeartbeatInterval   = 30 * time.Second
	ExportHeartbeatStaleAfter = 5 * time.Minute
	ExportQueuedStaleAfter    = 30 * time.Minute
	ExportAttemptTimeout      = 30 * time.Minute
	ExportDownloadExpiration  = 24 * time.Hour
)

// InProgressは、エクスポートが結果へ向かう途中かどうかを返す。新しいエクスポートの開始を
// 拒否するかどうかも、エクスポート画面の表示も、この判定に従う。
//
// 途中の状態はどちらも、最近そう言えるものがある間だけ数える。startedはheartbeatで、queuedは
// ワーカーを待っている時間で判断する。プロセスごと停止させられて取り残されたエクスポートはその
// 状態のまま残り続けるため、実行中として扱うとそのスペースが二度とエクスポートできなくなる。
//
// こうして止まったと判断されるqueuedは、ジョブがこれから届くものではない。ワーカーへ届いた
// ジョブはExportQueuedStaleAfterより十分早くエクスポートをstartedへ進めるし、失敗にされた
// 後に届いたジョブは終端の状態を見つけて何もしない。
func (e *Export) InProgress(now time.Time) bool {
	switch e.Status {
	case ExportStatusQueued:
		return now.Sub(e.StatusChangedAt) < ExportQueuedStaleAfter
	case ExportStatusStarted:
		return e.HeartbeatAt != nil && now.Sub(*e.HeartbeatAt) < ExportHeartbeatStaleAfter
	default:
		return false
	}
}

// Downloadableはエクスポートのアーカイブをまだ渡せるかどうかを返す。画面はこれが成り立つ間
// だけダウンロードを表示し、ダウンロード自身も同じ判定をやり直すため、期限切れより前に描画された
// リンクがアーカイブより長生きすることはない。
//
// Rails版で成功したエクスポートはObjectKeyを持たず、そのアーカイブは本アプリケーションが渡せる
// ものではない。それらのエクスポートは状態を行へ移したマイグレーションからExportDownloadExpiration
// のうちに期限切れになるため、この条件が失わせるのは、その日のうちなら成功したはずのダウンロード
// だけである。
func (e *Export) Downloadable(now time.Time) bool {
	return e.Status == ExportStatusSucceeded &&
		e.ObjectKey != nil &&
		now.Sub(e.StatusChangedAt) < ExportDownloadExpiration
}
