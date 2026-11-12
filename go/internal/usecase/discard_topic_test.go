package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// rejectingJobInserterはどのジョブも受け付けない。論理削除したのに物理削除のジョブを投入できない
// 場合を再現する。
type rejectingJobInserter struct{}

func (rejectingJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return nil, errors.New("キューに接続できない")
}

// newDiscardTopicUsecaseはフィクスチャのトランザクションと、渡したジョブ投入器を使うUseCaseを
// 組み立てる。
func newDiscardTopicUsecase(f topicSettingsGeneralFixture, inserter dispatcher.JobInserter) *DiscardTopicUsecase {
	return NewDiscardTopicUsecase(
		repository.NewSpaceRepository(f.queries),
		repository.NewSpaceMemberRepository(f.queries),
		repository.NewTopicRepository(f.queries),
		repository.NewTopicMemberRepository(f.queries),
		dispatcher.NewDispatcher(inserter),
		validator.NewTopicDeletionCreateValidator(),
	)
}

// isTopicActiveはトピックが論理削除されていないかを返す。
func isTopicActive(t *testing.T, f topicSettingsGeneralFixture) bool {
	t.Helper()

	topic, err := repository.NewTopicRepository(f.queries).FindBySpaceAndID(t.Context(), f.spaceID, f.topicID)
	if err != nil {
		t.Fatalf("トピックの取得に失敗: %v", err)
	}
	return topic != nil
}

func TestDiscardTopicUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "discard-success", model.SpaceRoleAdmin, "")
	inserter := &mockJobInserter{}
	uc := newDiscardTopicUsecase(f, inserter)

	output, err := uc.Execute(ctx, DiscardTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		TopicName:       "日報 discard-success",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if output.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
	}
	if isTopicActive(t, f) {
		t.Error("トピックが論理削除されていない")
	}
	args, ok := inserter.args.(dispatcher.DeleteTopicArgs)
	if !ok {
		t.Fatalf("投入したジョブの引数 = %T、DeleteTopicArgsを期待", inserter.args)
	}
	if args.SpaceID != string(f.spaceID) || args.TopicID != string(f.topicID) {
		t.Errorf("ジョブの引数 = %+v、期待値 = スペース%s・トピック%s", args, f.spaceID, f.topicID)
	}
}

func TestDiscardTopicUsecase_Execute_名前が一致しなければ削除しない(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "discard-mismatch", model.SpaceRoleAdmin, "")
	inserter := &mockJobInserter{}
	uc := newDiscardTopicUsecase(f, inserter)

	output, err := uc.Execute(ctx, DiscardTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		TopicName:       "別の名前",
	})

	if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("topic_name") {
		t.Fatalf("topic_nameのValidationErrorを期待したが、%vだった", err)
	}
	// 確認画面を描画し直せるよう、検証に失敗してもトピックを返す
	if output == nil || output.Topic.ID != f.topicID {
		t.Errorf("output = %+v、トピック%sを期待", output, f.topicID)
	}
	if !isTopicActive(t, f) {
		t.Error("名前が一致しないのにトピックが論理削除された")
	}
	if inserter.called {
		t.Error("名前が一致しないのに物理削除のジョブが投入された")
	}
}

func TestDiscardTopicUsecase_Execute_削除できない閲覧者は拒否する(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "discard-editor", model.SpaceRoleEditor, "")
	inserter := &mockJobInserter{}
	uc := newDiscardTopicUsecase(f, inserter)

	_, err := uc.Execute(ctx, DiscardTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		TopicName:       "日報 discard-editor",
	})

	ae := model.AsAppError(err)
	if ae == nil || ae.Code != model.AppErrCodeForbidden {
		t.Fatalf("ForbiddenのAppErrorを期待したが、%vだった", err)
	}
	if !isTopicActive(t, f) {
		t.Error("topic:deleteを持たないメンバーがトピックを論理削除できた")
	}
	if inserter.called {
		t.Error("topic:deleteを持たないメンバーの送信で物理削除のジョブが投入された")
	}
}

// ジョブを投入できなければ、誰からも見えないのに消えないトピックを残さないよう、論理削除を取り消す。
func TestDiscardTopicUsecase_Execute_ジョブを投入できなければ論理削除を取り消す(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupTopicSettingsGeneralFixture(t, "discard-enqueue-fail", model.SpaceRoleAdmin, "")
	uc := newDiscardTopicUsecase(f, rejectingJobInserter{})

	_, err := uc.Execute(ctx, DiscardTopicInput{
		SpaceIdentifier: f.identifier,
		TopicNumber:     1,
		UserID:          f.userID,
		TopicName:       "日報 discard-enqueue-fail",
	})
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
	if model.AsAppError(err) != nil || model.AsValidationError(err) != nil {
		t.Errorf("想定外のエラーとして扱われるべきだが、%vだった", err)
	}
	if !isTopicActive(t, f) {
		t.Error("ジョブを投入できなかったのにトピックが論理削除されたまま残った")
	}
}

func TestGetTopicDeletionNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)

	tests := []struct {
		name      string
		suffix    string
		spaceRole model.SpaceRole
		topicRole model.TopicRole
		refused   bool
	}{
		{name: "スペースの管理者は開ける", suffix: "deletion-new-admin", spaceRole: model.SpaceRoleAdmin},
		{name: "トピックの管理者のロールを持つ編集者は開ける", suffix: "deletion-new-topic-admin", spaceRole: model.SpaceRoleEditor, topicRole: model.TopicRoleAdmin},
		{name: "topic:deleteを持たない編集者は拒否する", suffix: "deletion-new-editor", spaceRole: model.SpaceRoleEditor, refused: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupTopicSettingsGeneralFixture(t, tt.suffix, tt.spaceRole, tt.topicRole)
			uc := NewGetTopicDeletionNewUsecase(
				repository.NewSpaceRepository(f.queries),
				repository.NewSpaceMemberRepository(f.queries),
				repository.NewTopicRepository(f.queries),
				repository.NewTopicMemberRepository(f.queries),
			)

			output, err := uc.Execute(ctx, GetTopicDeletionNewInput{
				SpaceIdentifier: f.identifier,
				TopicNumber:     1,
				UserID:          f.userID,
			})

			if tt.refused {
				ae := model.AsAppError(err)
				if ae == nil || ae.Code != model.AppErrCodeForbidden {
					t.Fatalf("ForbiddenのAppErrorを期待したが、%vだった", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.Topic.ID != f.topicID {
				t.Errorf("Topic.ID = %v、期待値 = %v", output.Topic.ID, f.topicID)
			}
		})
	}
}
