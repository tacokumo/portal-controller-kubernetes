# ADR001-application-and-release

## Status

Approved

## Decision Outcome

TACOKUMOではあるアプリケーションは必ずあるGitリポジトリと紐づいていることを仮定します。
それぞれのアプリケーションは `Application` リソースとして表現されます。

`Application` リソースは、実際にクラスタ上にデプロイされるアプリケーションのインスタンスとして `Release` リソースを生成します。
それぞれの `Release` リソースは特定のGitコミットを指し示して作成されます。

TACOKUMOは、ロールバック以外のケースで手動でリリースすることを想定しません。
すべては自動でリリースされることを前提にします。

### リソース設計

Applicationリソースは以下のような情報を持ちます。

```yaml
# 中略
metadata:
  name: "portal-controller-kubernetes"
spec:
  repo:
    url: "https://github.com/tacokumo/portal-controller-kubernetes"
    appConfigPath: "/appconfig.yaml"
```

Releaseリソースは以下のような情報を持ちます。

```yaml
# 中略
spec:
  applicationRef:
    name: "portal-controller-kubernetes"
  commit: "a1b2c3d4e5f6g7h8i9j0"
```

`development/staging/production` など、事前定義されたステージについてリリースを行う場合は、
同名の `Release` リソースがすでに存在する場合、その　``

ここで、appconfig.yamlは各リポジトリに存在するTACOKUMOアプリケーションの定義ファイルです。
appconfig.yamlは以下のような情報を持ちます。

```yaml
# 中略
stages:
  - name: "production"
    policy:
      type: "branch"
      branch:
        name: "main"
```

### プレビュー

サーバアプリケーションの開発については、Pull Requestに対して自動でプレビュー環境が構築されます。
プレビュー環境が必要な場合は、 appconfig.yaml に `.service.preview` のような設定を用意します。

Application Reconcilerは対象アプリケーションの `.service.preview` 設定を検出し、
`<application-name>-preview-<pull-request-number>` のような名前でReleaseリソースを生成します。

### ロールバック

TACOKUMOでは、事前定義されたメトリクスの変化に基づく自動ロールバックの他、
開発者による手動ロールバックをサポートします。

手動ロールバックの場合は、
ロールバック先のコミットを対象の `Release` リソースに設定し直すことで実現します。

