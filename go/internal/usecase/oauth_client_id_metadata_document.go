package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wikinoapp/wikino/go/internal/cimd"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// findClientIDMetadataDocumentClientは、CIMDのクライアントID (HTTPSのURL) clientIDのOAuthアプリを返す。
// 文書のキャッシュが期限内ならその行を使い、無いか期限が切れていれば取得し直して作成・更新する。
//
// ユーザーuserIDにMCPサーバーのフィーチャーフラグが無効なら、文書を取得する前に*model.AppError
// (見つからない) を返す。許可リストの外のクライアントIDと、文書を取得・検証できない場合は、使える
// アプリが無いとしてnilを返す。取得の失敗はキャッシュせず、次の認可要求で取得し直す。
//
// 文書の取得 (外部へのHTTPの要求) と行の作成・更新はトランザクションの外で行い、作成・更新は単独の
// 書き込みにする。同意の永続化と結び付ける必要が無く、取得の間にトランザクションを開いたままにしないため
func findClientIDMetadataDocumentClient(
	ctx context.Context,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	fetcher *cimd.Fetcher,
	userID model.UserID,
	clientID string,
) (*model.OAuthApplication, error) {
	if err := checkFeatureFlagEnabled(ctx, featureFlagRepo, userID, model.FeatureFlagRemoteMCPServer); err != nil {
		return nil, err
	}
	if !cimd.IsAllowed(clientID) {
		return nil, nil
	}

	app, err := oauthApplicationRepo.FindByClientID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app != nil && !app.IsClientIDMetadataDocument() {
		// URLのクライアントIDは登録したクライアントに付けないが、CIMDの文書で登録したクライアントを
		// 書き換えないよう、念のため使わない
		return nil, nil
	}
	now := time.Now()
	if app != nil && app.IsMetadataFresh(now) {
		return app, nil
	}

	doc, err := fetcher.Fetch(ctx, clientID, now)
	if err != nil {
		// 取得の失敗はクライアント側の不具合や一時的な障害で、Wikinoの不具合ではないため警告に留める
		slog.WarnContext(ctx, "CIMDの文書を取得できませんでした", "client_id", clientID, "error", err)
		return nil, nil
	}

	app, err = oauthApplicationRepo.UpsertClientIDMetadataDocument(ctx, repository.UpsertClientIDMetadataDocumentInput{
		ClientID:          doc.ClientID,
		Name:              doc.ClientName,
		RedirectURIs:      doc.RedirectURIs,
		MetadataExpiresAt: doc.ExpiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("CIMDのクライアントの作成・更新に失敗: %w", err)
	}
	return app, nil
}
