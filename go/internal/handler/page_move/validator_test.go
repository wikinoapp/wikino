package page_move_test

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/handler/page_move"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateValidator_EmptyDestTopic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	validator := page_move.NewCreateValidator(nil, nil, nil)
	result := validator.Validate(ctx, page_move.CreateValidatorInput{
		DestTopicNumber: "",
	})

	if !result.FormErrors.HasErrors() {
		t.Error("expected errors but got none")
	}
	if !result.FormErrors.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error")
	}
}

func TestCreateValidator_InvalidDestTopicNumber(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	validator := page_move.NewCreateValidator(nil, nil, nil)
	result := validator.Validate(ctx, page_move.CreateValidatorInput{
		DestTopicNumber: "abc",
	})

	if !result.FormErrors.HasErrors() {
		t.Error("expected errors but got none")
	}
	if !result.FormErrors.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error")
	}
}

func TestCreateValidator_SameTopic(t *testing.T) {
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

	validator := page_move.NewCreateValidator(pageRepo, topicRepo, topicMemberRepo)
	result := validator.Validate(ctx, page_move.CreateValidatorInput{
		DestTopicNumber: "1",
		PageID:          pageID,
		PageTitle:       "Test Page",
		CurrentTopicID:  topicID,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	if !result.FormErrors.HasErrors() {
		t.Error("expected errors but got none")
	}
	if !result.FormErrors.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error for same topic")
	}
}

func TestCreateValidator_TitleExistsInDestTopic(t *testing.T) {
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

	validator := page_move.NewCreateValidator(pageRepo, topicRepo, topicMemberRepo)
	result := validator.Validate(ctx, page_move.CreateValidatorInput{
		DestTopicNumber: "2",
		PageID:          pageID,
		PageTitle:       "Duplicate Title",
		CurrentTopicID:  topicID1,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	if !result.FormErrors.HasErrors() {
		t.Error("expected errors but got none")
	}
	if !result.FormErrors.HasFieldError("dest_topic") {
		t.Error("expected dest_topic field error for title exists")
	}
}

func TestCreateValidator_Success(t *testing.T) {
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

	validator := page_move.NewCreateValidator(pageRepo, topicRepo, topicMemberRepo)
	result := validator.Validate(ctx, page_move.CreateValidatorInput{
		DestTopicNumber: "2",
		PageID:          pageID,
		PageTitle:       "Test Page",
		CurrentTopicID:  topicID1,
		SpaceID:         spaceID,
		SpaceMember:     &model.SpaceMember{ID: spaceMemberID, Role: model.SpaceMemberRoleOwner, SpaceID: spaceID, Active: true},
	})

	if result.FormErrors.HasErrors() {
		t.Errorf("unexpected errors: %v", result.FormErrors)
	}
	if result.DestTopic == nil {
		t.Error("expected dest topic but got nil")
	}
}
