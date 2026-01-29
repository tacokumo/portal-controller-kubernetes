package repoconnector

import (
	"context"
	"io"

	"github.com/go-git/go-billy/v6"
	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"
)

// DefaultConnector は go-git を使用した GitRepositoryConnector の実装
type DefaultConnector struct{}

// NewDefaultConnector は DefaultConnector を生成する
func NewDefaultConnector() *DefaultConnector {
	return &DefaultConnector{}
}

// Clone はリポジトリをcloneし、Worktreeを返す
func (c *DefaultConnector) Clone(ctx context.Context, url string, branch string) (Worktree, error) {
	fs := memfs.New()
	storer := memory.NewStorage()

	gitRepo, err := git.CloneContext(ctx, storer, fs, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return nil, err
	}

	wt, err := gitRepo.Worktree()
	if err != nil {
		return nil, err
	}

	return &defaultWorktree{fs: wt.Filesystem}, nil
}

// defaultWorktree は go-git の worktree を Worktree インターフェースに適合させる
type defaultWorktree struct {
	fs billy.Filesystem
}

// Open は指定されたパスのファイルを開く
func (w *defaultWorktree) Open(path string) (io.ReadCloser, error) {
	return w.fs.Open(path)
}
