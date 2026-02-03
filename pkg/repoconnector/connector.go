package repoconnector

import (
	"context"
	"io"

	appconfig "github.com/tacokumo/appconfig"

	"go.yaml.in/yaml/v3"
)

// GitRepositoryConnector はGitリポジトリへの接続を抽象化するインターフェース
type GitRepositoryConnector interface {
	// Clone はリポジトリをcloneし、Worktreeを返す
	Clone(ctx context.Context, url string, branch string) (Worktree, error)
	// GetLatestCommit は指定されたブランチの最新コミットハッシュを取得する
	GetLatestCommit(ctx context.Context, url string, branch string) (string, error)
}

// Worktree はGitのworktreeを抽象化するインターフェース
type Worktree interface {
	// Open は指定されたパスのファイルを開く
	Open(path string) (io.ReadCloser, error)
}

// CloneApplicationRepository は指定されたGitリポジトリからアプリケーション設定をクローンする
func CloneApplicationRepository(
	ctx context.Context,
	connector GitRepositoryConnector,
	url string,
	refName string,
	appConfigPath string,
) (cfg appconfig.AppConfig, err error) {
	wt, err := connector.Clone(ctx, url, refName)
	if err != nil {
		return appconfig.AppConfig{}, err
	}

	f, err := wt.Open(appConfigPath)
	if err != nil {
		return appconfig.AppConfig{}, err
	}
	defer func() {
		err = f.Close()
	}()

	var appCfg appconfig.AppConfig
	if err := yaml.NewDecoder(f).Decode(&appCfg); err != nil {
		return appconfig.AppConfig{}, err
	}

	return appCfg, nil
}
