package model

// OAuthAudienceはOAuthのトークンの宛先。認可コードに記録し、交換で発行するアクセストークンと
// リフレッシュトークンに引き継ぐ。宛先の違うトークンは照合で受け付けない (RFC 8707 §2)
type OAuthAudience int32

const (
	// OAuthAudiencePublicAPIは公開API (`/api/v1`) 宛て
	OAuthAudiencePublicAPI OAuthAudience = 0
	// OAuthAudienceMCPServerはリモートMCPサーバー (`/mcp`) 宛て
	OAuthAudienceMCPServer OAuthAudience = 1
)
