package config

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestWithLockedSerializesProcesses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var commands []*exec.Cmd
	for i := range 3 {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWithLockedProcess$")
		cmd.Env = append(os.Environ(), "WIKINO_TEST_LOCK_DIR="+dir, "WIKINO_TEST_LOCK_MEMBER="+strconv.Itoa(i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, cmd)
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Credentials) != 3 {
		t.Fatalf("資格情報数 = %d、期待値 = 3", len(cfg.Credentials))
	}
}

// TestWithLockedProcessは、別プロセス同士が読み取りから保存まで同じロックを使うことを検証する。
func TestWithLockedProcess(t *testing.T) {
	t.Parallel()
	dir := os.Getenv("WIKINO_TEST_LOCK_DIR")
	if dir == "" {
		t.Skip("親テストから子プロセスとして実行する")
	}
	err := WithLocked(context.Background(), dir, func(cfg *Config) error {
		time.Sleep(30 * time.Millisecond)
		member := os.Getenv("WIKINO_TEST_LOCK_MEMBER")
		cfg.AddCredential(Credential{Host: DefaultHost, Space: "team", MemberID: member, MemberAtname: member, Store: "file"})
		return cfg.Save(dir)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithLockedCancellationAndRelease(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sentinel := errors.New("更新失敗")
	err := WithLocked(context.Background(), dir, func(_ *Config) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()
		if err := WithLocked(ctx, dir, func(_ *Config) error { t.Error("ロック中に更新された"); return nil }); !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("ロック待ちのエラー = %v、期限切れを期待", err)
		}
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("戻り値 = %v、元のエラーを期待", err)
	}
	if err := WithLocked(context.Background(), dir, func(_ *Config) error { return nil }); err != nil {
		t.Fatalf("失敗後のロック取得 = %v", err)
	}
}
