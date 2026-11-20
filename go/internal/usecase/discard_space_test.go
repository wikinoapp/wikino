package usecase

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// countingJobInserterは、並行に投入されたジョブを数える。
type countingJobInserter struct {
	mu   sync.Mutex
	args []river.JobArgs
}

func (c *countingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.args = append(c.args, args)
	return &rivertype.JobInsertResult{}, nil
}

// discardSpaceFixtureは、スペースの削除のテストが使うスペース・メンバー・トピック。
// 削除は自身でトランザクションを開くため、フィクスチャはDBに直接コミットする。
type discardSpaceFixture struct {
	identifier model.SpaceIdentifier
	spaceName  string
	spaceID    model.SpaceID
	userID     model.UserID
	topicIDs   []model.TopicID
	// discardedTopicIDは、スペースを削除する前に論理削除しておいたトピック
	discardedTopicID model.TopicID
	discardedTopicAt time.Time
}

func setupDiscardSpaceFixture(t *testing.T, identifier string, role model.SpaceRole) discardSpaceFixture {
	t.Helper()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceName := "削除するスペース " + identifier
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).WithName(spaceName).Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).WithRole(role).Build()

	topicIDs := []model.TopicID{
		testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック1").Build(),
		testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(2).WithName("トピック2").Build(),
	}
	discardedTopicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(3).WithName("削除済みのトピック").Build()
	discardedTopicAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	if _, err := db.ExecContext(t.Context(), "UPDATE topics SET discarded_at = $1 WHERE id = $2", discardedTopicAt, string(discardedTopicID)); err != nil {
		t.Fatalf("トピックの論理削除に失敗: %v", err)
	}

	return discardSpaceFixture{
		identifier:       model.SpaceIdentifier(identifier),
		spaceName:        spaceName,
		spaceID:          spaceID,
		userID:           userID,
		topicIDs:         topicIDs,
		discardedTopicID: discardedTopicID,
		discardedTopicAt: discardedTopicAt,
	}
}

func newDiscardSpaceUsecase(inserter dispatcher.JobInserter) *DiscardSpaceUsecase {
	db := testutil.GetTestDB()
	queries := query.New(db)
	return NewDiscardSpaceUsecase(
		db,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewTopicRepository(queries),
		dispatcher.NewDispatcher(inserter, nil),
		validator.NewSpaceDeletionCreateValidator(),
	)
}

// discardedAtは、テーブルの行のdiscarded_atを返す。論理削除していなければValidがfalseになる。
func discardedAt(t *testing.T, table string, id string) sql.NullTime {
	t.Helper()

	var at sql.NullTime
	// tableはテストの中で渡す固定の名前だけを受け取る
	if err := testutil.GetTestDB().QueryRowContext(t.Context(), "SELECT discarded_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil { //nolint:gosec
		t.Fatalf("%sのdiscarded_atの取得に失敗: %v", table, err)
	}
	return at
}

func TestDiscardSpaceUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupDiscardSpaceFixture(t, "discard-space-ok", model.SpaceRoleAdmin)
	inserter := &countingJobInserter{}

	output, err := newDiscardSpaceUsecase(inserter).Execute(ctx, DiscardSpaceInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		SpaceName:       f.spaceName,
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Space.ID != f.spaceID {
		t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
	}

	spaceAt := discardedAt(t, "spaces", string(f.spaceID))
	if !spaceAt.Valid {
		t.Fatal("スペースが論理削除されていない")
	}
	for _, id := range f.topicIDs {
		if at := discardedAt(t, "topics", string(id)); !at.Valid || !at.Time.Equal(spaceAt.Time) {
			t.Errorf("トピック%sのdiscarded_at = %v、期待値 = スペースと同じ%v", id, at, spaceAt.Time)
		}
	}
	if at := discardedAt(t, "topics", string(f.discardedTopicID)); !at.Time.Equal(f.discardedTopicAt) {
		t.Errorf("既に論理削除していたトピックのdiscarded_at = %v、期待値 = %v (変わらない)", at.Time, f.discardedTopicAt)
	}

	if len(inserter.args) != 1 {
		t.Fatalf("投入したジョブの数 = %d、期待値 = 1", len(inserter.args))
	}
	if args, ok := inserter.args[0].(dispatcher.DeleteSpaceArgs); !ok || args.SpaceID != string(f.spaceID) {
		t.Errorf("投入したジョブ = %+v、スペース%sの物理削除を期待", inserter.args[0], f.spaceID)
	}

	// 論理削除したスペースは、識別子で引けなくなる
	space, err := repository.NewSpaceRepository(query.New(testutil.GetTestDB())).FindByIdentifier(ctx, f.identifier)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	if space != nil {
		t.Error("削除したスペースがまだ見える")
	}
}

func TestDiscardSpaceUsecase_Execute_名前が一致しなければ何も削除しない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		spaceName string
		wantMsg   string
	}{
		{name: "名前が違う", spaceName: "別のスペース", wantMsg: "スペース名が間違っています"},
		{name: "名前が空", spaceName: "", wantMsg: "入力してください"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			f := setupDiscardSpaceFixture(t, "discard-space-mismatch-"+string(rune('a'+i)), model.SpaceRoleAdmin)
			inserter := &countingJobInserter{}

			output, err := newDiscardSpaceUsecase(inserter).Execute(ctx, DiscardSpaceInput{
				SpaceIdentifier: f.identifier,
				UserID:          f.userID,
				SpaceName:       tt.spaceName,
			})
			ve := model.AsValidationError(err)
			if ve == nil {
				t.Fatalf("エラー = %v、ValidationErrorを期待", err)
			}
			if errs := ve.GetFieldErrors("space_name"); len(errs) != 1 || errs[0] != tt.wantMsg {
				t.Errorf("space_nameのエラー = %q、期待値 = [%q]", errs, tt.wantMsg)
			}
			if output == nil || output.Space.ID != f.spaceID {
				t.Errorf("output = %+v、確認画面を描画し直すためのスペースを期待", output)
			}
			if discardedAt(t, "spaces", string(f.spaceID)).Valid {
				t.Error("名前が一致しないのにスペースが論理削除された")
			}
			if discardedAt(t, "topics", string(f.topicIDs[0])).Valid {
				t.Error("名前が一致しないのにトピックが論理削除された")
			}
			if len(inserter.args) != 0 {
				t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
			}
		})
	}
}

func TestDiscardSpaceUsecase_Execute_削除できない閲覧者を拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		suffix   string
		role     model.SpaceRole
		outsider bool
		wantCode model.AppErrorCode
	}{
		{name: "space:deleteを持たない編集者", suffix: "editor", role: model.SpaceRoleEditor, wantCode: model.AppErrCodeForbidden},
		{name: "メンバーでない閲覧者", suffix: "outsider", role: model.SpaceRoleAdmin, outsider: true, wantCode: model.AppErrCodeResourceNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
			identifier := "discard-space-deny-" + tt.suffix
			f := setupDiscardSpaceFixture(t, identifier, tt.role)
			userID := f.userID
			if tt.outsider {
				userID = testutil.NewUserBuilderDB(t, testutil.GetTestDB()).
					WithEmail(identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(identifier, "-", "_") + "_other").
					Build()
			}
			inserter := &countingJobInserter{}

			_, err := newDiscardSpaceUsecase(inserter).Execute(ctx, DiscardSpaceInput{
				SpaceIdentifier: f.identifier,
				UserID:          userID,
				SpaceName:       f.spaceName,
			})
			ae := model.AsAppError(err)
			if ae == nil || ae.Code != tt.wantCode {
				t.Fatalf("エラー = %v、コード%vのAppErrorを期待", err, tt.wantCode)
			}
			if discardedAt(t, "spaces", string(f.spaceID)).Valid {
				t.Error("削除できない閲覧者がスペースを論理削除した")
			}
			if len(inserter.args) != 0 {
				t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
			}
		})
	}
}

func TestDiscardSpaceUsecase_Execute_ジョブを投入できなければ今回の論理削除だけを戻す(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupDiscardSpaceFixture(t, "discard-space-enqueue-fail", model.SpaceRoleAdmin)

	_, err := newDiscardSpaceUsecase(rejectingJobInserter{}).Execute(ctx, DiscardSpaceInput{
		SpaceIdentifier: f.identifier,
		UserID:          f.userID,
		SpaceName:       f.spaceName,
	})
	if err == nil {
		t.Fatal("エラーを期待したが、nilだった")
	}
	if model.AsAppError(err) != nil || model.AsValidationError(err) != nil {
		t.Errorf("エラー = %v、システムエラーを期待", err)
	}

	if discardedAt(t, "spaces", string(f.spaceID)).Valid {
		t.Error("スペースの論理削除が取り消されていない")
	}
	for _, id := range f.topicIDs {
		if discardedAt(t, "topics", string(id)).Valid {
			t.Errorf("トピック%sの論理削除が取り消されていない", id)
		}
	}
	if at := discardedAt(t, "topics", string(f.discardedTopicID)); !at.Valid || !at.Time.Equal(f.discardedTopicAt) {
		t.Errorf("既に論理削除していたトピックのdiscarded_at = %v、期待値 = %v (戻さない)", at, f.discardedTopicAt)
	}
}

func TestDiscardSpaceUsecase_Execute_同時に削除しても1回だけ削除する(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	f := setupDiscardSpaceFixture(t, "discard-space-concurrent", model.SpaceRoleAdmin)
	inserter := &countingJobInserter{}
	uc := newDiscardSpaceUsecase(inserter)

	const n = 2
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = uc.Execute(ctx, DiscardSpaceInput{
				SpaceIdentifier: f.identifier,
				UserID:          f.userID,
				SpaceName:       f.spaceName,
			})
		}()
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case model.AsAppError(err) != nil && model.AsAppError(err).Code == model.AppErrCodeResourceNotFound:
		default:
			t.Errorf("Execute()のエラー = %v、成功か「見つからない」を期待", err)
		}
	}
	if succeeded != 1 {
		t.Errorf("成功した削除の数 = %d、期待値 = 1", succeeded)
	}
	if len(inserter.args) != 1 {
		t.Errorf("投入したジョブの数 = %d、期待値 = 1", len(inserter.args))
	}
}

func TestDiscardSpaceUsecase_添付ファイルのURLは論理削除中だけ発行しない(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		suffix    string
		inserter  dispatcher.JobInserter
		wantFound bool
		wantErr   bool
	}{
		{name: "削除の成功後は発行しない", suffix: "hidden", inserter: &countingJobInserter{}},
		{name: "投入の失敗で取り消した後は発行する", suffix: "restored", inserter: rejectingJobInserter{}, wantFound: true, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := setupDiscardSpaceFixture(t, "ds-att-"+tt.suffix, model.SpaceRoleAdmin)
			db := testutil.GetTestDB()
			q := query.New(db)
			members := repository.NewSpaceMemberRepository(q)
			member, err := members.FindActiveBySpaceAndUser(t.Context(), f.spaceID, f.userID)
			if err != nil || member == nil {
				t.Fatalf("スペースメンバーの取得に失敗: %v", err)
			}
			id := testutil.NewAttachmentBuilderDB(t, db).WithSpaceID(f.spaceID).WithSpaceMemberID(member.ID).WithBlobKey("ds-att-" + tt.suffix).Build()

			// 同じメンバーが別のスペースで持つ添付ファイルを、一括取得から消さないことも確かめる。
			otherSpaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("ds-att-other-" + tt.suffix).Build()
			otherMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(otherSpaceID).WithUserID(f.userID).WithRole(model.SpaceRoleAdmin).Build()
			otherID := testutil.NewAttachmentBuilderDB(t, db).WithSpaceID(otherSpaceID).WithSpaceMemberID(otherMemberID).WithBlobKey("ds-att-other-" + tt.suffix).Build()
			objects := storage.NewFakeObjectStorage()
			objects.Put("ds-att-"+tt.suffix, []byte("file"), "image/png")
			attachments := repository.NewAttachmentRepository(q)
			singleUC := NewGetAttachmentUsecase(attachments, members, objects)
			batchUC := NewGetAttachmentSignedURLsUsecase(attachments, members, objects)

			assertAccess := func(wantFound bool) {
				t.Helper()
				output, err := singleUC.Execute(t.Context(), GetAttachmentInput{AttachmentID: id, UserID: &f.userID})
				if wantFound {
					if err != nil || output == nil || output.DownloadURL == "" {
						t.Fatalf("単件の取得結果 = %+v、エラー = %v、URLの発行を期待", output, err)
					}
				} else if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound || output != nil {
					t.Errorf("単件の取得結果 = %+v、エラー = %v、未存在を期待", output, err)
				}
				batch, err := batchUC.Execute(t.Context(), GetAttachmentSignedURLsInput{AttachmentIDs: []model.AttachmentID{id, otherID}, UserID: &f.userID})
				if err != nil {
					t.Fatalf("一括取得のエラー = %v", err)
				}
				if got := batch.SignedURLs[id] != ""; got != wantFound {
					t.Errorf("対象のURLを発行したか = %v、期待値 = %v", got, wantFound)
				}
				if batch.SignedURLs[otherID] == "" {
					t.Error("別のスペースの添付ファイルのURLが無い")
				}
			}
			assertAccess(true)
			_, err = newDiscardSpaceUsecase(tt.inserter).Execute(t.Context(), DiscardSpaceInput{SpaceIdentifier: f.identifier, UserID: f.userID, SpaceName: f.spaceName})
			if (err != nil) != tt.wantErr {
				t.Fatalf("スペースの削除結果 = %v、投入の失敗を期待するか = %v", err, tt.wantErr)
			}
			// Workerを実行せず、行と実体が残った状態でも、論理削除の結果で参照を制御する。
			assertAccess(tt.wantFound)
		})
	}
}
