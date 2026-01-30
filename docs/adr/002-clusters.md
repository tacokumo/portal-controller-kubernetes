# ADR002-clusters

## Status

Approved

## Decision Outcome

TACOKUMOでは､データプレーンとして複数のクラスタを管理できるようにします｡
これは､TACOKUMO自体がクラスタ間のワークロードマイグレーションを可能にするためです｡

Portal Controllerはそれぞれのクラスタに対して一つずつインストールされます｡つまり､N Cluster = N Portal Controllerとなります｡

クラスタをどのように管理するかについては､Portal Controller Kubernetesでは関心外とします｡クラスタ管理はadminやその他コンポーネントが担います｡

### エンドポイント設計

Portal Controller Kubernetesでは､
各アプリケーションの外部エンドポイントは `Gateway` リソースとして表現されます｡
Gatewayの責務は､クラスタにインストールされたGateway APIコントローラをラップして､
外部エンドポイントを管理することです｡
一般的には､AWS Route53やGoogle Cloud DNSなどのDNSサービスと連携して､
アプリケーションの外部エンドポイントを管理します｡

ここで､クラスタAにデプロイされたアプリケーションをクラスタBにマイグレーションする場合の動作を簡潔に説明します｡

- クラスタAにデプロイされたアプリケーションは `app.example.com` というエンドポイントを持っています
- ユーザはTACOKUMO APIを使って､クラスタマイグレーションのジョブをキックします
- APIはクラスタBにアプリケーションをデプロイします
  - このとき､クラスタBの `Gateway` リソースは､ `app.example.com` に対するweighted routingを設定します
  - weightは0%に設定されます
    - より具体的には､ `app.example.com` に対し `app.clusterA.example.com` と `app.clusterB.example.com` という2つのCNAMEレコードを作成し､`app.clusterA.example.com` に100%､ `app.clusterB.example.com` に0%のweightを設定します
- マイグレーションジョブは､weightedの値を少しずつ変更しながら､外部からのE2Eトラフィックの状態を監視します
- 問題がなければ､最終的に `app.clusterA.example.com` に0%､ `app.clusterB.example.com` に100%のweightを設定します
- マイグレーションが完了したら､クラスタA上のアプリケーションを削除します
