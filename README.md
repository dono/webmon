# WEBMON
Webサイトの死活監視やパフォーマンス測定を行うプログラム
Slack通知機能付き


# Usage


## メモ
### httptraceのコールバック関数が呼ばれる順番
- DNSStart()
- DNSDone()
- ConnectStart()
- ConnectDone()
- TLSHandshakeStart()
- TLSHandshakeDone()
- GotConn()
- GotFirstResponseByte()

## ToDo
- データベースとの連携
- レスポンスボディと証明書の更新チェック
- UIの実装