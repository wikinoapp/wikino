package model

import (
	"time"
)

// ExportStatus is where an export stands.
//
// [Ja] ExportStatus はエクスポートがどの状態にあるかを表す。
type ExportStatus int32

const (
	// ExportStatusQueued is the state of an export whose job is waiting to run.
	//
	// [Ja] ExportStatusQueued はジョブの実行待ちのエクスポートの状態
	ExportStatusQueued ExportStatus = 0

	// ExportStatusStarted is the state of an export a worker is generating.
	//
	// [Ja] ExportStatusStarted はワーカーが生成中のエクスポートの状態
	ExportStatusStarted ExportStatus = 1

	// ExportStatusSucceeded is the state of an export whose ZIP is in the bucket.
	//
	// [Ja] ExportStatusSucceeded は ZIP がバケットに置かれたエクスポートの状態
	ExportStatusSucceeded ExportStatus = 2

	// ExportStatusFailed is the state of an export whose last attempt gave up.
	//
	// [Ja] ExportStatusFailed は最後の試行が諦めたエクスポートの状態
	ExportStatusFailed ExportStatus = 3
)

// Export is the domain model of a space export: one attempt to write the whole space out as a
// ZIP of Markdown files and attachments.
//
// HeartbeatAt is what the worker updates while it runs. An export left in
// ExportStatusStarted by a process that was killed keeps that status forever, so a stale
// heartbeat is what tells a slow export from a stopped one.
//
// ObjectKey points at the ZIP in the object storage and is set when the export succeeds. It
// stays empty for the exports that Rails wrote, whose ZIP lives under an ActiveStorage blob
// key instead.
//
// [Ja] Export はスペースのエクスポートのドメインモデル。スペース全体を Markdown ファイルと添付
// ファイルの ZIP として書き出す 1 回の試行を表す。
//
// HeartbeatAt はワーカーが処理中に更新する値。プロセスごと停止させられたエクスポートは
// ExportStatusStarted のまま残り続けるため、heartbeat が古いことが、遅いエクスポートと止まった
// エクスポートを見分ける手がかりになる。
//
// ObjectKey はオブジェクトストレージ上の ZIP を指し、エクスポートの成功時に設定される。Rails が
// 書いたエクスポートでは空のままで、それらの ZIP は代わりに ActiveStorage の blob のキーの下にある。
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

// The timing of an export is decided by these five values together, which is why they sit in one
// place rather than beside the code that reads each of them.
//
// ExportAttemptTimeout bounds one attempt. River requires its RescueStuckJobsAfter to be longer
// than the timeout it gives a job, so the worker client sets both from here.
//
// ExportHeartbeatStaleAfter is when a started export stops counting as running. It is well above
// ExportHeartbeatInterval, so that a worker that is merely slow to reach its next beat is not
// taken for a stopped one, and well below ExportAttemptTimeout, so that a stopped export can be
// replaced without waiting out an attempt that will never finish.
//
// ExportQueuedStaleAfter is when a queued export stops counting as running. It answers the case a
// heartbeat cannot: an export whose job never reached a worker has nothing to beat with, and
// without a bound of its own it would keep its space from ever exporting again. It sits above
// ExportHeartbeatStaleAfter because waiting for a worker is a longer wait than waiting for a beat
// from one that is already running.
//
// ExportDownloadExpiration is how long the link in the completion email keeps working. The
// wording of that email is built from this value, so the two cannot drift apart.
//
// [Ja] エクスポートの時間に関する 5 つの値は互いに関係して決まるため、それぞれを読むコードの
// そばではなく 1 か所に置く。
//
// ExportAttemptTimeout は 1 回の試行の上限。River はジョブに与えるタイムアウトより
// RescueStuckJobsAfter を長くすることを要求するため、ワーカークライアントは両方をここから決める。
//
// ExportHeartbeatStaleAfter は、started のエクスポートが実行中と見なされなくなる時間。
// ExportHeartbeatInterval より十分大きいため、次の heartbeat が遅れているだけのワーカーを
// 止まったものと見なさない。ExportAttemptTimeout より十分小さいため、止まったエクスポートを
// 置き換えるのに、終わることのない試行の上限を待たずに済む。
//
// ExportQueuedStaleAfter は、queued のエクスポートが実行中と見なされなくなる時間。heartbeat では
// 答えられない場合のためにある。ワーカーへ届かなかったジョブのエクスポートは打つべき heartbeat を
// 持たず、それ自身の上限が無ければ、そのスペースが二度とエクスポートできなくなる。
// ExportHeartbeatStaleAfter より大きいのは、ワーカーを待つ時間が、既に動いているワーカーからの
// 鼓動を待つ時間より長いためである。
//
// ExportDownloadExpiration は完了メールのリンクが使える期間。メールの文面はこの値から組み立てる
// ので、両者がずれることはない。
const (
	ExportHeartbeatInterval   = 30 * time.Second
	ExportHeartbeatStaleAfter = 5 * time.Minute
	ExportQueuedStaleAfter    = 30 * time.Minute
	ExportAttemptTimeout      = 30 * time.Minute
	ExportDownloadExpiration  = 24 * time.Hour
)

// InProgress reports whether the export is still on its way to a result, which is what both the
// refusal to start another export and the export screen go by.
//
// Both statuses that are on their way count only while something says so recently. A started
// export goes by its heartbeat, and a queued one by how long it has been waiting for a worker:
// an export that was left behind by a process that was killed keeps its status forever, and
// treating it as running would block the space from ever exporting again.
//
// A queued export declared stopped this way is not one whose job is still coming. A job that
// reaches a worker moves the export to started well inside ExportQueuedStaleAfter, and one that
// arrives after the export was failed finds a terminal status and does nothing.
//
// [Ja] InProgress は、エクスポートが結果へ向かう途中かどうかを返す。新しいエクスポートの開始を
// 拒否するかどうかも、エクスポート画面の表示も、この判定に従う。
//
// 途中の状態はどちらも、最近そう言えるものがある間だけ数える。started は heartbeat で、queued は
// ワーカーを待っている時間で判断する。プロセスごと停止させられて取り残されたエクスポートはその
// 状態のまま残り続けるため、実行中として扱うとそのスペースが二度とエクスポートできなくなる。
//
// こうして止まったと判断される queued は、ジョブがこれから届くものではない。ワーカーへ届いた
// ジョブは ExportQueuedStaleAfter より十分早くエクスポートを started へ進めるし、失敗にされた
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

// Downloadable reports whether the archive of the export can still be handed out. The screen shows
// the download only while this holds, and the download itself checks it again, so a link that was
// rendered before the archive expired does not outlive it.
//
// An export that succeeded on the Rails version carries no ObjectKey, and its archive is therefore
// not one this application can hand out. Those exports expire within ExportDownloadExpiration of
// the migration that moved the state onto the row, so what the condition costs is a download that
// would have worked for the rest of that day.
//
// [Ja] Downloadable はエクスポートのアーカイブをまだ渡せるかどうかを返す。画面はこれが成り立つ間
// だけダウンロードを表示し、ダウンロード自身も同じ判定をやり直すため、期限切れより前に描画された
// リンクがアーカイブより長生きすることはない。
//
// Rails 版で成功したエクスポートは ObjectKey を持たず、そのアーカイブは本アプリケーションが渡せる
// ものではない。それらのエクスポートは状態を行へ移したマイグレーションから ExportDownloadExpiration
// のうちに期限切れになるため、この条件が失わせるのは、その日のうちなら成功したはずのダウンロード
// だけである。
func (e *Export) Downloadable(now time.Time) bool {
	return e.Status == ExportStatusSucceeded &&
		e.ObjectKey != nil &&
		now.Sub(e.StatusChangedAt) < ExportDownloadExpiration
}
