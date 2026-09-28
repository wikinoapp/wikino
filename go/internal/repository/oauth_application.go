package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthApplicationRepositoryはOAuthアプリリポジトリ
type OAuthApplicationRepository struct {
	q *query.Queries
}

// NewOAuthApplicationRepositoryはOAuthApplicationRepositoryを生成する
func NewOAuthApplicationRepository(q *query.Queries) *OAuthApplicationRepository {
	return &OAuthApplicationRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthApplicationRepository) WithTx(tx *sql.Tx) *OAuthApplicationRepository {
	return &OAuthApplicationRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthApplicationInputはスペースのOAuthアプリ作成の入力パラメータ。publicクライアント
// ではClientSecretDigestをnilにする
type CreateOAuthApplicationInput struct {
	SpaceID              model.SpaceID
	CreatedSpaceMemberID model.SpaceMemberID
	Name                 string
	ClientID             string
	ClientSecretDigest   *string
	ClientType           model.OAuthClientType
	RedirectURIs         []string
}

// CreateはスペースのOAuthアプリを作成する。公式クライアントはこのメソッドでは作らない
func (r *OAuthApplicationRepository) Create(ctx context.Context, input CreateOAuthApplicationInput) (*model.OAuthApplication, error) {
	var clientSecretDigest sql.NullString
	if input.ClientSecretDigest != nil {
		clientSecretDigest = sql.NullString{String: *input.ClientSecretDigest, Valid: true}
	}

	row, err := r.q.CreateOAuthApplication(ctx, query.CreateOAuthApplicationParams{
		SpaceID:              string(input.SpaceID),
		CreatedSpaceMemberID: string(input.CreatedSpaceMemberID),
		Name:                 input.Name,
		ClientID:             input.ClientID,
		ClientSecretDigest:   clientSecretDigest,
		ClientType:           int32(input.ClientType),
		RedirectUris:         input.RedirectURIs,
		Now:                  time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByClientIDはクライアントIDで削除されていないOAuthアプリを取得する。見つからない
// 場合は (nil, nil) を返す。
//
// 認可要求の入口と、トークンの失効でクライアントを先に認証するときに使うため、スペースを
// 条件に含めない。認可要求では、返ったアプリのSpaceID (公式クライアントでは認可要求の
// resourceから決めたスペース) を以降の取得の条件に必ず含める。トークンの失効では、トークンを
// 取得した後でアプリがトークンのスペースで使えるか確かめる。スペースが先に分かっている場合は
// FindByClientIDAndSpaceIDを使う。
func (r *OAuthApplicationRepository) FindByClientID(ctx context.Context, clientID string) (*model.OAuthApplication, error) {
	row, err := r.q.FindOAuthApplicationByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByClientIDAndSpaceIDは、クライアントIDで、スペースで使える削除されていないOAuthアプリを
// 取得する。そのスペースのアプリか、公式クライアントを返す。見つからない場合は (nil, nil) を返す
func (r *OAuthApplicationRepository) FindByClientIDAndSpaceID(ctx context.Context, clientID string, spaceID model.SpaceID) (*model.OAuthApplication, error) {
	row, err := r.q.FindOAuthApplicationByClientIDAndSpaceID(ctx, query.FindOAuthApplicationByClientIDAndSpaceIDParams{
		ClientID: clientID,
		SpaceID:  string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByIDAndSpaceIDは、IDで、スペースの削除されていないOAuthアプリを取得する。見つからない
// 場合は (nil, nil) を返す。IDはURLからこのメソッドへ渡ってくるため、UUIDでない文字列は
// データベースへ送らず「見つからない」として答える。公式クライアントはスペースに属さないため
// 見つからない
func (r *OAuthApplicationRepository) FindByIDAndSpaceID(ctx context.Context, id model.OAuthApplicationID, spaceID model.SpaceID) (*model.OAuthApplication, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.FindOAuthApplicationByIDAndSpaceID(ctx, query.FindOAuthApplicationByIDAndSpaceIDParams{
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ListBySpaceはスペースの削除されていないOAuthアプリを新しい順に取得する。公式クライアントは含めない
func (r *OAuthApplicationRepository) ListBySpace(ctx context.Context, spaceID model.SpaceID) ([]*model.OAuthApplication, error) {
	rows, err := r.q.ListOAuthApplicationsBySpace(ctx, string(spaceID))
	if err != nil {
		return nil, err
	}
	apps := make([]*model.OAuthApplication, len(rows))
	for i, row := range rows {
		apps[i] = r.toModel(row)
	}
	return apps, nil
}

// ListAvailableInSpaceByIDsは、IDで、スペースで使える削除されていないOAuthアプリ (そのスペースの
// アプリか公式クライアント) を取得する。見つからないIDは結果に含めない。並び順は保証しない
func (r *OAuthApplicationRepository) ListAvailableInSpaceByIDs(ctx context.Context, ids []model.OAuthApplicationID, spaceID model.SpaceID) ([]*model.OAuthApplication, error) {
	if len(ids) == 0 {
		return []*model.OAuthApplication{}, nil
	}

	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = string(id)
	}
	rows, err := r.q.ListOAuthApplicationsAvailableInSpaceByIDs(ctx, query.ListOAuthApplicationsAvailableInSpaceByIDsParams{
		Ids:     idStrs,
		SpaceID: string(spaceID),
	})
	if err != nil {
		return nil, err
	}
	apps := make([]*model.OAuthApplication, len(rows))
	for i, row := range rows {
		apps[i] = r.toModel(row)
	}
	return apps, nil
}

// UpdateOAuthApplicationInputはスペースのOAuthアプリ更新の入力パラメータ
type UpdateOAuthApplicationInput struct {
	ID              model.OAuthApplicationID
	SpaceID         model.SpaceID
	ExpectedVersion int64
	Name            string
	RedirectURIs    []string
}

// Updateは版が一致する場合だけ、スペースのOAuthアプリの名前とリダイレクトURIを更新する。
// 対象が無いか版が異なる場合は (nil, nil) を返す。IDはURLから渡ってくるため、UUIDでない
// 文字列はデータベースへ送らず「見つからない」として答える
func (r *OAuthApplicationRepository) Update(ctx context.Context, input UpdateOAuthApplicationInput) (*model.OAuthApplication, error) {
	if !uuidRegex.MatchString(string(input.ID)) {
		return nil, nil
	}

	row, err := r.q.UpdateOAuthApplication(ctx, query.UpdateOAuthApplicationParams{
		Name:            input.Name,
		RedirectUris:    input.RedirectURIs,
		ExpectedVersion: input.ExpectedVersion,
		Now:             time.Now(),
		ID:              string(input.ID),
		SpaceID:         string(input.SpaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// UpdateClientSecretDigestは、スペースの削除されていないconfidentialクライアントのシークレットの
// ダイジェストを置き換える。対象が無いか、publicクライアントの場合は (nil, nil) を返す
func (r *OAuthApplicationRepository) UpdateClientSecretDigest(ctx context.Context, id model.OAuthApplicationID, spaceID model.SpaceID, clientSecretDigest string) (*model.OAuthApplication, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.UpdateOAuthApplicationClientSecretDigest(ctx, query.UpdateOAuthApplicationClientSecretDigestParams{
		ClientSecretDigest: clientSecretDigest,
		Now:                time.Now(),
		ID:                 string(id),
		SpaceID:            string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Discardは、スペースの削除されていないOAuthアプリを削除し、そのアプリの許可と、許可に属する
// アクセストークン・リフレッシュトークンをすべて失効する。1文で行うため、アプリだけが削除
// された状態は途中にも残らない。対象が無いか既に削除されている場合は何も変えずに (nil, nil) を返す
func (r *OAuthApplicationRepository) Discard(ctx context.Context, id model.OAuthApplicationID, spaceID model.SpaceID) (*model.OAuthApplication, error) {
	if !uuidRegex.MatchString(string(id)) {
		return nil, nil
	}

	row, err := r.q.DiscardOAuthApplication(ctx, query.DiscardOAuthApplicationParams{
		Now:     time.Now(),
		ID:      string(id),
		SpaceID: string(spaceID),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(query.OauthApplication(row)), nil
}

// toModelはquery.OauthApplicationをmodel.OAuthApplicationに変換する
func (r *OAuthApplicationRepository) toModel(row query.OauthApplication) *model.OAuthApplication {
	var spaceID *model.SpaceID
	if row.SpaceID != nil {
		id := model.SpaceID(*row.SpaceID)
		spaceID = &id
	}
	var createdSpaceMemberID *model.SpaceMemberID
	if row.CreatedSpaceMemberID != nil {
		id := model.SpaceMemberID(*row.CreatedSpaceMemberID)
		createdSpaceMemberID = &id
	}
	var clientSecretDigest *string
	if row.ClientSecretDigest.Valid {
		clientSecretDigest = &row.ClientSecretDigest.String
	}
	var discardedAt *time.Time
	if row.DiscardedAt.Valid {
		discardedAt = &row.DiscardedAt.Time
	}
	return &model.OAuthApplication{
		ID:                   model.OAuthApplicationID(row.ID),
		Version:              row.Version,
		SpaceID:              spaceID,
		CreatedSpaceMemberID: createdSpaceMemberID,
		Name:                 row.Name,
		ClientID:             row.ClientID,
		ClientSecretDigest:   clientSecretDigest,
		ClientType:           model.OAuthClientType(row.ClientType),
		RedirectURIs:         row.RedirectUris,
		DiscardedAt:          discardedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}
