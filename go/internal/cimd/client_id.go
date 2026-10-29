// Package cimdは、OAuthのClient ID Metadata Document (CIMD) のクライアントIDの判定と、
// メタデータの文書の取得・検証を行う。CIMDのクライアントは、自分が公開するHTTPSのURLを
// クライアントIDに使い、認可サーバーはそのURLから取得した文書でクライアントを知る。
//
// 知らないホストへの取得はSSRFの面を広げ、知らないクライアントへの同意はフィッシングに使われうる
// ため、許可リストのホストのクライアントIDだけを受け付ける。
package cimd

import (
	"net/url"
	"slices"
	"strings"
)

// allowedHostsは、CIMDのクライアントIDとして受け付けるURLのホスト。主なMCPクライアント (Claude・
// ChatGPT・VS Code) が文書を配信するホストで、ポートを含めて完全に一致するものだけを受け付ける
var allowedHosts = []string{"claude.ai", "chatgpt.com", "vscode.dev", "insiders.vscode.dev"}

// IsClientIDMetadataDocumentURLは、クライアントIDがCIMDのクライアントID (HTTPSのURL) の形であるかを返す。
// Wikinoに登録したクライアントのクライアントIDはURLにならないため、この形であれば登録したクライアントではない。
// 許可リストで受け付けるかはIsAllowedで確かめる
func IsClientIDMetadataDocumentURL(clientID string) bool {
	return strings.HasPrefix(clientID, "https://")
}

// IsAllowedは、クライアントIDが許可リストのホストのCIMDのクライアントIDとして受け付けられるかを返す。
//
// URLはHTTPSでパスを含み、ユーザー情報・フラグメント・ドットのセグメントを含まないものに限る
// (draft-ietf-oauth-client-id-metadata-document §3)。ホストはポートも含めて許可リストと
// 文字列で一致させ、既定のポート (`:443`) も正規化しない。
func IsAllowed(clientID string) bool {
	u, err := url.Parse(clientID)
	if err != nil {
		return false
	}
	if u.Scheme != "https" || u.User != nil || u.Fragment != "" || strings.Contains(clientID, "#") {
		return false
	}
	if !slices.Contains(allowedHosts, u.Host) {
		return false
	}
	if u.Path == "" || u.Path == "/" {
		return false
	}
	for _, segment := range strings.Split(u.Path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return u.String() == clientID
}
