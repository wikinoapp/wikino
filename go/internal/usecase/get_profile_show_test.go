package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newGetProfileShowUsecaseForTestはトランザクション内のリポジトリでユースケースを組み立てる
func newGetProfileShowUsecaseForTest(tx *sql.Tx) *GetProfileShowUsecase {
	queries := testutil.QueriesWithTx(tx)
	return NewGetProfileShowUsecase(
		repository.NewUserRepository(queries),
		repository.NewSpaceRepository(queries),
	)
}

// addProfileSpaceはスペースを作り、ユーザーをメンバーにしてスペースのIDを返す
func addProfileSpace(t *testing.T, tx *sql.Tx, userID model.UserID, identifier string, discarded, active bool) model.SpaceID {
	t.Helper()

	builder := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).WithName(identifier)
	if discarded {
		builder = builder.WithDiscarded()
	}
	spaceID := builder.Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		WithActive(active).
		Build()

	return spaceID
}

func TestGetProfileShowUsecase_Execute_参加中のスペースだけを返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-show-spaces@example.com").
		WithAtname("profile_show_spaces").
		Build()

	activeSpaceID := addProfileSpace(t, tx, userID, "profile-show-active", false, true)
	addProfileSpace(t, tx, userID, "profile-show-discarded", true, true)
	addProfileSpace(t, tx, userID, "profile-show-inactive", false, false)

	output, err := newGetProfileShowUsecaseForTest(tx).Execute(context.Background(), GetProfileShowInput{
		Atname: "profile_show_spaces",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output == nil {
		t.Fatal("output = nil、非nilを期待")
	}
	if output.User.ID != userID {
		t.Errorf("User.ID = %q、期待値 = %q", output.User.ID, userID)
	}
	if len(output.JoinedSpaces) != 1 {
		t.Fatalf("len(JoinedSpaces) = %d、期待値 = 1", len(output.JoinedSpaces))
	}
	if output.JoinedSpaces[0].ID != activeSpaceID {
		t.Errorf("JoinedSpaces[0].ID = %q、期待値 = %q", output.JoinedSpaces[0].ID, activeSpaceID)
	}
}

func TestGetProfileShowUsecase_Execute_本人のときだけ編集できる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ownerID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-show-owner@example.com").
		WithAtname("profile_show_owner").
		Build()
	otherID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-show-other@example.com").
		WithAtname("profile_show_other").
		Build()

	tests := []struct {
		name     string
		viewerID *model.UserID
		want     bool
	}{
		{name: "本人", viewerID: &ownerID, want: true},
		{name: "ほかのユーザー", viewerID: &otherID, want: false},
		{name: "ログインしていない閲覧者", viewerID: nil, want: false},
	}

	uc := newGetProfileShowUsecaseForTest(tx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetProfileShowInput{
				Atname:       "profile_show_owner",
				ViewerUserID: tt.viewerID,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output.CanEditProfile != tt.want {
				t.Errorf("CanEditProfile = %v、期待値 = %v", output.CanEditProfile, tt.want)
			}
		})
	}
}

func TestGetProfileShowUsecase_Execute_ユーザーが見つからなければnilを返す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	discardedID := testutil.NewUserBuilder(t, tx).
		WithEmail("profile-show-discarded@example.com").
		WithAtname("profile_show_discarded").
		Build()
	if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(discardedID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}

	tests := []struct {
		name   string
		atname string
	}{
		{name: "削除済みのユーザー", atname: "profile_show_discarded"},
		{name: "存在しないアットネーム", atname: "profile_show_missing"},
	}

	uc := newGetProfileShowUsecaseForTest(tx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := uc.Execute(context.Background(), GetProfileShowInput{Atname: tt.atname})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}
			if output != nil {
				t.Errorf("output = %+v、nilを期待", output)
			}
		})
	}
}
