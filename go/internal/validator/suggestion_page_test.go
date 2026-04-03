package validator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestSuggestionPageUpdateValidator_DraftPageが存在しリンクされている場合は成功する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spv-ok@example.com").
		WithAtname("spvok").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spv-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	suggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(suggestionPageID).
		WithBody("提案の本文").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	v := validator.NewSuggestionPageUpdateValidator(draftPageRepo)

	dp, err := v.Validate(context.Background(), validator.SuggestionPageUpdateValidatorInput{
		SuggestionPageID: suggestionPageID,
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		SpaceID:          spaceID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dp == nil {
		t.Fatal("DraftPage should not be nil")
	}
	if dp.Body != "提案の本文" {
		t.Errorf("DraftPage.Body = %q, want %q", dp.Body, "提案の本文")
	}
}

func TestSuggestionPageUpdateValidator_DraftPageが存在しない場合はエラー(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spv-nodp@example.com").
		WithAtname("spvnodp").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spv-nodp-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	v := validator.NewSuggestionPageUpdateValidator(draftPageRepo)

	dp, err := v.Validate(context.Background(), validator.SuggestionPageUpdateValidatorInput{
		SuggestionPageID: "nonexistent-sp-id",
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		SpaceID:          spaceID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, validator.ErrDraftPageNotFound) {
		t.Errorf("expected ErrDraftPageNotFound, got %v", err)
	}
	if dp != nil {
		t.Error("DraftPage should be nil on error")
	}
}

func TestSuggestionPageUpdateValidator_DraftPageが別のSuggestionPageにリンクされている場合はエラー(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spv-wronglink@example.com").
		WithAtname("spvwronglink").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spv-wronglnk-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	otherSuggestionPageID := testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	// DraftPageは otherSuggestionPageID にリンクされている
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithSuggestionPageID(otherSuggestionPageID).
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	v := validator.NewSuggestionPageUpdateValidator(draftPageRepo)

	// 別のSuggestionPageIDを指定
	dp, err := v.Validate(context.Background(), validator.SuggestionPageUpdateValidatorInput{
		SuggestionPageID: "different-sp-id",
		PageID:           pageID,
		SpaceMemberID:    spaceMemberID,
		SpaceID:          spaceID,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, validator.ErrDraftPageNotLinked) {
		t.Errorf("expected ErrDraftPageNotLinked, got %v", err)
	}
	if dp != nil {
		t.Error("DraftPage should be nil on error")
	}
}

func TestSuggestionPageCreateValidator_正常系_有効な下書きページで成功する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spcv-ok@example.com").
		WithAtname("spcvok").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spcv-ok-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()

	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("本文").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	v := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	draftPages, err := v.Validate(context.Background(), validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  []model.DraftPageID{draftPageID},
		SpaceMemberID: spaceMemberID,
		TopicID:       topicID,
		SpaceID:       spaceID,
		SuggestionID:  suggestionID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(draftPages) != 1 {
		t.Fatalf("draftPages count = %d, want 1", len(draftPages))
	}
	if draftPages[0].Body != "本文" {
		t.Errorf("DraftPage.Body = %q, want %q", draftPages[0].Body, "本文")
	}
}

func TestSuggestionPageCreateValidator_異常系_下書きページ未選択(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	draftPageRepo := repository.NewDraftPageRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	v := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	_, err := v.Validate(context.Background(), validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  []model.DraftPageID{},
		SpaceMemberID: "dummy-sm",
		TopicID:       "dummy-topic",
		SpaceID:       "dummy-space",
		SuggestionID:  "dummy-sug",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("draft_page_ids") {
		t.Error("expected draft_page_ids field error")
	}
}

func TestSuggestionPageCreateValidator_異常系_下書きページが存在しない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spcv-nodp@example.com").
		WithAtname("spcvnodp").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spcv-nodp-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	v := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	_, err := v.Validate(context.Background(), validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  []model.DraftPageID{"00000000-0000-0000-0000-000000000099"},
		SpaceMemberID: spaceMemberID,
		TopicID:       topicID,
		SpaceID:       spaceID,
		SuggestionID:  suggestionID,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("draft_page_ids") {
		t.Error("expected draft_page_ids field error")
	}
}

func TestSuggestionPageCreateValidator_異常系_別メンバーの下書きページ(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spcv-othermem@example.com").
		WithAtname("spcvothermem").
		Build()
	otherUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("spcv-othermem2@example.com").
		WithAtname("spcvothermem2").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spcv-othermem-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	otherSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(otherUserID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()

	// 別メンバーが作成した下書きページ
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(otherSpaceMemberID).
		WithTopicID(topicID).
		WithBody("本文").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	v := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	_, err := v.Validate(context.Background(), validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  []model.DraftPageID{draftPageID},
		SpaceMemberID: spaceMemberID,
		TopicID:       topicID,
		SpaceID:       spaceID,
		SuggestionID:  suggestionID,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("draft_page_ids") {
		t.Error("expected draft_page_ids field error")
	}
}

func TestSuggestionPageCreateValidator_異常系_既に編集提案に含まれているページ(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("spcv-dup@example.com").
		WithAtname("spcvdup").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("spcv-dup-sp").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithVisibility(0).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		Build()
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	// 既に編集提案に含まれているSuggestionPage
	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	// 同じページの下書き
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithBody("本文").
		Build()

	draftPageRepo := repository.NewDraftPageRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	v := validator.NewSuggestionPageCreateValidator(draftPageRepo, suggestionPageRepo)

	_, err := v.Validate(context.Background(), validator.SuggestionPageCreateValidatorInput{
		DraftPageIDs:  []model.DraftPageID{draftPageID},
		SpaceMemberID: spaceMemberID,
		TopicID:       topicID,
		SpaceID:       spaceID,
		SuggestionID:  suggestionID,
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !ve.HasFieldError("draft_page_ids") {
		t.Error("expected draft_page_ids field error")
	}
}
