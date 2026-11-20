package repository

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestAttachmentRepository_ExistsByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-exists@example.com").
		WithAtname("attachexists").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-exists-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("photo.jpg").WithContentType("image/jpeg").Build()

	t.Run("存在する添付ファイルはtrueを返す", func(t *testing.T) {
		exists, err := repo.ExistsByIDAndSpace(context.Background(), attachmentID, spaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace()のエラー = %v", err)
		}
		if !exists {
			t.Error("ExistsByIDAndSpace() = false、期待値 = true")
		}
	})

	t.Run("存在しないIDはfalseを返す", func(t *testing.T) {
		exists, err := repo.ExistsByIDAndSpace(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByIDAndSpace() = true、期待値 = false")
		}
	})

	t.Run("異なるスペースIDはfalseを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-exists-other").
			Build()

		exists, err := repo.ExistsByIDAndSpace(context.Background(), attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("ExistsByIDAndSpace()のエラー = %v", err)
		}
		if exists {
			t.Error("ExistsByIDAndSpace() = true、期待値 = false")
		}
	})
}

func TestAttachmentRepository_FindByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-find@example.com").
		WithAtname("attachfind").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-find-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("photo.jpg").WithContentType("image/jpeg").Build()

	t.Run("存在する添付ファイルを取得できる", func(t *testing.T) {
		attachment, err := repo.FindByIDAndSpace(context.Background(), attachmentID, spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindByIDAndSpace()がnilを返した、期待値 = 添付ファイル")
		}
		if attachment.ID != attachmentID {
			t.Errorf("attachment.ID = %v、期待値 = %v", attachment.ID, attachmentID)
		}
		if attachment.SpaceID != spaceID {
			t.Errorf("attachment.SpaceID = %v、期待値 = %v", attachment.SpaceID, spaceID)
		}
		if attachment.Filename != "photo.jpg" {
			t.Errorf("attachment.Filename = %v、期待値 = %v", attachment.Filename, "photo.jpg")
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindByIDAndSpace(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindByIDAndSpace() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("異なるスペースIDはnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-find-other").
			Build()

		attachment, err := repo.FindByIDAndSpace(context.Background(), attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindByIDAndSpace() = %v、期待値 = nil", attachment)
		}
	})
}

func TestAttachmentRepository_FindBlobByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-find-blob-by-id@example.com").
		WithAtname("attachfindblobbyid").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-find-blob-by-id").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("report.pdf").
		WithContentType("application/pdf").
		WithByteSize(4096).
		WithBlobKey("attach-find-blob-by-id-key").
		Build()

	t.Run("スペースを指定せずに、スペース・ファイル名・種類・バイト数・ストレージキーを取得できる", func(t *testing.T) {
		attachment, err := repo.FindBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindBlobByID()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindBlobByID()がnilを返した、期待値 = 添付ファイル")
		}
		want := model.Attachment{
			ID:          attachmentID,
			SpaceID:     spaceID,
			Filename:    "report.pdf",
			BlobKey:     "attach-find-blob-by-id-key",
			ContentType: "application/pdf",
			ByteSize:    4096,
		}
		if *attachment != want {
			t.Errorf("FindBlobByID() = %+v、期待値 = %+v", *attachment, want)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindBlobByID(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"))
		if err != nil {
			t.Fatalf("FindBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("UUIDでないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindBlobByID(context.Background(), model.AttachmentID("not-a-uuid"))
		if err != nil {
			t.Fatalf("FindBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindBlobByID() = %v、期待値 = nil", attachment)
		}
	})
}

func TestAttachmentRepository_FindBlobsByIDs(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewAttachmentRepository(testutil.QueriesWithTx(tx))

	spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-find-blobs")
	otherSpaceID, otherSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-find-blobs-other")
	firstID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
		WithFilename("a.png").WithContentType("image/png").WithByteSize(10).WithBlobKey("attach-find-blobs-a").Build()
	secondID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(otherSpaceID).WithSpaceMemberID(otherSpaceMemberID).
		WithFilename("b.pdf").WithContentType("application/pdf").WithByteSize(20).WithBlobKey("attach-find-blobs-b").Build()

	t.Run("複数のスペースの添付ファイルを、スペースとあわせて一括取得する", func(t *testing.T) {
		attachments, err := repo.FindBlobsByIDs(context.Background(), []model.AttachmentID{
			firstID, secondID, "00000000-0000-0000-0000-000000000000", "not-a-uuid",
		})
		if err != nil {
			t.Fatalf("FindBlobsByIDs()のエラー = %v", err)
		}
		got := map[model.AttachmentID]model.Attachment{}
		for _, a := range attachments {
			got[a.ID] = *a
		}
		want := map[model.AttachmentID]model.Attachment{
			firstID:  {ID: firstID, SpaceID: spaceID, Filename: "a.png", BlobKey: "attach-find-blobs-a", ContentType: "image/png", ByteSize: 10},
			secondID: {ID: secondID, SpaceID: otherSpaceID, Filename: "b.pdf", BlobKey: "attach-find-blobs-b", ContentType: "application/pdf", ByteSize: 20},
		}
		if len(got) != len(want) {
			t.Fatalf("件数 = %d、期待値 = %d (%v)", len(got), len(want), got)
		}
		for id, w := range want {
			if got[id] != w {
				t.Errorf("FindBlobsByIDs()[%s] = %+v、期待値 = %+v", id, got[id], w)
			}
		}
	})

	t.Run("UUIDの形のIDが無ければクエリを発行せずにnilを返す", func(t *testing.T) {
		attachments, err := repo.FindBlobsByIDs(context.Background(), []model.AttachmentID{"not-a-uuid"})
		if err != nil {
			t.Fatalf("FindBlobsByIDs()のエラー = %v", err)
		}
		if attachments != nil {
			t.Errorf("FindBlobsByIDs() = %v、期待値 = nil", attachments)
		}
	})
}

func TestAttachmentRepository_ListPubliclyReferencedIDsByAttachments(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)
	parRepo := NewPageAttachmentReferenceRepository(q)

	spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-list-public")
	publicTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("公開").Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("非公開").
		WithVisibility(int32(model.TopicVisibilityPrivate)).Build()

	newReferenced := func(name string, topicID model.TopicID, number model.PageNumber, trashed bool) *model.Attachment {
		id := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
			WithBlobKey("attach-list-public-" + name).Build()
		attachment := &model.Attachment{ID: id, SpaceID: spaceID}
		if topicID == "" {
			return attachment
		}
		pageBuilder := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(number).WithTitle(name)
		if trashed {
			pageBuilder = pageBuilder.WithTrashed()
		}
		if _, err := parRepo.CreateBatch(context.Background(), pageBuilder.Build(), spaceID, []model.AttachmentID{id}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		return attachment
	}
	public := newReferenced("public", publicTopicID, 1, false)
	private := newReferenced("private", privateTopicID, 2, false)
	trashed := newReferenced("trashed", publicTopicID, 3, true)
	unreferenced := newReferenced("unreferenced", "", 0, false)

	otherSpaceID, otherSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-list-public-other")
	otherTopicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(otherSpaceID).WithNumber(1).WithName("公開").Build()
	otherPageID := testutil.NewPageBuilder(t, tx).WithSpaceID(otherSpaceID).WithTopicID(otherTopicID).WithNumber(1).WithTitle("公開").Build()
	otherID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(otherSpaceID).WithSpaceMemberID(otherSpaceMemberID).
		WithBlobKey("attach-list-public-other").Build()
	if _, err := parRepo.CreateBatch(context.Background(), otherPageID, otherSpaceID, []model.AttachmentID{otherID}); err != nil {
		t.Fatalf("別スペースの参照の作成に失敗: %v", err)
	}
	other := &model.Attachment{ID: otherID, SpaceID: otherSpaceID}

	tests := []struct {
		name        string
		attachments []*model.Attachment
		want        []model.AttachmentID
	}{
		{
			name:        "公開トピックの生きているページだけが参照する添付ファイルを返す",
			attachments: []*model.Attachment{public, private, trashed, unreferenced, {ID: "not-a-uuid", SpaceID: spaceID}},
			want:        []model.AttachmentID{public.ID},
		},
		{
			name:        "複数スペースの添付ファイルを所属スペースとの組で判定する",
			attachments: []*model.Attachment{public, other},
			want:        []model.AttachmentID{public.ID, other.ID},
		},
		{
			name:        "同じ組を重複して渡してもIDは1回だけ返す",
			attachments: []*model.Attachment{public, public, other},
			want:        []model.AttachmentID{public.ID, other.ID},
		},
		{
			name:        "別スペースを指定すると公開添付ファイルでも返さない",
			attachments: []*model.Attachment{{ID: public.ID, SpaceID: otherSpaceID}},
		},
		{
			name:        "IDとスペースを入れ替えると両方とも返さない",
			attachments: []*model.Attachment{{ID: public.ID, SpaceID: otherSpaceID}, {ID: other.ID, SpaceID: spaceID}},
		},
		{
			name:        "正しい組と誤った組が混在すれば正しい組だけ返す",
			attachments: []*model.Attachment{{ID: public.ID, SpaceID: otherSpaceID}, other},
			want:        []model.AttachmentID{other.ID},
		},
		{
			name: "空の入力は結果を返さない",
		},
		{
			name:        "UUIDでないIDやスペースの組は省く",
			attachments: []*model.Attachment{{ID: "not-a-uuid", SpaceID: spaceID}, {ID: public.ID, SpaceID: "invalid-space"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := repo.ListPubliclyReferencedIDsByAttachments(t.Context(), tt.attachments)
			if err != nil {
				t.Fatalf("ListPubliclyReferencedIDsByAttachments()のエラー = %v", err)
			}
			if len(ids) != len(tt.want) {
				t.Fatalf("公開の添付ファイル = %v、期待値 = %v", ids, tt.want)
			}
			got := make(map[model.AttachmentID]bool, len(ids))
			for _, id := range ids {
				got[id] = true
			}
			for _, id := range tt.want {
				if !got[id] {
					t.Errorf("公開添付ファイル%sが含まれていない (結果 = %v)", id, ids)
				}
			}
		})
	}
}

func TestAttachmentRepository_FindBlobByIDAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-find-blob@example.com").
		WithAtname("attachfindblob").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-find-blob-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("diagram.svg").
		WithContentType("image/svg+xml").
		WithByteSize(2048).
		WithBlobKey("attach-find-blob-key").
		Build()

	t.Run("ファイル名・種類・バイト数・ストレージキーを取得できる", func(t *testing.T) {
		attachment, err := repo.FindBlobByIDAndSpace(context.Background(), attachmentID, spaceID)
		if err != nil {
			t.Fatalf("FindBlobByIDAndSpace()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindBlobByIDAndSpace()がnilを返した、期待値 = 添付ファイル")
		}
		want := model.Attachment{
			ID:          attachmentID,
			SpaceID:     spaceID,
			Filename:    "diagram.svg",
			BlobKey:     "attach-find-blob-key",
			ContentType: "image/svg+xml",
			ByteSize:    2048,
		}
		if *attachment != want {
			t.Errorf("FindBlobByIDAndSpace() = %+v、期待値 = %+v", *attachment, want)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindBlobByIDAndSpace(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"), spaceID)
		if err != nil {
			t.Fatalf("FindBlobByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindBlobByIDAndSpace() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("UUIDでないIDはnilを返す", func(t *testing.T) {
		attachment, err := repo.FindBlobByIDAndSpace(context.Background(), model.AttachmentID("not-a-uuid"), spaceID)
		if err != nil {
			t.Fatalf("FindBlobByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindBlobByIDAndSpace() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("異なるスペースIDはnilを返す", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-find-blob-other").
			Build()

		attachment, err := repo.FindBlobByIDAndSpace(context.Background(), attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindBlobByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindBlobByIDAndSpace() = %v、期待値 = nil", attachment)
		}
	})
}

func TestAttachmentRepository_FindPubliclyReferencedBlobByIDAndSpace(t *testing.T) {
	t.Parallel()

	type reference struct {
		visibility     model.TopicVisibility
		trashed        bool
		discardedPage  bool
		discardedTopic bool
	}
	tests := []struct {
		name         string
		suffix       string
		references   []reference
		wrongSpace   bool
		attachmentID model.AttachmentID
		wantFound    bool
	}{
		{name: "所属スペースの公開添付ファイルを取得できる", suffix: "public", references: []reference{{visibility: model.TopicVisibilityPublic}}, wantFound: true},
		{name: "同じ添付ファイルIDでも異なるスペースでは取得できない", suffix: "other-space", references: []reference{{visibility: model.TopicVisibilityPublic}}, wrongSpace: true},
		{name: "非公開トピックの参照があれば取得できない", suffix: "private", references: []reference{{visibility: model.TopicVisibilityPrivate}}},
		{name: "公開と非公開の参照が混在すると取得できない", suffix: "mixed", references: []reference{{visibility: model.TopicVisibilityPublic}, {visibility: model.TopicVisibilityPrivate}}},
		{name: "参照がなければ取得できない", suffix: "unreferenced"},
		{name: "ゴミ箱のページだけの参照では取得できない", suffix: "trashed", references: []reference{{visibility: model.TopicVisibilityPublic, trashed: true}}},
		{name: "削除済みページだけの参照では取得できない", suffix: "discarded-page", references: []reference{{visibility: model.TopicVisibilityPublic, discardedPage: true}}},
		{name: "削除済みトピックだけの参照では取得できない", suffix: "discarded-topic", references: []reference{{visibility: model.TopicVisibilityPublic, discardedTopic: true}}},
		{name: "ゴミ箱の非公開ページの参照は公開判定から除く", suffix: "ignore-trashed", references: []reference{{visibility: model.TopicVisibilityPublic}, {visibility: model.TopicVisibilityPrivate, trashed: true}}, wantFound: true},
		{name: "削除済みの非公開ページの参照は公開判定から除く", suffix: "ignore-page", references: []reference{{visibility: model.TopicVisibilityPublic}, {visibility: model.TopicVisibilityPrivate, discardedPage: true}}, wantFound: true},
		{name: "削除済みの非公開トピックの参照は公開判定から除く", suffix: "ignore-topic", references: []reference{{visibility: model.TopicVisibilityPublic}, {visibility: model.TopicVisibilityPrivate, discardedTopic: true}}, wantFound: true},
		{name: "存在しないIDは取得できない", suffix: "missing", attachmentID: "00000000-0000-0000-0000-000000000000"},
		{name: "UUIDでないIDは取得できない", suffix: "invalid", attachmentID: "not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			repo := NewAttachmentRepository(q)
			parRepo := NewPageAttachmentReferenceRepository(q)
			prefix := "attach-scoped-" + tt.suffix
			spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, prefix)
			attachmentID := testutil.NewAttachmentBuilder(t, tx).
				WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
				WithContentType("image/png").WithBlobKey(prefix + "-key").Build()
			for i, ref := range tt.references {
				topic := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).
					WithNumber(int32(i + 1)).WithName(fmt.Sprintf("reference-%d", i+1)).WithVisibility(int32(ref.visibility))
				if ref.discardedTopic {
					topic.WithDiscarded()
				}
				page := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topic.Build()).
					WithNumber(model.PageNumber(i + 1)).WithTitle("参照元")
				if ref.trashed {
					page.WithTrashed()
				}
				if ref.discardedPage {
					page.WithDiscarded()
				}
				if _, err := parRepo.CreateBatch(t.Context(), page.Build(), spaceID, []model.AttachmentID{attachmentID}); err != nil {
					t.Fatalf("添付ファイルの参照の作成に失敗: %v", err)
				}
			}

			lookupSpaceID := spaceID
			if tt.wrongSpace {
				lookupSpaceID = testutil.NewSpaceBuilder(t, tx).WithIdentifier(prefix + "-other").Build()
			}
			if tt.attachmentID != "" {
				attachmentID = tt.attachmentID
			}
			got, err := repo.FindPubliclyReferencedBlobByIDAndSpace(t.Context(), attachmentID, lookupSpaceID)
			if err != nil {
				t.Fatalf("FindPubliclyReferencedBlobByIDAndSpace()のエラー = %v", err)
			}
			if !tt.wantFound {
				if got != nil {
					t.Errorf("取得結果 = %+v、期待値 = nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("添付ファイルを取得できなかった、期待値 = 公開添付ファイル")
			}
			want := model.Attachment{ID: attachmentID, SpaceID: spaceID, BlobKey: prefix + "-key", ContentType: "image/png"}
			if *got != want {
				t.Errorf("取得結果 = %+v、期待値 = %+v", *got, want)
			}
		})
	}
}

func TestAttachmentRepository_FindPubliclyReferencedBlobByID(t *testing.T) {
	t.Parallel()

	t.Run("公開トピックのページから参照されている添付はblob情報を取得できる", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-1")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithFilename("og.png").
			WithContentType("image/png").
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID()がnilを返した、期待値 = 添付ファイル")
		}
		if attachment.ID != attachmentID {
			t.Errorf("attachment.ID = %v、期待値 = %v", attachment.ID, attachmentID)
		}
		if attachment.SpaceID != spaceID {
			t.Errorf("attachment.SpaceID = %v、期待値 = %v", attachment.SpaceID, spaceID)
		}
		if attachment.BlobKey == "" {
			t.Error("attachment.BlobKeyが空")
		}
		if attachment.ContentType != "image/png" {
			t.Errorf("attachment.ContentType = %v、期待値 = image/png", attachment.ContentType)
		}
	})

	t.Run("非公開トピックのページから参照されている添付はnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-2")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page 1").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("公開と非公開のページから混在参照されている添付はnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-3")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(privateTopicID).
			WithNumber(2).
			WithTitle("Private").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), privatePageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("どのページからも参照されていない添付はnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-4")
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("論理削除されたページからの参照は無視される (有効な公開参照がない場合nil)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-5")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Discarded").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		// ページを論理削除する
		if _, err := tx.ExecContext(context.Background(),
			"UPDATE pages SET discarded_at = NOW() WHERE id = $1", string(pageID)); err != nil {
			t.Fatalf("論理削除に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("論理削除されたトピックからの参照は無視される (有効な公開参照がない場合nil)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-6")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			WithDiscarded().
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Page in discarded topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("公開トピックのゴミ箱に入ったページから参照されている添付はnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-7")
		topicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(1).
			WithTitle("Trashed").
			WithTrashed().
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("非公開トピックのゴミ箱に入ったページからの参照はvisibility判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-8")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		trashedPrivatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(privateTopicID).
			WithNumber(2).
			WithTitle("Trashed private").
			WithTrashed().
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), trashedPrivatePageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID()がnilを返した、期待値 = 添付ファイル")
		}
	})

	// EXISTS側の `p.space_id = a.space_id` を単独で固定する。トピックはattachmentと
	// 同じspaceに置くため、この参照を除外できるのはページ側の条件だけになる。
	t.Run("参照元ページが別spaceにある参照だけでは添付を公開しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-9-attachment")
		pageSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-9-page")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(pageSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Foreign public").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		// CreateBatchはpageとattachmentのspaceが異なる参照を作れないため直接INSERTする。
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
			 VALUES ($1, $2, NOW(), NOW())`,
			string(attachmentID),
			string(publicPageID),
		); err != nil {
			t.Fatalf("cross-spaceの参照作成に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	// EXISTS側の `t.space_id = a.space_id` を単独で固定する。ページはattachmentと
	// 同じspaceに置くため、この参照を除外できるのはトピック側の条件だけになる。
	t.Run("参照元ページのトピックが別spaceにある参照だけでは添付を公開しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-11-attachment")
		topicSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-11-topic")
		foreignTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(topicSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(foreignTopicID).
			WithNumber(1).
			WithTitle("Page in a foreign topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), pageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	// NOT EXISTS側の `p.space_id = a.space_id` を単独で固定する。非公開トピックは
	// attachmentと同じspaceに置くため、visibility判定から外せるのはページ側の条件だけになる。
	t.Run("別spaceのページからの非公開参照はvisibility判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-10-attachment")
		pageSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-10-page")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		privateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(2).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(pageSpaceID).
			WithTopicID(privateTopicID).
			WithNumber(1).
			WithTitle("Foreign private").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		// CreateBatchはpageとattachmentのspaceが異なる参照を作れないため直接INSERTする。
		if _, err := tx.ExecContext(
			context.Background(),
			`INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
			 VALUES ($1, $2, NOW(), NOW())`,
			string(attachmentID),
			string(privatePageID),
		); err != nil {
			t.Fatalf("cross-spaceの参照作成に失敗: %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID()がnilを返した、期待値 = 添付ファイル")
		}
		if attachment.SpaceID != attachmentSpaceID {
			t.Errorf("attachment.SpaceID = %v、期待値 = %v", attachment.SpaceID, attachmentSpaceID)
		}
	})

	// NOT EXISTS側の `t.space_id = a.space_id` を単独で固定する。非公開ページは
	// attachmentと同じspaceに置くため、visibility判定から外せるのはトピック側の条件だけになる。
	t.Run("別spaceのトピックに属する非公開参照はvisibility判定に影響しない", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)
		parRepo := NewPageAttachmentReferenceRepository(q)

		attachmentSpaceID, attachmentSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-pub-12-attachment")
		topicSpaceID, _ := testutil.SetupSpaceWithMember(t, tx, "attach-pub-12-topic")
		publicTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithNumber(1).
			WithName("public").
			WithVisibility(int32(model.TopicVisibilityPublic)).
			Build()
		foreignPrivateTopicID := testutil.NewTopicBuilder(t, tx).
			WithSpaceID(topicSpaceID).
			WithNumber(1).
			WithName("private").
			WithVisibility(int32(model.TopicVisibilityPrivate)).
			Build()
		publicPageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(publicTopicID).
			WithNumber(1).
			WithTitle("Public").
			Build()
		privatePageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithTopicID(foreignPrivateTopicID).
			WithNumber(2).
			WithTitle("Page in a foreign private topic").
			Build()
		attachmentID := testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(attachmentSpaceID).
			WithSpaceMemberID(attachmentSpaceMemberID).
			Build()
		if _, err := parRepo.CreateBatch(context.Background(), publicPageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}
		if _, err := parRepo.CreateBatch(context.Background(), privatePageID, attachmentSpaceID, []model.AttachmentID{attachmentID}); err != nil {
			t.Fatalf("CreateBatch()のエラー = %v", err)
		}

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), attachmentID)
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindPubliclyReferencedBlobByID()がnilを返した、期待値 = 添付ファイル")
		}
		if attachment.SpaceID != attachmentSpaceID {
			t.Errorf("attachment.SpaceID = %v、期待値 = %v", attachment.SpaceID, attachmentSpaceID)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), model.AttachmentID("00000000-0000-0000-0000-000000000000"))
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})

	t.Run("UUID形式でないIDはnilを返す (DBアクセスを行わない)", func(t *testing.T) {
		t.Parallel()
		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		repo := NewAttachmentRepository(q)

		attachment, err := repo.FindPubliclyReferencedBlobByID(context.Background(), model.AttachmentID("not-a-uuid"))
		if err != nil {
			t.Fatalf("FindPubliclyReferencedBlobByID()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindPubliclyReferencedBlobByID() = %v、期待値 = nil", attachment)
		}
	})
}

func TestAttachmentRepository_FindByIDsAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-findids@example.com").
		WithAtname("attachfindids").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("attach-findids-space").
		Build()

	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()

	attachmentID1 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("file1.png").Build()
	attachmentID2 := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).WithFilename("file2.jpg").Build()

	t.Run("IDリストに含まれる添付ファイルを一括取得できる", func(t *testing.T) {
		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{attachmentID1, attachmentID2}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace()のエラー = %v", err)
		}
		if len(attachments) != 2 {
			t.Fatalf("len(attachments) = %v、期待値 = 2", len(attachments))
		}
	})

	t.Run("空のIDリストは空のスライスを返す", func(t *testing.T) {
		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{}, spaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace()のエラー = %v", err)
		}
		if len(attachments) != 0 {
			t.Errorf("len(attachments) = %v、期待値 = 0", len(attachments))
		}
	})

	t.Run("異なるスペースIDの添付ファイルは取得されない", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-findids-other").
			Build()

		attachments, err := repo.FindByIDsAndSpace(context.Background(), []model.AttachmentID{attachmentID1, attachmentID2}, otherSpaceID)
		if err != nil {
			t.Fatalf("FindByIDsAndSpace()のエラー = %v", err)
		}
		if len(attachments) != 0 {
			t.Errorf("len(attachments) = %v、期待値 = 0", len(attachments))
		}
	})
}

func TestAttachmentRepository_ListByPageIDsAndSpace(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)
	refRepo := NewPageAttachmentReferenceRepository(q)
	ctx := context.Background()

	spaceID, spaceMemberID := testutil.SetupSpaceWithMember(t, tx, "attach-list-by-pages")
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("メモ").Build()

	referencedPageID := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("参照あり").Build()
	otherPageID := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("参照なし").Build()

	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithFilename("図.png").
		WithContentType("image/png").
		Build()

	if _, err := refRepo.CreateBatch(ctx, referencedPageID, spaceID, []model.AttachmentID{attachmentID}); err != nil {
		t.Fatalf("CreateBatch()のエラー = %v", err)
	}

	t.Run("参照元のページと添付ファイルの組を返す", func(t *testing.T) {
		pageAttachments, err := repo.ListByPageIDsAndSpace(ctx, []model.PageID{referencedPageID, otherPageID}, spaceID)
		if err != nil {
			t.Fatalf("ListByPageIDsAndSpace()のエラー = %v", err)
		}
		if len(pageAttachments) != 1 {
			t.Fatalf("ListByPageIDsAndSpace()の件数 = %d、期待値 = 1", len(pageAttachments))
		}
		if pageAttachments[0].PageID != referencedPageID {
			t.Errorf("PageID = %q、期待値 = %q", pageAttachments[0].PageID, referencedPageID)
		}
		if pageAttachments[0].Attachment.Filename != "図.png" {
			t.Errorf("Filename = %q、期待値 = %q", pageAttachments[0].Attachment.Filename, "図.png")
		}
		if pageAttachments[0].Attachment.BlobKey == "" {
			t.Error("BlobKeyが空です")
		}
	})

	t.Run("ページIDが空ならクエリを投げない", func(t *testing.T) {
		pageAttachments, err := repo.ListByPageIDsAndSpace(ctx, nil, spaceID)
		if err != nil {
			t.Fatalf("ListByPageIDsAndSpace()のエラー = %v", err)
		}
		if pageAttachments != nil {
			t.Errorf("ListByPageIDsAndSpace() = %v、期待値 = nil", pageAttachments)
		}
	})

	t.Run("異なるスペースIDの添付ファイルは取得されない", func(t *testing.T) {
		otherSpaceID := testutil.NewSpaceBuilder(t, tx).
			WithIdentifier("attach-list-by-pages-other").
			Build()

		pageAttachments, err := repo.ListByPageIDsAndSpace(ctx, []model.PageID{referencedPageID}, otherSpaceID)
		if err != nil {
			t.Fatalf("ListByPageIDsAndSpace()のエラー = %v", err)
		}
		if len(pageAttachments) != 0 {
			t.Errorf("ListByPageIDsAndSpace()の件数 = %d、期待値 = 0", len(pageAttachments))
		}
	})
}

func TestAttachmentRepository_Processing(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewAttachmentRepository(q)
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-processing@example.com").
		WithAtname("attachprocessing").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("attach-processing").Build()
	otherSpaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("attach-processing-other").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()

	attachedAt := time.Date(2026, 11, 15, 1, 2, 3, 0, time.UTC)
	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithContentType("image/jpeg").
		WithByteSize(2048).
		WithBlobKey("attach-processing-key").
		WithAttachedAt(attachedAt).
		Build()
	otherAttachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(spaceMemberID).
		WithBlobKey("attach-processing-other-key").
		Build()

	t.Run("処理に使う項目を取得できる", func(t *testing.T) {
		attachment, err := repo.FindForProcessingByIDAndSpace(ctx, attachmentID, spaceID)
		if err != nil {
			t.Fatalf("FindForProcessingByIDAndSpace()のエラー = %v", err)
		}
		if attachment == nil {
			t.Fatal("FindForProcessingByIDAndSpace()がnilを返した、期待値 = 添付ファイル")
		}
		want := model.Attachment{
			ID:               attachmentID,
			SpaceID:          spaceID,
			BlobKey:          "attach-processing-key",
			ContentType:      "image/jpeg",
			ByteSize:         2048,
			ProcessingStatus: model.AttachmentProcessingStatusPending,
			AttachedAt:       attachedAt,
		}
		if !attachment.AttachedAt.Equal(want.AttachedAt) {
			t.Errorf("AttachedAt = %v、期待値 = %v", attachment.AttachedAt, want.AttachedAt)
		}
		attachment.AttachedAt = want.AttachedAt
		if *attachment != want {
			t.Errorf("FindForProcessingByIDAndSpace() = %+v、期待値 = %+v", *attachment, want)
		}
	})

	t.Run("ほかのスペースを指定するとnilを返す", func(t *testing.T) {
		attachment, err := repo.FindForProcessingByIDAndSpace(ctx, attachmentID, otherSpaceID)
		if err != nil {
			t.Fatalf("FindForProcessingByIDAndSpace()のエラー = %v", err)
		}
		if attachment != nil {
			t.Errorf("FindForProcessingByIDAndSpace() = %+v、期待値 = nil", attachment)
		}
	})

	t.Run("今の状態がfromのときだけ状態を変える", func(t *testing.T) {
		now := time.Now()
		changed, err := repo.UpdateProcessingStatus(ctx, attachmentID, spaceID, model.AttachmentProcessingStatusCompleted, model.AttachmentProcessingStatusFailed, now)
		if err != nil {
			t.Fatalf("UpdateProcessingStatus()のエラー = %v", err)
		}
		if changed {
			t.Error("状態が違うのに変えた")
		}

		changed, err = repo.UpdateProcessingStatus(ctx, attachmentID, otherSpaceID, model.AttachmentProcessingStatusPending, model.AttachmentProcessingStatusProcessing, now)
		if err != nil {
			t.Fatalf("UpdateProcessingStatus()のエラー = %v", err)
		}
		if changed {
			t.Error("ほかのスペースを指定したのに変えた")
		}

		changed, err = repo.UpdateProcessingStatus(ctx, attachmentID, spaceID, model.AttachmentProcessingStatusPending, model.AttachmentProcessingStatusProcessing, now)
		if err != nil {
			t.Fatalf("UpdateProcessingStatus()のエラー = %v", err)
		}
		if !changed {
			t.Error("状態を変えなかった")
		}

		for id, want := range map[model.AttachmentID]model.AttachmentProcessingStatus{
			attachmentID:      model.AttachmentProcessingStatusProcessing,
			otherAttachmentID: model.AttachmentProcessingStatusPending,
		} {
			got, err := repo.FindForProcessingByIDAndSpace(ctx, id, spaceID)
			if err != nil {
				t.Fatalf("FindForProcessingByIDAndSpace()のエラー = %v", err)
			}
			if got.ProcessingStatus != want {
				t.Errorf("%sの状態 = %d、期待値 = %d", id, got.ProcessingStatus, want)
			}
		}
	})

	t.Run("blobのバイト数とチェックサムを、指定したスペースの添付ファイルだけ更新する", func(t *testing.T) {
		if err := repo.UpdateBlobContent(ctx, attachmentID, otherSpaceID, 1, "other"); err != nil {
			t.Fatalf("UpdateBlobContent()のエラー = %v", err)
		}
		if err := repo.UpdateBlobContent(ctx, attachmentID, spaceID, 1000, "CY9rzUYh03PK3k6DJie09g=="); err != nil {
			t.Fatalf("UpdateBlobContent()のエラー = %v", err)
		}

		rows, err := tx.QueryContext(ctx, `SELECT a.id, asb.byte_size, COALESCE(asb.checksum, '') FROM attachments a
			INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
			INNER JOIN active_storage_blobs asb ON asa.blob_id = asb.id
			WHERE a.id = ANY($1::uuid[])`, "{"+string(attachmentID)+","+string(otherAttachmentID)+"}")
		if err != nil {
			t.Fatalf("blobの取得に失敗: %v", err)
		}
		defer func() { _ = rows.Close() }()
		got := map[string]string{}
		for rows.Next() {
			var id, checksum string
			var byteSize int64
			if err := rows.Scan(&id, &byteSize, &checksum); err != nil {
				t.Fatalf("blobの読み取りに失敗: %v", err)
			}
			got[id] = fmt.Sprintf("%d %s", byteSize, checksum)
		}
		want := map[string]string{
			string(attachmentID):      "1000 CY9rzUYh03PK3k6DJie09g==",
			string(otherAttachmentID): "1024 ",
		}
		for id, w := range want {
			if got[id] != w {
				t.Errorf("%sのblob = %q、期待値 = %q", id, got[id], w)
			}
		}
	})
}

func TestAttachmentRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewAttachmentRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("attach-create@example.com").
		WithAtname("attachcreate").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("attach-create").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()

	// UTCでない時刻を渡しても、UTCの時刻で書く
	attachedAt := time.Date(2026, 11, 15, 10, 2, 3, 456000000, time.FixedZone("JST", 9*60*60))
	id, err := repo.Create(ctx, CreateAttachmentInput{
		SpaceID:       spaceID,
		UserID:        userID,
		SpaceMemberID: spaceMemberID,
		BlobKey:       "attachcreatekey0000000000000",
		Filename:      "写真.png",
		ContentType:   "image/png",
		ByteSize:      2048,
		Checksum:      "1B2M2Y8AsgTpgAmY7PhCfg==",
		AttachedAt:    attachedAt,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}

	var (
		key, filename, contentType, metadata, serviceName, checksum string
		byteSize                                                    int64
		blobCreatedAt                                               time.Time
		name, recordType, recordID                                  string
		storageCreatedAt                                            time.Time
		attachmentSpaceID, attachedSpaceMemberID                    string
		gotAttachedAt, createdAt, updatedAt                         time.Time
		processingStatus                                            int32
	)
	err = tx.QueryRowContext(ctx, `
		SELECT b.key, b.filename, b.content_type, b.metadata, b.service_name, b.byte_size, b.checksum, b.created_at,
		       asa.name, asa.record_type, asa.record_id, asa.created_at,
		       a.space_id, a.attached_space_member_id, a.attached_at, a.created_at, a.updated_at, a.processing_status
		FROM attachments a
		INNER JOIN active_storage_attachments asa ON asa.id = a.active_storage_attachment_id
		INNER JOIN active_storage_blobs b ON b.id = asa.blob_id
		WHERE a.id = $1 AND a.space_id = $2`, string(id), string(spaceID),
	).Scan(&key, &filename, &contentType, &metadata, &serviceName, &byteSize, &checksum, &blobCreatedAt,
		&name, &recordType, &recordID, &storageCreatedAt,
		&attachmentSpaceID, &attachedSpaceMemberID, &gotAttachedAt, &createdAt, &updatedAt, &processingStatus)
	if err != nil {
		t.Fatalf("作った行の取得に失敗: %v", err)
	}

	wantMetadata := `{"space_id":"` + string(spaceID) + `","user_id":"` + string(userID) + `"}`
	for _, c := range []struct{ field, got, want string }{
		{"key", key, "attachcreatekey0000000000000"},
		{"filename", filename, "写真.png"},
		{"content_type", contentType, "image/png"},
		{"metadata", metadata, wantMetadata},
		{"service_name", serviceName, "cloudflare_r2"},
		{"checksum", checksum, "1B2M2Y8AsgTpgAmY7PhCfg=="},
		{"name", name, "file"},
		{"record_type", recordType, "SpaceRecord"},
		{"record_id", recordID, string(spaceID)},
		{"space_id", attachmentSpaceID, string(spaceID)},
		{"attached_space_member_id", attachedSpaceMemberID, string(spaceMemberID)},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q、期待値 = %q", c.field, c.got, c.want)
		}
	}
	if byteSize != 2048 {
		t.Errorf("byte_size = %d、期待値 = 2048", byteSize)
	}
	if processingStatus != int32(model.AttachmentProcessingStatusPending) {
		t.Errorf("processing_status = %d、期待値 = %d", processingStatus, model.AttachmentProcessingStatusPending)
	}
	// タイムゾーンを持たない列は、書いたUTCの壁時計の時刻をそのまま返す
	wantTime := attachedAt.UTC()
	for field, got := range map[string]time.Time{
		"active_storage_blobs.created_at":       blobCreatedAt,
		"active_storage_attachments.created_at": storageCreatedAt,
		"attachments.attached_at":               gotAttachedAt,
		"attachments.created_at":                createdAt,
		"attachments.updated_at":                updatedAt,
	} {
		if !got.Equal(wantTime) {
			t.Errorf("%s = %v、期待値 = %v", field, got, wantTime)
		}
	}
}

// attachmentDeletionFixtureは、添付ファイルの実体のキーと行の削除のテストが使う2つのスペース
type attachmentDeletionFixture struct {
	repo          *AttachmentRepository
	spaceID       model.SpaceID
	otherSpaceID  model.SpaceID
	memberID      model.SpaceMemberID
	otherMemberID model.SpaceMemberID
}

func setupAttachmentDeletionFixture(t *testing.T, suffix string) (*attachmentDeletionFixture, *sql.Tx) {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	f := &attachmentDeletionFixture{repo: NewAttachmentRepository(testutil.QueriesWithTx(tx))}
	for i, target := range []struct {
		spaceID  *model.SpaceID
		memberID *model.SpaceMemberID
	}{{&f.spaceID, &f.memberID}, {&f.otherSpaceID, &f.otherMemberID}} {
		name := fmt.Sprintf("attach-del-%s-%d", suffix, i)
		userID := testutil.NewUserBuilder(t, tx).WithEmail(name + "@example.com").WithAtname(fmt.Sprintf("attach_del_%s_%d", suffix, i)).Build()
		*target.spaceID = testutil.NewSpaceBuilder(t, tx).WithIdentifier(name).Build()
		*target.memberID = testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(*target.spaceID).WithUserID(userID).Build()
	}
	return f, tx
}

// addAttachmentVariantは、Rails版が作るのと同じ形のサムネイル (variant) の行を、variantのblobのキーをkeyにして作る
func addAttachmentVariant(t *testing.T, tx *sql.Tx, id model.AttachmentID, key string) {
	t.Helper()

	ctx := context.Background()
	var variantRecordID, variantBlobID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO active_storage_variant_records (blob_id, variation_digest)
		SELECT asa.blob_id, 'digest' FROM attachments a
		INNER JOIN active_storage_attachments asa ON a.active_storage_attachment_id = asa.id
		WHERE a.id = $1
		RETURNING id`, string(id)).Scan(&variantRecordID); err != nil {
		t.Fatalf("variantの行の作成に失敗: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO active_storage_blobs (key, filename, content_type, metadata, service_name, byte_size, checksum, created_at)
		VALUES ($1, 'thumb.png', 'image/png', '{}', 'test', 1, 'x', NOW()) RETURNING id`, key).Scan(&variantBlobID); err != nil {
		t.Fatalf("variantのblobの作成に失敗: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO active_storage_attachments (name, record_type, record_id, blob_id, created_at)
		VALUES ('image', 'ActiveStorage::VariantRecord', $1, $2, NOW())`, variantRecordID, variantBlobID); err != nil {
		t.Fatalf("variantのattachmentの作成に失敗: %v", err)
	}
}

func TestAttachmentRepository_ListObjectKeysByIDsAndSpace(t *testing.T) {
	t.Parallel()

	f, tx := setupAttachmentDeletionFixture(t, "keys")
	id := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey("attach-del-keys-target").Build()
	addAttachmentVariant(t, tx, id, "attach-del-keys-target-variant")
	plainID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey("attach-del-keys-plain").Build()
	otherID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.otherSpaceID).WithSpaceMemberID(f.otherMemberID).WithBlobKey("attach-del-keys-other").Build()

	keys, err := f.repo.ListObjectKeysByIDsAndSpace(context.Background(), []model.AttachmentID{id, plainID, otherID, "not-a-uuid"}, f.spaceID)
	if err != nil {
		t.Fatalf("ListObjectKeysByIDsAndSpace()のエラー = %v", err)
	}
	slices.Sort(keys)
	want := []string{"attach-del-keys-plain", "attach-del-keys-target", "attach-del-keys-target-variant"}
	if !slices.Equal(keys, want) {
		t.Errorf("keys = %v、期待値 = %v (ほかのスペースのキーを含まず、variantのキーを含む)", keys, want)
	}

	empty, err := f.repo.ListObjectKeysByIDsAndSpace(context.Background(), nil, f.spaceID)
	if err != nil || len(empty) != 0 {
		t.Errorf("IDが無いとき: keys = %v、エラー = %v", empty, err)
	}
}

func TestAttachmentRepository_DeleteByIDsAndSpace(t *testing.T) {
	t.Parallel()

	f, tx := setupAttachmentDeletionFixture(t, "rows")
	ctx := context.Background()
	id := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey("attach-del-rows-target").Build()
	addAttachmentVariant(t, tx, id, "attach-del-rows-target-variant")
	keepID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey("attach-del-rows-keep").Build()
	otherID := testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.otherSpaceID).WithSpaceMemberID(f.otherMemberID).WithBlobKey("attach-del-rows-other").Build()

	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(f.spaceID).Build()
	pageID := testutil.NewPageBuilder(t, tx).WithSpaceID(f.spaceID).WithTopicID(topicID).WithFeaturedImageAttachmentID(id).Build()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO page_attachment_references (attachment_id, page_id, created_at, updated_at)
		VALUES ($1, $3, NOW(), NOW()), ($2, $3, NOW(), NOW())`, string(id), string(keepID), string(pageID)); err != nil {
		t.Fatalf("ページからの参照の作成に失敗: %v", err)
	}

	// ほかのスペースを指定しても、そのスペースに無い添付ファイルは消さない
	if _, err := f.repo.DeleteByIDsAndSpace(ctx, []model.AttachmentID{otherID}, f.spaceID); err != nil {
		t.Fatalf("DeleteByIDsAndSpace()のエラー = %v", err)
	}
	if _, err := f.repo.DeleteByIDsAndSpace(ctx, []model.AttachmentID{id, "not-a-uuid"}, f.spaceID); err != nil {
		t.Fatalf("DeleteByIDsAndSpace()のエラー = %v", err)
	}

	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := tx.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			t.Fatalf("行の数の取得に失敗: %v", err)
		}
		return n
	}
	for _, tt := range []struct {
		name string
		got  int
		want int
	}{
		{name: "消した添付ファイル", got: count(`SELECT count(*) FROM attachments WHERE id = $1`, string(id)), want: 0},
		{name: "消した添付ファイルへの参照", got: count(`SELECT count(*) FROM page_attachment_references WHERE attachment_id = $1`, string(id)), want: 0},
		{name: "消した添付ファイルとvariantのblob", got: count(`SELECT count(*) FROM active_storage_blobs WHERE key LIKE 'attach-del-rows-target%'`), want: 0},
		{name: "variantの行", got: count(`SELECT count(*) FROM active_storage_variant_records WHERE variation_digest = 'digest'`), want: 0},
		{name: "残す添付ファイル", got: count(`SELECT count(*) FROM attachments WHERE id IN ($1, $2)`, string(keepID), string(otherID)), want: 2},
		{name: "残す添付ファイルへの参照", got: count(`SELECT count(*) FROM page_attachment_references WHERE attachment_id = $1`, string(keepID)), want: 1},
		{name: "残す添付ファイルのblob", got: count(`SELECT count(*) FROM active_storage_blobs WHERE key IN ('attach-del-rows-keep', 'attach-del-rows-other')`), want: 2},
	} {
		if tt.got != tt.want {
			t.Errorf("%sの行の数 = %d、期待値 = %d", tt.name, tt.got, tt.want)
		}
	}

	var featured sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT featured_image_attachment_id FROM pages WHERE id = $1 AND space_id = $2`, string(pageID), string(f.spaceID)).Scan(&featured); err != nil {
		t.Fatalf("ページのカバー画像の取得に失敗: %v", err)
	}
	if featured.Valid {
		t.Errorf("ページのカバー画像 = %q、NULLを期待", featured.String)
	}
}

func TestAttachmentRepository_ListBySpacePaginated(t *testing.T) {
	t.Parallel()

	f, tx := setupAttachmentDeletionFixture(t, "list")
	ctx := context.Background()
	base := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	build := func(filename string, attachedAt time.Time) model.AttachmentID {
		return testutil.NewAttachmentBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithSpaceMemberID(f.memberID).
			WithFilename(filename).
			WithContentType("image/png").
			WithByteSize(123).
			WithAttachedAt(attachedAt).
			Build()
	}
	oldest := build("Report_2026.PNG", base)
	middle := build("100%-photo.png", base.Add(time.Hour))
	tieA := build("photo-a.png", base.Add(2*time.Hour))
	tieB := build("photo-b.png", base.Add(2*time.Hour))
	testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(f.otherSpaceID).
		WithSpaceMemberID(f.otherMemberID).
		WithFilename("photo-other.png").
		Build()

	// 同じ時刻の2件は、IDの降順に並ぶ
	newer, older := tieA, tieB
	if string(tieB) > string(tieA) {
		newer, older = tieB, tieA
	}

	ids := func(result *PaginatedAttachments) []model.AttachmentID {
		got := make([]model.AttachmentID, len(result.Attachments))
		for i, a := range result.Attachments {
			got[i] = a.ID
		}
		return got
	}

	tests := []struct {
		name      string
		query     string
		page      int32
		limit     int32
		wantIDs   []model.AttachmentID
		wantTotal int64
	}{
		{name: "ほかのスペースを除いて新しい順に並ぶ", query: "", page: 1, limit: 10, wantIDs: []model.AttachmentID{newer, older, middle, oldest}, wantTotal: 4},
		{name: "2ページ目は続きを返す", query: "", page: 2, limit: 3, wantIDs: []model.AttachmentID{oldest}, wantTotal: 4},
		{name: "大文字と小文字を区別せずに部分一致で絞り込む", query: "report", page: 1, limit: 10, wantIDs: []model.AttachmentID{oldest}, wantTotal: 1},
		{name: "検索語の%を文字として扱う", query: "100%", page: 1, limit: 10, wantIDs: []model.AttachmentID{middle}, wantTotal: 1},
		{name: "検索語の_を文字として扱う", query: "t_2", page: 1, limit: 10, wantIDs: []model.AttachmentID{oldest}, wantTotal: 1},
		{name: "_が任意の1文字に一致しない", query: "photo_a", page: 1, limit: 10, wantIDs: []model.AttachmentID{}, wantTotal: 0},
		{name: "空白だけの検索語は絞り込まない", query: "  ", page: 1, limit: 10, wantIDs: []model.AttachmentID{newer, older, middle, oldest}, wantTotal: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.repo.ListBySpacePaginated(ctx, f.spaceID, tt.query, tt.page, tt.limit)
			if err != nil {
				t.Fatalf("ListBySpacePaginated()のエラー = %v", err)
			}
			if got := ids(result); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ID = %v、期待値 = %v", got, tt.wantIDs)
			}
			if result.TotalCount != tt.wantTotal {
				t.Errorf("TotalCount = %d、期待値 = %d", result.TotalCount, tt.wantTotal)
			}
		})
	}

	t.Run("一覧に出す項目を設定する", func(t *testing.T) {
		result, err := f.repo.ListBySpacePaginated(ctx, f.spaceID, "Report", 1, 10)
		if err != nil {
			t.Fatalf("ListBySpacePaginated()のエラー = %v", err)
		}
		if len(result.Attachments) != 1 {
			t.Fatalf("件数 = %d、期待値 = 1", len(result.Attachments))
		}
		got := result.Attachments[0]
		if got.SpaceID != f.spaceID || got.Filename != "Report_2026.PNG" || got.ContentType != "image/png" || got.ByteSize != 123 || got.BlobKey == "" {
			t.Errorf("添付ファイル = %+v", got)
		}
		if !got.AttachedAt.Equal(base) {
			t.Errorf("AttachedAt = %v、期待値 = %v", got.AttachedAt, base)
		}
	})
}

func TestAttachmentRepository_ListIDsBySpaceAfter(t *testing.T) {
	t.Parallel()

	f, tx := setupAttachmentDeletionFixture(t, "list-after")
	ctx := context.Background()
	var want []model.AttachmentID
	for i := range 3 {
		want = append(want, testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.spaceID).WithSpaceMemberID(f.memberID).WithBlobKey(fmt.Sprintf("attach-list-after-%d", i)).Build())
	}
	testutil.NewAttachmentBuilder(t, tx).WithSpaceID(f.otherSpaceID).WithSpaceMemberID(f.otherMemberID).WithBlobKey("attach-list-after-other").Build()
	slices.Sort(want)

	first, err := f.repo.ListIDsBySpaceAfter(ctx, f.spaceID, "", 2)
	if err != nil {
		t.Fatalf("ListIDsBySpaceAfter()のエラー = %v", err)
	}
	if !slices.Equal(first, want[:2]) {
		t.Errorf("最初の2件 = %v、期待値 = %v", first, want[:2])
	}

	rest, err := f.repo.ListIDsBySpaceAfter(ctx, f.spaceID, first[len(first)-1], 2)
	if err != nil {
		t.Fatalf("ListIDsBySpaceAfter()のエラー = %v", err)
	}
	if !slices.Equal(rest, want[2:]) {
		t.Errorf("残りの件 = %v、期待値 = %v (ほかのスペースの添付ファイルを含まない)", rest, want[2:])
	}
}
