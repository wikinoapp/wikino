package model

import (
	"time"
)

// ExportStatus is where an export stands. The values match the kind column of the Rails-era
// export_statuses rows, which is why the state could move onto the exports row without
// translating them.
//
// [Ja] ExportStatus はエクスポートがどの状態にあるかを表す。値は Rails 時代の export_statuses の
// kind 列と一致しており、そのおかげで状態を exports の行へ読み替え無しで移せた。
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
