// Package clientipはクライアントIPアドレスの取得機能を提供します
package clientip

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// GetTrustedClientIPは接続元が信頼済みプロキシのときだけX-Forwarded-Forを右から読む。
// 各プロキシが送信元IPをヘッダーの末尾に追記する構成を前提とする。
// 信頼していない接続元から送られた転送ヘッダーは無視する。
func GetTrustedClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !isTrustedProxy(peer, trustedProxies) {
		return peer.String()
	}
	for _, raw := range reverseForwardedFor(r.Header.Get("X-Forwarded-For")) {
		ip, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil {
			return peer.String()
		}
		ip = ip.Unmap()
		if !isTrustedProxy(ip, trustedProxies) {
			return ip.String()
		}
		peer = ip
	}
	return peer.String()
}

func isTrustedProxy(ip netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func reverseForwardedFor(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}

// GetClientIPはHTTPリクエストからクライアントのIPアドレスを取得します
//
// 優先順位:
// 1. CF-Connecting-IP (Cloudflareが設定する実際のクライアントIP)
// 2. X-Forwarded-For (プロキシチェーンの最初のIP)
// 3. X-Real-IP
// 4. RemoteAddr (直接接続の場合)
func GetClientIP(r *http.Request) string {
	// 1. CF-Connecting-IP (Cloudflareが設定する実際のクライアントIP)
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	// 2. X-Forwarded-Forの最初のIP (プロキシチェーンの最初)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// カンマ区切りの最初のIPを取得
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// 3. X-Real-IPヘッダーをチェック
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 4. RemoteAddrを使用 (直接接続の場合、ポート番号を除去)
	clientIP := r.RemoteAddr
	if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
		clientIP = clientIP[:idx]
	}
	return clientIP
}
