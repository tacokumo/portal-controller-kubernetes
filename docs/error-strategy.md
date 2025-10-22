# エラー戦略

本プロジェクトではエラーを二種類に分類しています。

- Requeue Error
- Other Error

## Requeue Error

｢Reconcile処理としては完遂できなかったが、再度Reconcileを試みることで解決する可能性があるエラー｣を指します。

例えば、PodがReadyになるまで待つ処理や、
ヘルスチェックが成功するまで待つ処理などが該当します。

このエラーが発生した場合、コントローラはログを出力した後、一定時間後に再度Reconcileを試みます。

## Other Error

コントローラロジックのエラーを即時エラーとして計上したい場合に用いられます。

この場合はExponential Backoffによる再試行が行われます。
