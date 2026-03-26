package validator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSuggestionCreateValidator_FormatValidation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	tests := []struct {
		name          string
		title         string
		draftPageIDs  []model.DraftPageID
		wantError     bool
		expectedField string
	}{
		{
			name:          "タイトルが空の場合はエラー",
			title:         "",
			draftPageIDs:  []model.DraftPageID{"draft-1"},
			wantError:     true,
			expectedField: "title",
		},
		{
			name:          "タイトルが200文字を超える場合はエラー",
			title:         strings.Repeat("あ", 201),
			draftPageIDs:  []model.DraftPageID{"draft-1"},
			wantError:     true,
			expectedField: "title",
		},
		{
			name:          "下書きページが未選択の場合はエラー",
			title:         "テスト提案",
			draftPageIDs:  []model.DraftPageID{},
			wantError:     true,
			expectedField: "draft_page_ids",
		},
	}

	// 形式バリデーションのみテストするためnilのdraftPageRepoを使用
	v := validator.NewSuggestionCreateValidator(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.Validate(ctx, validator.SuggestionCreateValidatorInput{
				Title:         tt.title,
				DraftPageIDs:  tt.draftPageIDs,
				SpaceMemberID: "test-member-id",
				TopicID:       "test-topic-id",
				SpaceID:       "test-space-id",
			})

			if !result.FormErrors.HasErrors() {
				t.Error("expected errors but got none")
			}
			if !result.FormErrors.HasFieldError(tt.expectedField) {
				t.Errorf("expected %s field error but got none", tt.expectedField)
			}
		})
	}
}

func TestSuggestionCreateValidator_StateValidation(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("suggestion-validator-test").
		Build()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("suggestion-validator@example.com").
		WithAtname("suggestionvalidator").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Draft Title").
		WithBody("Draft body").
		Build()

	// 別のメンバーの下書きを作成
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("other-validator@example.com").
		WithAtname("othervalidator").
		Build()
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(otherUserID).
		Build()
	otherPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Other Page").
		Build()
	otherDraftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(otherPageID).
		WithSpaceMemberID(otherMemberID).
		WithTopicID(topicID).
		WithTitle("Other Draft").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	v := validator.NewSuggestionCreateValidator(draftPageRepo)

	t.Run("正常系: 有効な入力で下書きページのバリデーションが通る", func(t *testing.T) {
		result := v.Validate(ctx, validator.SuggestionCreateValidatorInput{
			Title:         "テスト編集提案",
			DraftPageIDs:  []model.DraftPageID{draftPageID},
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
			SpaceID:       spaceID,
		})

		if result.FormErrors.HasErrors() {
			t.Errorf("unexpected errors: %v", result.FormErrors)
		}
		if len(result.DraftPages) != 1 {
			t.Errorf("DraftPages count = %d, want 1", len(result.DraftPages))
		}
	})

	t.Run("存在しない下書きページIDの場合はエラー", func(t *testing.T) {
		// 有効なUUID形式だが存在しないID
		result := v.Validate(ctx, validator.SuggestionCreateValidatorInput{
			Title:         "テスト編集提案",
			DraftPageIDs:  []model.DraftPageID{"00000000-0000-0000-0000-000000000000"},
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
			SpaceID:       spaceID,
		})

		if !result.FormErrors.HasErrors() {
			t.Error("expected error but got none")
		}
		if !result.FormErrors.HasFieldError("draft_page_ids") {
			t.Error("expected draft_page_ids field error")
		}
	})

	t.Run("他のメンバーの下書きページの場合はエラー", func(t *testing.T) {
		result := v.Validate(ctx, validator.SuggestionCreateValidatorInput{
			Title:         "テスト編集提案",
			DraftPageIDs:  []model.DraftPageID{otherDraftPageID},
			SpaceMemberID: spaceMemberID,
			TopicID:       topicID,
			SpaceID:       spaceID,
		})

		if !result.FormErrors.HasErrors() {
			t.Error("expected error but got none")
		}
		if !result.FormErrors.HasFieldError("draft_page_ids") {
			t.Error("expected draft_page_ids field error")
		}
	})
}

func TestSuggestionUpdateValidator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	v := validator.NewSuggestionUpdateValidator()

	tests := []struct {
		name          string
		title         string
		body          string
		wantError     bool
		expectedField string
	}{
		{
			name:      "正常系: 有効な入力",
			title:     "テスト提案",
			body:      "テスト本文",
			wantError: false,
		},
		{
			name:      "正常系: 本文が空でもOK",
			title:     "テスト提案",
			body:      "",
			wantError: false,
		},
		{
			name:          "タイトルが空の場合はエラー",
			title:         "",
			body:          "テスト本文",
			wantError:     true,
			expectedField: "title",
		},
		{
			name:          "タイトルが200文字を超える場合はエラー",
			title:         strings.Repeat("あ", 201),
			body:          "",
			wantError:     true,
			expectedField: "title",
		},
		{
			name:      "タイトルがちょうど200文字の場合はOK",
			title:     strings.Repeat("あ", 200),
			body:      "",
			wantError: false,
		},
		{
			name:          "本文が10000文字を超える場合はエラー",
			title:         "テスト提案",
			body:          strings.Repeat("あ", 10001),
			wantError:     true,
			expectedField: "body",
		},
		{
			name:      "本文がちょうど10000文字の場合はOK",
			title:     "テスト提案",
			body:      strings.Repeat("あ", 10000),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.Validate(ctx, validator.SuggestionUpdateValidatorInput{
				Title: tt.title,
				Body:  tt.body,
			})

			if tt.wantError {
				if !result.FormErrors.HasErrors() {
					t.Error("expected errors but got none")
				}
				if tt.expectedField != "" && !result.FormErrors.HasFieldError(tt.expectedField) {
					t.Errorf("expected %s field error but got none", tt.expectedField)
				}
			} else {
				if result.FormErrors.HasErrors() {
					t.Errorf("unexpected errors: %v", result.FormErrors)
				}
			}
		})
	}
}
