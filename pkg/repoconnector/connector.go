package repoconnector

import (
	"context"
	"io"

	apispec "github.com/tacokumo/api-spec"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/appconfig"

	"go.yaml.in/yaml/v3"
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

// CloneApplicationRepository は指定されたGitリポジトリからアプリケーション設定をクローンする
func CloneApplicationRepository(
	ctx context.Context,
	connector GitRepositoryConnector,
	url string,
	refName string,
	appConfigPath string,
) (repo appconfig.Repository, err error) {
	wt, err := connector.Clone(ctx, url, refName)
	if err != nil {
		return appconfig.Repository{}, err
	}

	f, err := wt.Open(appConfigPath)
	if err != nil {
		return appconfig.Repository{}, err
	}
	defer func() {
		err = f.Close()
	}()

	appConfig := apispec.AppConfig{}
	if err := yaml.NewDecoder(f).Decode(&appConfig); err != nil {
		return appconfig.Repository{}, err
	}

	return appconfig.Repository{
		AppConfig: appConfig,
	}, nil
}
