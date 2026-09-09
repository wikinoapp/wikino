package usecase

import (
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// assertAppErrCode asserts that err is a *model.AppError carrying the given code.
//
// [Ja] assertAppErrCode は err が *model.AppError かつ指定した Code を持つかを検証する。
func assertAppErrCode(t *testing.T, err error, code model.AppErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected *model.AppError, got nil")
	}
	var ae *model.AppError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *model.AppError, got %T: %v", err, err)
	}
	if ae.Code != code {
		t.Errorf("AppError.Code = %d, want %d", ae.Code, code)
	}
}
