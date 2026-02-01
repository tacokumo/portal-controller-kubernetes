# ADR004-Gateway

## Status

Approved

## Decision Outcome

TACOKUMOでは､ `Gateway` と呼ばれる機能を提供して､TACOKUMO上のワークロードに対する外部からのアクセスを可能にします｡
`Gateway` は､KubernetesのIngressやGateway APIの概念を抽象化したものです｡
あるいは､今後実装される可能性のあるTACOKUMO AWSディストリビューションのELBやALBのようなロードバランサー機能を抽象化したものとも言えます｡

以後､このドキュメントではKubernetesディストリビューションにおける `Gateway` の実装について説明します｡

### Gatewayリソース

TACOKUMOのPortal APIで､ `Gateway` リソースを管理する機能を提供します｡
portal-controller-kubernetesは､自分のクラスタでどのIngress/Gateway API controllerが利用可能かを検出し､
それに応じたIngressやGatewayリソースを作成します｡

アプリケーションに必要なのは､｢どのポートを公開するか｣という情報だけです｡
これらはappconfig.yamlに記述され､コントローラによって解釈されます｡

portal-controller-kubernetesの責務は､
IngressやGateway/HTTPRouteなどのKubernetesリソースを作成し､
appconfigの情報を基に､適切なルーティング設定を行うことです｡

完全にInternal Networkに閉じたアプリケーションを作成することができるように､
Gatewayリソースを作成せずにアプリケーションをデプロイすることも可能です｡

KubernetesのCluster DNSで同じnamespaceのアプリケーションが `<app-name>.<namespace>.svc.cluster.local` , あるいは `<app-name>` で解決できることを利用します｡

### OAuthやOIDC連携

TACOKUMOはインターネットに公開するアプリケーションの他､
イントラ向けのアプリケーションもサポートします｡

そのため､アプリケーションに対してOAuthやOIDCによる認証を追加する機能を提供します｡

TACOKUMOを運用する人間が､それぞれのテナント/アプリケーションに対し共有できる設定を行えるように､
`ClusterGateway` のようなリソースを提供します｡
それ以外に､ `Gateway` でテナント/アプリケーション固有の設定を行うことも可能とします｡

### ドメイン設計

例えばあるTACOKUMOクラスタ(ここで､クラスタとはK8sクラスタ一つではなく､一つ以上のK8sクラスタを含むTACOKUMOインスタンス全体を指します)に､
`example.com` というドメインを割り当てたとします｡

ここで､各テナントにおけるアプリケーションとそのドメインの関係を以下のように設計します｡

- テナントA
  - アプリケーション1: `app1.tenantA.app.example.com`
- テナントB
  - アプリケーション1: `app1.tenantB.app.example.com`
  - アプリケーション2: `app2.tenantB.app.example.com`

TACOKUMOのPortal Controllerは､Gatewayリソースの作成･更新を検知すると､
Gatewayが対象とするアプリケーションの名前を下に､
上記のようなFQDNを各種DNSプロバイダに登録します｡

DNSプロバイダはプラガブルに設定可能です｡
AWS Route53やCloudflare DNSなどをサポートする予定です｡

`mydomain.io` などのカスタムドメインを利用したい場合がありますが､
TACOKUMOでビルトインにサポートされません｡
ユーザは `mydomain.io` から `app.tenant.app.example.com` へのCNAMEレコードを自分で設定する必要があります｡
