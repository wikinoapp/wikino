package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// ExportStateはエクスポート画面がエクスポートについて示す状態。3つの値は画面が描き分ける
// 分岐であり、エクスポートが取りうる状態より少ない。停止したワーカーに取り残されたエクスポートは
// queuedやstartedのまま残り続けるため、画面はそれを実行中として見せるのではなく失敗と呼ぶ
// 必要がある。
type ExportState string

const (
	// ExportStateProcessingは結果へ向かう途中のエクスポート
	ExportStateProcessing ExportState = "processing"

	// ExportStateSucceededはアーカイブが書き出されたエクスポート
	ExportStateSucceeded ExportState = "succeeded"

	// ExportStateFailedは諦めたエクスポート、または停止したワーカーに取り残されたエクスポート
	ExportStateFailed ExportState = "failed"
)

// Exportはエクスポート画面で表示するエクスポート。
type Export struct {
	ID           string
	State        ExportState
	Downloadable bool
}

// NewExportはモデルからExportを生成する。どの状態にあるかはnowの時点で読む。
func NewExport(export *model.Export, now time.Time) Export {
	return Export{
		ID:           export.ID.String(),
		State:        exportState(export, now),
		Downloadable: export.Downloadable(now),
	}
}

// exportStateはエクスポートの状態を、画面が描き分ける分岐へ対応付ける。実行中でも成功でも
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
