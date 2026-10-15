package apiclient

import (
	"strings"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
)

// LookupOperationは、operationIdがidのoperationとその分類を返す。
// 確認プロトコルの対象 (ConfirmationRequired) と `--dry-run` を付けるコマンド (ReadOnlyでないもの) を、
// コマンドに一覧を持たずにOpenAPI記述から決めるために使う。
func LookupOperation(id string) (apigen.Operation, bool) {
	for _, op := range apigen.Operations {
		if op.ID == id {
			return op, true
		}
	}

	return apigen.Operation{}, false
}

// MatchOperationは、methodとAPIの基底からのpath (エスケープした形) に当たるoperationを返す。
// パスのテンプレートの `{...}` は、空でない1つのセグメントに当たる。
func MatchOperation(method, path string) (apigen.Operation, bool) {
	segments := strings.Split(path, "/")

	for _, op := range apigen.Operations {
		if op.Method != strings.ToUpper(method) {
			continue
		}
		if matchPath(strings.Split(op.Path, "/"), segments) {
			return op, true
		}
	}

	return apigen.Operation{}, false
}

func matchPath(template, segments []string) bool {
	if len(template) != len(segments) {
		return false
	}

	for i, t := range template {
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			if segments[i] == "" {
				return false
			}
			continue
		}
		if t != segments[i] {
			return false
		}
	}

	return true
}
