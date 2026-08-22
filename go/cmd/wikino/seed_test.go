package main

import (
	"context"
	"fmt"
	"testing"
)

func TestSeedDatabaseRejectsNonDevEnvBeforeLoadingConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  string
	}{
		{name: "テスト環境を拒否する", env: "test"},
		{name: "本番環境を拒否する", env: "prod"},
		{name: "未設定を拒否する", env: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			want := fmt.Sprintf("開発用データを扱うコマンドは開発環境でのみ実行できます (APP_ENV=%s)", tt.env)
			err := seedDatabase(context.Background(), tt.env)
			if err == nil {
				t.Fatal("開発環境以外ではエラーを期待したがnilだった")
			}
			if err.Error() != want {
				t.Errorf("エラーが %q であることを期待したが %q だった", want, err)
			}
		})
	}
}
