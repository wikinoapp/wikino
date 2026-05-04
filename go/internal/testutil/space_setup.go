package testutil

import (
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// SetupSpaceWithMember はテスト用に user / space / spaceMember を一括作成するヘルパー。
//
// prefix はテスト内でユニークな識別子を作るために使用する (email / atname / space identifier
// に同じ値を埋め込み、t.Parallel() 同士の衝突を避ける)。複数のビルダーを束ねた一括セットアップ
// であり、Wikino のテストでは「最小単位の公開可能なリソース構成」を組むケースが多いため、
// 各テストパッケージで再実装せずにこのヘルパーを使用する。
func SetupSpaceWithMember(t *testing.T, tx *sql.Tx, prefix string) (model.SpaceID, model.SpaceMemberID) {
	t.Helper()
	userID := NewUserBuilder(t, tx).
		WithEmail(prefix + "@example.com").
		WithAtname(prefix).
		Build()
	spaceID := NewSpaceBuilder(t, tx).
		WithIdentifier(prefix).
		Build()
	spaceMemberID := NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	return spaceID, spaceMemberID
}
