package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
)

// OAuthAuthorizationCodeRepositoryはOAuthの認可コードリポジトリ
type OAuthAuthorizationCodeRepository struct {
	q *query.Queries
}

// NewOAuthAuthorizationCodeRepositoryはOAuthAuthorizationCodeRepositoryを生成する
func NewOAuthAuthorizationCodeRepository(q *query.Queries) *OAuthAuthorizationCodeRepository {
	return &OAuthAuthorizationCodeRepository{q: q}
}

// WithTxはトランザクションを使用する新しいRepositoryを返す
func (r *OAuthAuthorizationCodeRepository) WithTx(tx *sql.Tx) *OAuthAuthorizationCodeRepository {
	return &OAuthAuthorizationCodeRepository{q: r.q.WithTx(tx)}
}

// CreateOAuthAuthorizationCodeInputは認可コード作成の入力パラメータ
type CreateOAuthAuthorizationCodeInput struct {
	OAuthGrantID  model.OAuthGrantID
	SpaceID       model.SpaceID
	CodeDigest    string
	Scopes        []model.Scope
	RedirectURI   string
	CodeChallenge string
	ExpiresAt     time.Time
}

// Createは認可コードを作成する
func (r *OAuthAuthorizationCodeRepository) Create(ctx context.Context, input CreateOAuthAuthorizationCodeInput) (*model.OAuthAuthorizationCode, error) {
	row, err := r.q.CreateOAuthAuthorizationCode(ctx, query.CreateOAuthAuthorizationCodeParams{
		OauthGrantID:  string(input.OAuthGrantID),
		SpaceID:       string(input.SpaceID),
		CodeDigest:    input.CodeDigest,
		Scopes:        model.ScopesToStrings(input.Scopes),
		RedirectUri:   input.RedirectURI,
		CodeChallenge: input.CodeChallenge,
		ExpiresAt:     input.ExpiresAt,
		Now:           time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateOAuthAuthorizationCodeWithGrantInputは、許可の作成または拡張と認可コードの作成の
// 入力パラメータ。Scopesは今回の同意で認めたスコープで、コードはこの範囲だけを持つ
type CreateOAuthAuthorizationCodeWithGrantInput struct {
	OAuthApplicationID model.OAuthApplicationID
	SpaceID            model.SpaceID
	SpaceMemberID      model.SpaceMemberID
	CodeDigest         string
	Scopes             []model.Scope
	RedirectURI        string
	CodeChallenge      string
	ExpiresAt          time.Time
}

// CreateWithGrantは、スペースメンバーのOAuthアプリへの許可を作成または拡張し、その許可に
// 属する認可コードを作成する。失効していない許可が既にあれば、許可のスコープを今回のスコープ
// との和に広げる。許可とコードを1つの文で作るため、同じアプリへの同時の同意でも許可は1つに
// 保たれ、許可だけが作られてコードが無い状態も残らない。
func (r *OAuthAuthorizationCodeRepository) CreateWithGrant(ctx context.Context, input CreateOAuthAuthorizationCodeWithGrantInput) (*model.OAuthAuthorizationCode, error) {
	row, err := r.q.CreateOAuthAuthorizationCodeWithGrant(ctx, query.CreateOAuthAuthorizationCodeWithGrantParams{
		OauthApplicationID: string(input.OAuthApplicationID),
		SpaceID:            string(input.SpaceID),
		SpaceMemberID:      string(input.SpaceMemberID),
		CodeDigest:         input.CodeDigest,
		Scopes:             model.ScopesToStrings(input.Scopes),
		RedirectUri:        input.RedirectURI,
		CodeChallenge:      input.CodeChallenge,
		ExpiresAt:          input.ExpiresAt,
		Now:                time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByCodeDigestはコードダイジェストで認可コードを取得する。見つからない場合は
// (nil, nil) を返す。使用済みや期限切れのコードも返すため、有効かどうかは呼び出し側が
// 判定する。
//
// トークン要求の入口で使うため、スペースを条件に含めない。呼び出し側は、返ったコードの
// SpaceIDを以降の取得の条件に必ず含める。
func (r *OAuthAuthorizationCodeRepository) FindByCodeDigest(ctx context.Context, codeDigest string) (*model.OAuthAuthorizationCode, error) {
	row, err := r.q.FindOAuthAuthorizationCodeByCodeDigest(ctx, codeDigest)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// ExchangeOAuthAuthorizationCodeInputは認可コードの交換の入力パラメータ。
// アクセストークンとリフレッシュトークンは、コードの許可・スコープで作る
type ExchangeOAuthAuthorizationCodeInput struct {
	ID                    model.OAuthAuthorizationCodeID
	SpaceID               model.SpaceID
	AccessTokenDigest     string
	AccessTokenExpiresAt  time.Time
	RefreshTokenDigest    string
	RefreshTokenExpiresAt time.Time
	Now                   time.Time
}

// Exchangeは未使用の認可コードを使用済みにし、コードの許可・スコープでアクセストークンと
// リフレッシュトークンを作成して、使用済みにしたコードを返す。判定と更新を1つの文で行うため、
// 同じコードの同時の交換でもトークンを作るのは1回だけである。見つからないか既に使用済みの
// 場合は何もせず (nil, nil) を返す。有効期限・リダイレクトURI・PKCEは呼び出し側が交換の前に
// 確かめる。
func (r *OAuthAuthorizationCodeRepository) Exchange(ctx context.Context, input ExchangeOAuthAuthorizationCodeInput) (*model.OAuthAuthorizationCode, error) {
	row, err := r.q.ExchangeOAuthAuthorizationCode(ctx, query.ExchangeOAuthAuthorizationCodeParams{
		Now:                   input.Now,
		ID:                    string(input.ID),
		SpaceID:               string(input.SpaceID),
		AccessTokenDigest:     input.AccessTokenDigest,
		AccessTokenExpiresAt:  input.AccessTokenExpiresAt,
		RefreshTokenDigest:    input.RefreshTokenDigest,
		RefreshTokenExpiresAt: input.RefreshTokenExpiresAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(query.OauthAuthorizationCode(row)), nil
}

// toModelはquery.OauthAuthorizationCodeをmodel.OAuthAuthorizationCodeに変換する
func (r *OAuthAuthorizationCodeRepository) toModel(row query.OauthAuthorizationCode) *model.OAuthAuthorizationCode {
	var usedAt *time.Time
	if row.UsedAt.Valid {
		usedAt = &row.UsedAt.Time
	}
	return &model.OAuthAuthorizationCode{
		ID:            model.OAuthAuthorizationCodeID(row.ID),
		OAuthGrantID:  model.OAuthGrantID(row.OauthGrantID),
		SpaceID:       model.SpaceID(row.SpaceID),
		CodeDigest:    row.CodeDigest,
		Scopes:        model.StringsToScopes(row.Scopes),
		RedirectURI:   row.RedirectUri,
		CodeChallenge: row.CodeChallenge,
		ExpiresAt:     row.ExpiresAt,
		UsedAt:        usedAt,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
