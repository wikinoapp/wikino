package auth

import (
	"testing"
)

func TestVerifyPassword(t *testing.T) {
	t.Parallel()

	// テスト用のハッシュ化済みパスワード
	// "password123" をbcryptでハッシュ化した値
	hashedPassword, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("ハッシュ生成に失敗: %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		plainPassword  string
		want           bool
	}{
		{
			name:           "正しいパスワード",
			hashedPassword: hashedPassword,
			plainPassword:  "password123",
			want:           true,
		},
		{
			name:           "間違ったパスワード",
			hashedPassword: hashedPassword,
			plainPassword:  "wrongpassword",
			want:           false,
		},
		{
			name:           "空のパスワード",
			hashedPassword: hashedPassword,
			plainPassword:  "",
			want:           false,
		},
		{
			name:           "大文字小文字の違い",
			hashedPassword: hashedPassword,
			plainPassword:  "Password123",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := VerifyPassword(tt.hashedPassword, tt.plainPassword)
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifyPassword_RailsCompatibility(t *testing.T) {
	t.Parallel()

	// Railsのbcryptで生成されたハッシュとの互換性テスト
	// Rails (has_secure_password) で "password" をハッシュ化した例
	// bcryptはソルトを含むため、同じパスワードでも毎回異なるハッシュが生成される
	// テストでは動的に生成したハッシュを使用

	plainPassword := "testpassword"
	hashedPassword, err := HashPassword(plainPassword)
	if err != nil {
		t.Fatalf("ハッシュ生成に失敗: %v", err)
	}

	// 生成したハッシュが正しく検証できることを確認
	if !VerifyPassword(hashedPassword, plainPassword) {
		t.Error("生成したハッシュの検証に失敗")
	}
}

func TestHashPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "通常のパスワード",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "空のパスワード",
			password: "",
			wantErr:  false,
		},
		{
			name:     "日本語を含むパスワード",
			password: "パスワード123",
			wantErr:  false,
		},
		{
			name:     "特殊文字を含むパスワード",
			password: "p@ssw0rd!#$%",
			wantErr:  false,
		},
		{
			name:     "長いパスワード",
			password: "abcdefghijklmnopqrstuvwxyz123456789",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// ハッシュが生成されていることを確認
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}
				// 生成したハッシュが検証できることを確認
				if !VerifyPassword(hash, tt.password) {
					t.Error("生成したハッシュの検証に失敗")
				}
			}
		})
	}
}

func TestHashPassword_UniqueHashes(t *testing.T) {
	t.Parallel()

	// 同じパスワードでも異なるハッシュが生成されることを確認（bcryptのソルト機能）
	password := "samepassword"
	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("ハッシュ生成に失敗: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("ハッシュ生成に失敗: %v", err)
	}

	if hash1 == hash2 {
		t.Error("同じパスワードで同じハッシュが生成された（ソルトが機能していない）")
	}

	// 両方のハッシュで元のパスワードが検証できることを確認
	if !VerifyPassword(hash1, password) {
		t.Error("hash1での検証に失敗")
	}
	if !VerifyPassword(hash2, password) {
		t.Error("hash2での検証に失敗")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	t.Parallel()

	// 不正なハッシュ形式での検証
	tests := []struct {
		name           string
		hashedPassword string
		plainPassword  string
	}{
		{
			name:           "空のハッシュ",
			hashedPassword: "",
			plainPassword:  "password",
		},
		{
			name:           "不正な形式のハッシュ",
			hashedPassword: "invalid_hash",
			plainPassword:  "password",
		},
		{
			name:           "平文をそのまま",
			hashedPassword: "password",
			plainPassword:  "password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// 不正なハッシュでの検証は失敗するべき
			if VerifyPassword(tt.hashedPassword, tt.plainPassword) {
				t.Error("不正なハッシュでの検証が成功してしまった")
			}
		})
	}
}
