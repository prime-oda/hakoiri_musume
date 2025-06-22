# GitHub Pages デプロイメントガイド

## 🚀 GitHub Pagesで公開する手順

### 1. リポジトリの準備

ファイルが正しく配置されていることを確認してください：

```
/
├── index.html          # ゲームのメインページ
├── style.css           # スタイルシート
├── game.js             # ゲームロジック
├── README.md           # プロジェクト説明
├── docs.md             # ドキュメントページ
├── _config.yml         # Jekyll設定
└── .github/workflows/pages.yml  # GitHub Actions設定
```

### 2. GitHubでのPages設定

1. **GitHubリポジトリにアクセス**
   - リポジトリページを開く

2. **Settings タブをクリック**

3. **Pages セクションを見つける**
   - 左サイドバーから "Pages" を選択

4. **Source の設定**
   - "GitHub Actions" を選択
   - これにより自動デプロイが有効になります

### 3. 自動デプロイの確認

1. **mainブランチにpush**
   ```bash
   git add .
   git commit -m "GitHub Pages setup"
   git push origin main
   ```

2. **Actions タブで進行確認**
   - リポジトリの "Actions" タブを確認
   - "Deploy to GitHub Pages" ワークフローが実行される

3. **デプロイ完了の確認**
   - 数分後にデプロイが完了
   - Pages設定で公開URLを確認

### 4. アクセスURL

公開されたゲームは以下のURLでアクセスできます：

```
https://USERNAME.github.io/REPOSITORY_NAME/
```

例：
```
https://alice.github.io/hakoiri_musume/
```

### 5. カスタムドメインの設定（オプション）

独自ドメインを使用する場合：

1. **DNS設定**
   - CNAMEレコードを設定: `www.yourdomain.com` → `USERNAME.github.io`

2. **GitHub設定**
   - Pages設定で "Custom domain" にドメインを入力
   - "Enforce HTTPS" を有効化

### 6. トラブルシューティング

#### よくある問題と解決方法

**❌ 404エラーが表示される**
- ファイル名とパスを確認
- `index.html` がルートディレクトリにあるか確認

**❌ CSSやJSが読み込まれない**
- パスの確認（相対パスを使用）
- ブラウザキャッシュをクリア

**❌ デプロイが失敗する**
- Actions タブでエラーログを確認
- `_config.yml` の設定を確認

#### ログの確認方法

1. リポジトリの "Actions" タブ
2. 失敗したワークフローをクリック
3. エラーメッセージを確認

### 7. 更新方法

ゲームを更新する場合：

1. ファイルを編集
2. コミット・プッシュ
3. 自動的に再デプロイ

```bash
# 変更をコミット
git add .
git commit -m "ゲーム機能の改善"
git push origin main

# 数分後に自動更新される
```

## 📱 モバイル対応

作成したゲームはレスポンシブ対応済みです：
- スマートフォンでも快適にプレイ可能
- タッチ操作に対応
- 画面サイズに自動調整

## 🔗 共有方法

公開後は以下の方法で共有できます：
- URLを直接共有
- QRコードを生成して共有
- SNSでの投稿
- ブログやWebサイトへの埋め込み

---

これで「箱入り娘の大家族」が世界中の人にプレイしてもらえるようになります！