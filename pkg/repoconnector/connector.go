package repoconnector

import (
	"context"
	"io"
)

// GitRepositoryConnector はGitリポジトリへの接続を抽象化するインターフェース
type GitRepositoryConnector interface {
	// Clone はリポジトリをcloneし、Worktreeを返す
	Clone(ctx context.Context, url string, branch string) (Worktree, error)
}

// Worktree はGitのworktreeを抽象化するインターフェース
type Worktree interface {
	// Open は指定されたパスのファイルを開く
	Open(path string) (io.ReadCloser, error)
}
