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
