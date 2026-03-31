package model_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	ve := model.NewValidationError()
	if got := ve.Error(); got != "validation failed" {
		t.Errorf("Error() = %q, want %q", got, "validation failed")
	}
}

func TestValidationError_AddGlobal(t *testing.T) {
	t.Parallel()

	ve := model.NewValidationError()
	ve.AddGlobal("エラー1")
	ve.AddGlobal("エラー2")

	if len(ve.Global) != 2 {
		t.Fatalf("len(Global) = %d, want 2", len(ve.Global))
	}
	if ve.Global[0] != "エラー1" {
		t.Errorf("Global[0] = %q, want %q", ve.Global[0], "エラー1")
	}
	if ve.Global[1] != "エラー2" {
		t.Errorf("Global[1] = %q, want %q", ve.Global[1], "エラー2")
	}
}

func TestValidationError_AddField(t *testing.T) {
	t.Parallel()

	ve := model.NewValidationError()
	ve.AddField("email", "必須です")
	ve.AddField("email", "形式が正しくありません")
	ve.AddField("password", "8文字以上必要です")

	emailErrors := ve.Fields["email"]
	if len(emailErrors) != 2 {
		t.Fatalf("len(Fields[email]) = %d, want 2", len(emailErrors))
	}
	if emailErrors[0] != "必須です" {
		t.Errorf("Fields[email][0] = %q, want %q", emailErrors[0], "必須です")
	}

	passwordErrors := ve.Fields["password"]
	if len(passwordErrors) != 1 {
		t.Fatalf("len(Fields[password]) = %d, want 1", len(passwordErrors))
	}
}

func TestValidationError_AddField_NilFields(t *testing.T) {
	t.Parallel()

	ve := &model.ValidationError{}
	ve.AddField("title", "必須です")

	if !ve.HasFieldError("title") {
		t.Error("HasFieldError(title) = false, want true")
	}
}

func TestValidationError_HasErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ve   *model.ValidationError
		want bool
	}{
		{
			name: "nilの場合はfalse",
			ve:   nil,
			want: false,
		},
		{
			name: "空の場合はfalse",
			ve:   model.NewValidationError(),
			want: false,
		},
		{
			name: "グローバルエラーがある場合はtrue",
			ve: func() *model.ValidationError {
				ve := model.NewValidationError()
				ve.AddGlobal("エラー")
				return ve
			}(),
			want: true,
		},
		{
			name: "フィールドエラーがある場合はtrue",
			ve: func() *model.ValidationError {
				ve := model.NewValidationError()
				ve.AddField("email", "必須です")
				return ve
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ve.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_HasFieldError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ve    *model.ValidationError
		field string
		want  bool
	}{
		{
			name:  "nilの場合はfalse",
			ve:    nil,
			field: "email",
			want:  false,
		},
		{
			name:  "エラーがないフィールドはfalse",
			ve:    model.NewValidationError(),
			field: "email",
			want:  false,
		},
		{
			name: "エラーがあるフィールドはtrue",
			ve: func() *model.ValidationError {
				ve := model.NewValidationError()
				ve.AddField("email", "必須です")
				return ve
			}(),
			field: "email",
			want:  true,
		},
		{
			name: "別のフィールドにエラーがある場合はfalse",
			ve: func() *model.ValidationError {
				ve := model.NewValidationError()
				ve.AddField("password", "必須です")
				return ve
			}(),
			field: "email",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ve.HasFieldError(tt.field); got != tt.want {
				t.Errorf("HasFieldError(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

func TestValidationError_GetFieldErrors(t *testing.T) {
	t.Parallel()

	t.Run("nilの場合はnilを返す", func(t *testing.T) {
		var ve *model.ValidationError
		if got := ve.GetFieldErrors("email"); got != nil {
			t.Errorf("GetFieldErrors() = %v, want nil", got)
		}
	})

	t.Run("エラーがないフィールドはnilを返す", func(t *testing.T) {
		ve := model.NewValidationError()
		if got := ve.GetFieldErrors("email"); got != nil {
			t.Errorf("GetFieldErrors() = %v, want nil", got)
		}
	})

	t.Run("エラーがあるフィールドはメッセージを返す", func(t *testing.T) {
		ve := model.NewValidationError()
		ve.AddField("email", "必須です")
		ve.AddField("email", "形式が正しくありません")

		got := ve.GetFieldErrors("email")
		if len(got) != 2 {
			t.Fatalf("len(GetFieldErrors()) = %d, want 2", len(got))
		}
		if got[0] != "必須です" {
			t.Errorf("GetFieldErrors()[0] = %q, want %q", got[0], "必須です")
		}
		if got[1] != "形式が正しくありません" {
			t.Errorf("GetFieldErrors()[1] = %q, want %q", got[1], "形式が正しくありません")
		}
	})
}

func TestValidationError_FieldErrors(t *testing.T) {
	t.Parallel()

	t.Run("nilの場合はnilを返す", func(t *testing.T) {
		var ve *model.ValidationError
		if got := ve.FieldErrors(); got != nil {
			t.Errorf("FieldErrors() = %v, want nil", got)
		}
	})

	t.Run("フィールドエラーを列挙可能な形式で取得する", func(t *testing.T) {
		ve := model.NewValidationError()
		ve.AddField("email", "必須です")
		ve.AddField("password", "8文字以上必要です")

		got := ve.FieldErrors()
		if len(got) != 2 {
			t.Fatalf("len(FieldErrors()) = %d, want 2", len(got))
		}

		found := map[string]bool{}
		for _, fe := range got {
			found[fe.Field+":"+fe.Message] = true
		}
		if !found["email:必須です"] {
			t.Error("FieldErrors() does not contain email error")
		}
		if !found["password:8文字以上必要です"] {
			t.Error("FieldErrors() does not contain password error")
		}
	})
}

func TestValidationError_ImplementsError(t *testing.T) {
	t.Parallel()

	// コンパイル時にerrorインターフェースを満たすことを確認
	var _ error = model.NewValidationError()
}

func TestAppError_Error(t *testing.T) {
	t.Parallel()

	ae := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: "リソースが見つかりません",
	}
	if got := ae.Error(); got != "リソースが見つかりません" {
		t.Errorf("Error() = %q, want %q", got, "リソースが見つかりません")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("DB接続エラー")
	ae := &model.AppError{
		Code:     model.AppErrCodeInternal,
		UserMsg:  "内部エラー",
		Internal: cause,
	}

	if !errors.Is(ae, cause) {
		t.Error("errors.Is(ae, cause) = false, want true")
	}
}

func TestAppError_LogString(t *testing.T) {
	t.Parallel()

	cause := errors.New("DB接続エラー")
	ae := &model.AppError{
		Code:     model.AppErrCodeInternal,
		UserMsg:  "内部エラー",
		Internal: cause,
		Metadata: map[string]string{"user_id": "123"},
	}

	got := ae.LogString()
	if got == "" {
		t.Error("LogString() should not be empty")
	}

	expected := fmt.Sprintf("Code: %d | Msg: %s | Cause: %v | Meta: %v",
		model.AppErrCodeInternal, "内部エラー", cause, ae.Metadata)
	if got != expected {
		t.Errorf("LogString() = %q, want %q", got, expected)
	}
}

func TestAppError_ImplementsError(t *testing.T) {
	t.Parallel()

	// コンパイル時にerrorインターフェースを満たすことを確認
	var _ error = &model.AppError{
		Code:    model.AppErrCodeForbidden,
		UserMsg: "権限がありません",
	}
}

func TestAppErrorCode_Values(t *testing.T) {
	t.Parallel()

	if model.AppErrCodeResourceNotFound == 0 {
		t.Error("AppErrCodeResourceNotFound should not be zero")
	}
	if model.AppErrCodeForbidden == model.AppErrCodeResourceNotFound {
		t.Error("AppErrCodeForbidden should differ from AppErrCodeResourceNotFound")
	}
	if model.AppErrCodeConflict == model.AppErrCodeForbidden {
		t.Error("AppErrCodeConflict should differ from AppErrCodeForbidden")
	}
	if model.AppErrCodeInternal == model.AppErrCodeConflict {
		t.Error("AppErrCodeInternal should differ from AppErrCodeConflict")
	}
}

func TestAsValidationError(t *testing.T) {
	t.Parallel()

	t.Run("ValidationErrorから取り出せる", func(t *testing.T) {
		ve := model.NewValidationError()
		ve.AddField("email", "必須です")

		got := model.AsValidationError(ve)
		if got == nil {
			t.Fatal("AsValidationError() = nil, want non-nil")
		}
		if !got.HasFieldError("email") {
			t.Error("取り出したValidationErrorにemailエラーがない")
		}
	})

	t.Run("ラップされたValidationErrorから取り出せる", func(t *testing.T) {
		ve := model.NewValidationError()
		ve.AddGlobal("エラー")
		wrapped := fmt.Errorf("wrapped: %w", ve)

		got := model.AsValidationError(wrapped)
		if got == nil {
			t.Fatal("AsValidationError() = nil, want non-nil")
		}
		if len(got.Global) != 1 {
			t.Errorf("len(Global) = %d, want 1", len(got.Global))
		}
	})

	t.Run("AppErrorからは取り出せない", func(t *testing.T) {
		ae := &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: "forbidden"}
		got := model.AsValidationError(ae)
		if got != nil {
			t.Errorf("AsValidationError() = %v, want nil", got)
		}
	})

	t.Run("通常のerrorからは取り出せない", func(t *testing.T) {
		err := errors.New("通常のエラー")
		got := model.AsValidationError(err)
		if got != nil {
			t.Errorf("AsValidationError() = %v, want nil", got)
		}
	})

	t.Run("nilからは取り出せない", func(t *testing.T) {
		got := model.AsValidationError(nil)
		if got != nil {
			t.Errorf("AsValidationError(nil) = %v, want nil", got)
		}
	})
}

func TestAsAppError(t *testing.T) {
	t.Parallel()

	t.Run("AppErrorから取り出せる", func(t *testing.T) {
		ae := &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: "見つかりません",
		}

		got := model.AsAppError(ae)
		if got == nil {
			t.Fatal("AsAppError() = nil, want non-nil")
		}
		if got.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("Code = %d, want %d", got.Code, model.AppErrCodeResourceNotFound)
		}
	})

	t.Run("ラップされたAppErrorから取り出せる", func(t *testing.T) {
		ae := &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: "権限がありません",
		}
		wrapped := fmt.Errorf("wrapped: %w", ae)

		got := model.AsAppError(wrapped)
		if got == nil {
			t.Fatal("AsAppError() = nil, want non-nil")
		}
		if got.Code != model.AppErrCodeForbidden {
			t.Errorf("Code = %d, want %d", got.Code, model.AppErrCodeForbidden)
		}
	})

	t.Run("ValidationErrorからは取り出せない", func(t *testing.T) {
		ve := model.NewValidationError()
		got := model.AsAppError(ve)
		if got != nil {
			t.Errorf("AsAppError() = %v, want nil", got)
		}
	})

	t.Run("通常のerrorからは取り出せない", func(t *testing.T) {
		err := errors.New("通常のエラー")
		got := model.AsAppError(err)
		if got != nil {
			t.Errorf("AsAppError() = %v, want nil", got)
		}
	})

	t.Run("nilからは取り出せない", func(t *testing.T) {
		got := model.AsAppError(nil)
		if got != nil {
			t.Errorf("AsAppError(nil) = %v, want nil", got)
		}
	})
}
