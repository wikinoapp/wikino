package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// WithLockedは、プロセス間の設定ロックを取り、最新の設定を読み直してfnを呼ぶ。
// fnはトークンの変更と設定の保存を同じロック内で行う。ロックファイルは削除せず、
// 別プロセスが古いファイルをロックしたまま新しいファイルを作ることを防ぐ。
func WithLocked(ctx context.Context, dir string, fn func(*Config) error) error {
	return WithFileLock(ctx, dir, "config.lock", func() error {
		cfg, err := Load(dir)
		if err != nil {
			return err
		}
		return fn(cfg)
	})
}

// WithFileLockは、dir内のnameのプロセス間ロックを取り、fnの完了まで保持する。
// ctxはロック待ちを中止するために使い、開始済みのfnからはロックを取り上げない。
// ロックファイルは削除せず、inodeの入れ替わりによる排他の抜けを防ぐ。
func WithFileLock(ctx context.Context, dir, name string, fn func() error) (err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil { // #nosec G703 -- 利用者が指定した設定ディレクトリにロックを置く
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0o600) // #nosec G304 G703 -- 利用者の設定ディレクトリ内にロックを置く
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		locked, lockErr := tryLock(file)
		if lockErr != nil {
			return lockErr
		}
		if locked {
			break
		}
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	defer func() {
		if unlockErr := unlock(file); unlockErr != nil {
			err = errors.Join(err, unlockErr)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}
