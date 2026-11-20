package usecase

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// restorePagesSpaceは一括復元のテストで使う、DBにコミットしたスペースとメンバー。
type restorePagesSpace struct {
	identifier    model.SpaceIdentifier
	spaceID       model.SpaceID
	spaceMemberID model.SpaceMemberID
	userID        model.UserID
}

// setupRestorePagesSpaceは、指定したロールのメンバーがいるスペースをDBにコミットして作る。
// 一括復元のUseCaseは自分でトランザクションを開くため、テストのトランザクションに包まない。
func setupRestorePagesSpace(t *testing.T, db *sql.DB, suffix string, spaceRole model.SpaceRole) restorePagesSpace {
	t.Helper()

	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("restore-pages-" + suffix + "@example.com").
		WithAtname(strings.ReplaceAll("restore_pages_"+suffix, "-", "_")).
		Build()
	identifier := "restore-pages-" + suffix
	spaceID := testutil.NewSpaceBuilderDB(t, db).
		WithIdentifier(identifier).
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithRole(spaceRole).
		Build()

	return restorePagesSpace{
		identifier:    model.SpaceIdentifier(identifier),
		spaceID:       spaceID,
		spaceMemberID: spaceMemberID,
		userID:        userID,
	}
}

// addTopicはスペースにトピックを作る。topicRoleを渡すと、メンバーをそのロールで参加させる。
func (s restorePagesSpace) addTopic(t *testing.T, db *sql.DB, number int32, topicRole model.TopicRole) model.TopicID {
	t.Helper()

	topicID := testutil.NewTopicBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithNumber(number).
		WithName("トピック" + string(rune('A'+number-1))).
		Build()
	if topicRole != "" {
		testutil.NewTopicMemberBuilderDB(t, db).
			WithSpaceID(s.spaceID).
			WithTopicID(topicID).
			WithSpaceMemberID(s.spaceMemberID).
			WithRole(topicRole).
			Build()
	}
	return topicID
}

// addTrashedPageはトピックに、1時間前にゴミ箱へ入れたページを作る。
func (s restorePagesSpace) addTrashedPage(t *testing.T, db *sql.DB, topicID model.TopicID, number model.PageNumber) model.PageID {
	t.Helper()

	return testutil.NewPageBuilderDB(t, db).
		WithSpaceID(s.spaceID).
		WithTopicID(topicID).
		WithNumber(number).
		WithTitle("ページ" + string(rune('0'+number))).
		WithTrashedAt(time.Now().Add(-time.Hour)).
		Build()
}

func newRestorePagesUsecase(db *sql.DB) *RestorePagesUsecase {
	queries := query.New(db)
	pageRepo := repository.NewPageRepository(queries)
	return NewRestorePagesUsecase(
		db,
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		pageRepo,
		repository.NewTopicRepository(queries),
		repository.NewTopicMemberRepository(queries),
		validator.NewBulkRestoredPageCreateValidator(pageRepo),
	)
}

// isPageTrashedはページがゴミ箱に入っているかを返す。
func isPageTrashed(t *testing.T, db *sql.DB, pageID model.PageID) bool {
	t.Helper()

	var trashed bool
	if err := db.QueryRowContext(context.Background(), "SELECT trashed_at IS NOT NULL FROM pages WHERE id = $1", string(pageID)).Scan(&trashed); err != nil {
		t.Fatalf("ページの取得に失敗: %v", err)
	}
	return trashed
}

func TestRestorePagesUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()

	t.Run("選んだページをゴミ箱から戻す", func(t *testing.T) {
		t.Parallel()

		s := setupRestorePagesSpace(t, db, "success", model.SpaceRoleAdmin)
		topicID := s.addTopic(t, db, 1, "")
		page1 := s.addTrashedPage(t, db, topicID, 1)
		page2 := s.addTrashedPage(t, db, topicID, 2)
		untouched := s.addTrashedPage(t, db, topicID, 3)

		output, err := newRestorePagesUsecase(db).Execute(ctx, RestorePagesInput{
			SpaceIdentifier: s.identifier,
			UserID:          s.userID,
			PageIDs:         []string{string(page1), string(page2)},
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.Space.ID != s.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, s.spaceID)
		}
		if isPageTrashed(t, db, page1) || isPageTrashed(t, db, page2) {
			t.Error("選んだページがゴミ箱に残っている")
		}
		if !isPageTrashed(t, db, untouched) {
			t.Error("選んでいないページがゴミ箱から戻った")
		}
	})

	t.Run("page_trash:deleteを持たないトピックのページを含むと何も戻さない", func(t *testing.T) {
		t.Parallel()

		// スペースの閲覧者は、トピックの編集者として参加したトピックでだけページを戻せる
		s := setupRestorePagesSpace(t, db, "forbidden-topic", model.SpaceRoleViewer)
		editorTopicID := s.addTopic(t, db, 1, model.TopicRoleEditor)
		viewerTopicID := s.addTopic(t, db, 2, model.TopicRoleViewer)
		restorable := s.addTrashedPage(t, db, editorTopicID, 1)
		forbidden := s.addTrashedPage(t, db, viewerTopicID, 2)

		_, err := newRestorePagesUsecase(db).Execute(ctx, RestorePagesInput{
			SpaceIdentifier: s.identifier,
			UserID:          s.userID,
			PageIDs:         []string{string(restorable), string(forbidden)},
		})

		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("page_ids") {
			t.Fatalf("page_idsのValidationErrorを期待したが、%vだった", err)
		}
		if !isPageTrashed(t, db, restorable) || !isPageTrashed(t, db, forbidden) {
			t.Error("復元できないページを含むのに、ページがゴミ箱から戻った")
		}
	})

	t.Run("別のスペースのページは指定できない", func(t *testing.T) {
		t.Parallel()

		s := setupRestorePagesSpace(t, db, "other-space", model.SpaceRoleAdmin)
		topicID := s.addTopic(t, db, 1, "")
		page := s.addTrashedPage(t, db, topicID, 1)

		other := setupRestorePagesSpace(t, db, "other-space-target", model.SpaceRoleAdmin)
		otherTopicID := other.addTopic(t, db, 1, "")
		otherPage := other.addTrashedPage(t, db, otherTopicID, 1)

		_, err := newRestorePagesUsecase(db).Execute(ctx, RestorePagesInput{
			SpaceIdentifier: s.identifier,
			UserID:          s.userID,
			PageIDs:         []string{string(page), string(otherPage)},
		})

		if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("page_ids") {
			t.Fatalf("page_idsのValidationErrorを期待したが、%vだった", err)
		}
		if !isPageTrashed(t, db, page) || !isPageTrashed(t, db, otherPage) {
			t.Error("別のスペースのページを含むのに、ページがゴミ箱から戻った")
		}
	})

	t.Run("どのトピックでもpage_trash:deleteを持たないメンバーは拒否する", func(t *testing.T) {
		t.Parallel()

		s := setupRestorePagesSpace(t, db, "no-permission", model.SpaceRoleViewer)
		topicID := s.addTopic(t, db, 1, model.TopicRoleViewer)
		page := s.addTrashedPage(t, db, topicID, 1)

		_, err := newRestorePagesUsecase(db).Execute(ctx, RestorePagesInput{
			SpaceIdentifier: s.identifier,
			UserID:          s.userID,
			PageIDs:         []string{string(page)},
		})

		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeForbidden {
			t.Fatalf("ForbiddenのAppErrorを期待したが、%vだった", err)
		}
		if !isPageTrashed(t, db, page) {
			t.Error("権限が無いのにページがゴミ箱から戻った")
		}
	})
}

// 検証のあとに別の操作でページが復元できないトピックへ移ったときは、一部だけを戻さずに取り消す。
func TestRestorePagesUsecase_restorePages_戻せないページがあれば取り消す(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()

	s := setupRestorePagesSpace(t, db, "conflict", model.SpaceRoleAdmin)
	restorableTopicID := s.addTopic(t, db, 1, "")
	movedTopicID := s.addTopic(t, db, 2, "")
	page := s.addTrashedPage(t, db, restorableTopicID, 1)
	moved := s.addTrashedPage(t, db, movedTopicID, 2)

	pages, err := repository.NewPageRepository(query.New(db)).FindByIDs(ctx, []model.PageID{page, moved}, s.spaceID)
	if err != nil {
		t.Fatalf("ページの取得に失敗: %v", err)
	}

	err = newRestorePagesUsecase(db).restorePages(ctx, s.spaceID, pages, map[model.TopicID]bool{restorableTopicID: true}, time.Now())

	ae := model.AsAppError(err)
	if ae == nil || ae.Code != model.AppErrCodeConflict {
		t.Fatalf("ConflictのAppErrorを期待したが、%vだった", err)
	}
	if !isPageTrashed(t, db, page) {
		t.Error("一部のページだけがゴミ箱から戻った")
	}
}

// 検証後に別の送信がページを復元しても、残りの選択を復元して成功することを確認する。
func TestRestorePagesUsecase_Execute_先行する復元と競合しても成功する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		suffix       string
		restoreCount int
	}{
		{name: "一部を先に復元", suffix: "concurrent-partial", restoreCount: 1},
		{name: "全件を先に復元", suffix: "concurrent-all", restoreCount: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(i18n.SetLocale(context.Background(), i18n.LangJa), 10*time.Second)
			defer cancel()
			db := testutil.GetTestDB()
			s := setupRestorePagesSpace(t, db, tt.suffix, model.SpaceRoleAdmin)
			topicID := s.addTopic(t, db, 1, "")
			selected := []model.PageID{
				s.addTrashedPage(t, db, topicID, 1),
				s.addTrashedPage(t, db, topicID, 2),
			}
			untouched := s.addTrashedPage(t, db, topicID, 3)

			first, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("先行トランザクションの開始に失敗: %v", err)
			}
			defer func() { _ = first.Rollback() }()

			var blockingPID int
			if err := first.QueryRowContext(ctx, "SELECT pg_backend_pid()").Scan(&blockingPID); err != nil {
				t.Fatalf("先行トランザクションのPID取得に失敗: %v", err)
			}
			restoredAt := time.Now().UTC().Add(-30 * time.Minute).Truncate(time.Microsecond)
			if _, err := repository.NewPageRepository(query.New(first)).RestoreFromTrash(ctx, selected[:tt.restoreCount], []model.TopicID{topicID}, s.spaceID, restoredAt); err != nil {
				t.Fatalf("先行する復元に失敗: %v", err)
			}

			result := make(chan error, 1)
			go func() {
				_, err := newRestorePagesUsecase(db).Execute(ctx, RestorePagesInput{
					SpaceIdentifier: s.identifier,
					UserID:          s.userID,
					PageIDs:         model.PageIDsToStrings(selected),
				})
				result <- err
			}()

			// 先行する復元は検証から見えず、更新が行ロックを待つ状態になってからコミットする。
			waitForQueryBlockedBy(t, ctx, db, blockingPID, "RestorePagesFromTrash", result)
			if err := first.Commit(); err != nil {
				t.Fatalf("先行する復元のコミットに失敗: %v", err)
			}
			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("Execute()のエラー = %v", err)
				}
			case <-ctx.Done():
				t.Fatalf("復元が完了しなかった: %v", ctx.Err())
			}

			for _, pageID := range selected {
				if isPageTrashed(t, db, pageID) {
					t.Errorf("選んだページ%vがゴミ箱に残っている", pageID)
				}
			}
			if !isPageTrashed(t, db, untouched) {
				t.Error("選んでいないページがゴミ箱から戻った")
			}
			for _, pageID := range selected[:tt.restoreCount] {
				var updatedAt time.Time
				if err := db.QueryRowContext(ctx, "SELECT updated_at FROM pages WHERE id = $1 AND space_id = $2", pageID, s.spaceID).Scan(&updatedAt); err != nil {
					t.Fatalf("先に復元されたページの取得に失敗: %v", err)
				}
				if !updatedAt.Equal(restoredAt) {
					t.Errorf("先に復元されたページの更新日時 = %v、期待値 = %v", updatedAt, restoredAt)
				}
			}
		})
	}
}
