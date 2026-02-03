package repoconnector

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalConnector はローカルファイルシステムを使用するテスト用の GitRepositoryConnector 実装
type LocalConnector struct {
	// basePath はローカルファイルシステム上のベースディレクトリ
	basePath string
	// latestCommits はブランチ名とコミットハッシュのマッピング（テスト用）
	latestCommits map[string]string
}

// NewLocalConnector は LocalConnector を生成する
// basePath はテストデータのルートディレクトリを指定する
func NewLocalConnector(basePath string) *LocalConnector {
	return &LocalConnector{
		basePath: basePath,
	}
}

// WithLatestCommits はテスト用にブランチとコミットのマッピングを設定する
func (c *LocalConnector) WithLatestCommits(commits map[string]string) *LocalConnector {
	c.latestCommits = commits
	return c
}

// Clone はローカルのディレクトリを Worktree として返す
// url と branch は無視され、basePath をそのまま使用する
func (c *LocalConnector) Clone(_ context.Context, _ string, _ string) (Worktree, error) {
	return &localWorktree{basePath: c.basePath}, nil
}

// GetLatestCommit は指定されたブランチの最新コミットハッシュを取得する
// テスト用の実装として、WithLatestCommits で設定された値を返す
func (c *LocalConnector) GetLatestCommit(_ context.Context, _ string, branch string) (string, error) {
	if c.latestCommits == nil {
		return "", fmt.Errorf("no commits configured for LocalConnector")
	}
	commit, ok := c.latestCommits[branch]
	if !ok {
		return "", fmt.Errorf("branch %q not found in configured commits", branch)
	}
	return commit, nil
}

// localWorktree はローカルファイルシステムを Worktree インターフェースに適合させる
type localWorktree struct {
	basePath string
}

// Open は指定されたパスのファイルを開く
func (w *localWorktree) Open(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(w.basePath, path)
	return os.Open(fullPath)
}
