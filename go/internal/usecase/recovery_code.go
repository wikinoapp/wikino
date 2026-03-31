package usecase

// removeRecoveryCode はリカバリーコードをリストから削除する
func removeRecoveryCode(codes []string, codeToRemove string) []string {
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if code != codeToRemove {
			result = append(result, code)
		}
	}
	return result
}
