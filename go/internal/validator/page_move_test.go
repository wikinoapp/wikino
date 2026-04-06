package validator_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestPageMoveCreateValidator_EmptyDestTopic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	v := validator.NewPageMoveCreateValidator(nil, nil, nil, nil)
	_, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError but got nil")
	}
	if !ve.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error")
	}
}

func TestPageMoveCreateValidator_InvalidDestTopicNumber(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	v := validator.NewPageMoveCreateValidator(nil, nil, nil, nil)
	_, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "abc",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError but got nil")
	}
	if !ve.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error")
	}
}

func TestPageMoveCreateValidator_SameTopic(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("test-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0). // owner
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)

	v := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	_, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "1",
		PageID:          pageID,
		PageTitle:       "Test Page",
		CurrentTopicID:  topicID,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError but got nil")
	}
	if !ve.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error for same topic")
	}
}

func TestPageMoveCreateValidator_TitleExistsInDestTopic(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("test-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0). // owner
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Topic 2").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("Duplicate Title").
		Build()
	// 移動先トピックに同名のページを作成
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithNumber(2).
		WithTitle("Duplicate Title").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)

	v := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	_, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "2",
		PageID:          pageID,
		PageTitle:       "Duplicate Title",
		CurrentTopicID:  topicID1,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError but got nil")
	}
	if !ve.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error for title exists")
	}
}

func TestPageMoveCreateValidator_OpenSuggestionExists(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("test-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0). // owner
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Topic 2").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	// ページリビジョンとオープンな編集提案を作成
	pageRevisionID := testutil.NewPageRevisionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithPageID(pageID).
		Build()
	suggestionID := testutil.NewSuggestionBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithCreatedSpaceMemberID(spaceMemberID).
		WithStatus(model.SuggestionStatusOpen).
		WithTitle("オープン提案").
		Build()
	testutil.NewSuggestionPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSuggestionID(suggestionID).
		WithPageID(pageID).
		WithPageRevisionID(pageRevisionID).
		Build()

	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)

	v := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	_, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "2",
		PageID:          pageID,
		PageTitle:       "Test Page",
		CurrentTopicID:  topicID1,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError but got nil")
	}
	if !ve.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error for open suggestion exists")
	}
}

func TestPageMoveCreateValidator_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	// テストデータを作成
	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("test-space").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(0). // owner
		Build()
	topicID1 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Topic 1").
		Build()
	topicID2 := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Topic 2").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithSpaceMemberID(spaceMemberID).
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID2).
		WithSpaceMemberID(spaceMemberID).
		Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID1).
		WithNumber(1).
		WithTitle("Test Page").
		Build()

	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)

	v := validator.NewPageMoveCreateValidator(pageRepo, topicRepo, topicMemberRepo, suggestionPageRepo)
	destTopic, err := v.Validate(ctx, validator.PageMoveCreateValidatorInput{
		DestTopicNumber: "2",
		PageID:          pageID,
		PageTitle:       "Test Page",
		CurrentTopicID:  topicID1,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destTopic == nil {
		t.Error("expected dest topic but got nil")
	}
}
