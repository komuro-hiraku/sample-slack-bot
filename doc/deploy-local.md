# ローカル環境で Slack bot を実行する

ローカルでSlack Botを実行してテストできるようにします。

## 必要なもの

- [ngrok](https://ngrok.com/)

## 事前準備

ngrok をインストールします。公式サイトではアーカイブを取得して展開してバイナリを配置となっていますが、macで brew を利用しているならば brew でインストール可能です。

### brew install

```bash
$ brew cask install ngrok
```

### ngrok へサインアップ

Free アカウントで問題ありません。[こちら](https://dashboard.ngrok.com/signup)でサインアップを行います。

### authtoken の登録

マイページを啓くと `2. Connect your account` の項目があります。こちらをコピペして実行します。

```bash
$ ngrok authtoken XXXXXXXXXXXXXXXX
```

### ngrok の起動

以下のコマンドで中継するTunnelを起動します。

```bash
$ ngrok http 8080
```

これで準備完了。

## slack-bot-go の起動

ngrok が起動している状態で

```bash
$ go run main.go
```

を実行します。

## SlackにBotを認識させる

Slack はいくつかの Verification エンドポイントを持ってBotを認識します。

### Event Subscriptions

Slack API のアプリケーションを登録すると Settings や Features のページへ遷移します。
`Features > Event Subscriptions` へ移動し、 `Enable Events` を ON にします。

すると Request URL が表示されます。ここに

`https://{ngrok.domain}/slack/events` を入力するとVerificationが実行され、正常に導通が確認できればVerfiedとなります。ngrok は起動するたびにドメインのIDが少し変わるので、 ngrok を起動し直すたびに修正が必要です。

### Interactivity & Shortcuts

Interactive Components（つまりボタンとかセレクトメニュー等）を利用しているコマンドの場合、こちらを有効にする必要があります。

`Features > Interactivity & Shortcuts` へ遷移。 `Interactivity` を ON にします。

すると Request URL の入力欄が表示されます。そこに以下を入力します。

`https://{ngrok.domain}/slack/actions` 指定のエンドポイントで Action を処理することができるようになります。
