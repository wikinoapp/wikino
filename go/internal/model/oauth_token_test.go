package model_test

import (
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestAsOAuthTokenError(t *testing.T) {
	t.Parallel()

	err := &model.OAuthTokenError{Code: model.OAuthTokenErrorInvalidGrant}
	if got := model.AsOAuthTokenError(fmt.Errorf("包む: %w", err)); got != err {
		t.Errorf("AsOAuthTokenError() = %v、期待値 = %v", got, err)
	}
	if got := model.AsOAuthTokenError(fmt.Errorf("別のエラー")); got != nil {
		t.Errorf("AsOAuthTokenError() = %v、期待値 = nil", got)
	}
}
