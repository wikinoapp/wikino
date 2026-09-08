package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ExportState is what the export screen says about an export. The three values are the branches
// the screen renders, which is fewer than the statuses an export can hold: an export left behind
// by a worker that stopped keeps queued or started forever, and the screen has to call that a
// failure rather than show it as still running.
//
// [Ja] ExportState はエクスポート画面がエクスポートについて示す状態。3 つの値は画面が描き分ける
// 分岐であり、エクスポートが取りうる状態より少ない。停止したワーカーに取り残されたエクスポートは
// queued や started のまま残り続けるため、画面はそれを実行中として見せるのではなく失敗と呼ぶ
// 必要がある。
type ExportState string

const (
	// ExportStateProcessing is an export that is still on its way to a result.
	//
	// [Ja] ExportStateProcessing は結果へ向かう途中のエクスポート
	ExportStateProcessing ExportState = "processing"

	// ExportStateSucceeded is an export whose archive was written.
	//
	// [Ja] ExportStateSucceeded はアーカイブが書き出されたエクスポート
	ExportStateSucceeded ExportState = "succeeded"

	// ExportStateFailed is an export that gave up, or was left behind by a worker that stopped.
	//
	// [Ja] ExportStateFailed は諦めたエクスポート、または停止したワーカーに取り残されたエクスポート
	ExportStateFailed ExportState = "failed"
)

// Export is the export shown on the export screen.
//
// [Ja] Export はエクスポート画面で表示するエクスポート。
type Export struct {
	ID           string
	State        ExportState
	Downloadable bool
}

// NewExport builds an Export from the model, reading the state it is in as of now.
//
// [Ja] NewExport はモデルから Export を生成する。どの状態にあるかは now の時点で読む。
func NewExport(export *model.Export, now time.Time) Export {
	return Export{
		ID:           export.ID.String(),
		State:        exportState(export, now),
		Downloadable: export.Downloadable(now),
	}
}

// exportState maps the status of an export onto the branch the screen renders. Everything that is
// neither running nor succeeded is a failure to the reader, whether the export recorded one or was
// simply abandoned.
//
// [Ja] exportState はエクスポートの状態を、画面が描き分ける分岐へ対応付ける。実行中でも成功でも
// ないものは、失敗を記録したのか単に見捨てられたのかによらず、読み手にとっては失敗である。
func exportState(export *model.Export, now time.Time) ExportState {
	switch {
	case export.InProgress(now):
		return ExportStateProcessing
	case export.Status == model.ExportStatusSucceeded:
		return ExportStateSucceeded
	default:
		return ExportStateFailed
	}
}
