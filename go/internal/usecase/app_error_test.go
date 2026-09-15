package usecase

import (
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// assertAppErrCodeはerrが *model.AppErrorかつ指定したCodeを持つかを検証する。
func assertAppErrCode(t *testing.T, err error, code model.AppErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("*model.AppErrorを期待したが、nilだった")
	}
	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("*model.AppErrorを期待したが、%Tだった: %v", err, err)
	}
	if ae.Code != code {
		t.Errorf("AppError.Code = %d、期待値 = %d", ae.Code, code)
	}
}
