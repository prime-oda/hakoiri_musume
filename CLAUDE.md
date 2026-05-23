# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

「箱入り娘の大家族」のWebゲームとGo実装のパズルソルバーです。

### ゲームについて
- 箱入り娘の拡張版「箱入り娘の大家族」パズル
- 通常の箱入り娘（5×4マス）より大きな盤面（6×5マス）
- スライディングパズルで娘を脱出させるのが目的
- 1つの駒の連続した動き（空きマスを伝って到達できる位置への移動。途中で何度向きを変えてもよい）を1手と数える

#### 駒の構成（合計15コマ）
- **娘（2×2）**: 脱出させる目標の駒
- **父（1×2）**: 縦長の駒
- **母（1×2）**: 縦長の駒
- **大番頭（4×1）**: 横長の大きな駒
- **手代（1×1）**: 小さな駒（1個）
- **女中（2×1）**: 横長の駒
- **番頭（2×1）**: 横長の駒
- **祖父（2×1）**: 横長の駒
- **祖母（2×1）**: 横長の駒
- **番犬（1×1）**: 小さな駒（1個）
- **兄嫁（1×1）**: 小さな駒（1個）
- **丁稚（1×1）**: 小さな駒（3個）

#### 初期配置
```
空父娘娘母空
空父娘娘母空
手大大大大嫁
丁番頭女中丁
犬祖父祖母丁
```

## リポジトリ構成

```
.
├── web/                  # GitHub Pagesで配信されるブラウザ版ゲーム
│   ├── index.html
│   ├── style.css
│   ├── game.js           # ゲーム本体（描画・操作・手数カウント）
│   ├── solver.js         # ブラウザ内 最短手ソルバー（BFS + 事前計算テーブル）
│   └── precomputed/      # ゴール距離テーブル（goal_distances.bin / .bin.gz）
├── cmd/solver/main.go    # ソルバー CLI のエントリポイント
├── cmd/precompute/main.go# ゴール距離テーブル生成 CLI
├── pkg/
│   ├── board/            # 盤面表現・移動生成・Zobristハッシュ
│   │   ├── board.go
│   │   ├── moves.go      # 連続移動ルールの flood-fill 移動生成
│   │   ├── hash.go
│   │   └── config_extended.go / config_classic.go  # 盤面定義（build tag で切替）
│   └── solver/           # 探索アルゴリズム
│       ├── bfs.go
│       ├── bidirectional.go
│       ├── precompute.go # ゴール距離テーブル構築
│       ├── packed.go / hashset.go
├── go.mod / go.sum
└── .github/workflows/static.yml  # GitHub Pages デプロイ（web/ を配信）
```

## Webゲーム（web/）

- HTML5 Canvas + 素のJavaScript（ビルドステップなし）
- マウス / タッチ両対応
- ローカル確認: `cd web && python3 -m http.server 8000`
- GitHub Pagesは `web/` 配下を直接配信（`.github/workflows/static.yml` の `path: './web'`）

## Goソルバー（cmd/solver, pkg/）

### ビルド・実行
```bash
go build -o solver ./cmd/solver
./solver -algo bfs -verbose
```

### 主なフラグ
- `-algo`: `bfs` / `bidirectional`
- `-max-states`: 探索する最大状態数（デフォルト 1,000,000,000）
- `-max-time`: 探索の最大時間（デフォルト 30分）
- `-verbose`: 進捗表示
- `-bench`: ベンチマークモード

### アルゴリズム
- **BFS**: 最短手解を保証（既定。Zobristハッシュ + 事前計算テーブルと組み合わせる）
- **双方向探索**: 順・逆方向から探索

> 連続移動ルールでは Manhattan 距離が許容ヒューリスティックにならない（1手で娘を任意位置へ運べる）ため、ヒューリスティック依存の IDA* / A* / 並列探索は廃止した。

### 主要パッケージ
- **`pkg/board`**: 盤面を1次元配列で表現し、移動生成は `MoveGenerator` で実装。Zobristハッシュで状態の同一性判定と差分更新を行う
- **`pkg/solver`**: 上記アルゴリズム実装と `SearchResult` / `SolverStats` の共通型

### 解答出力
`./solver` 実行成功時は `solve.txt` に初期盤面・手順・最終盤面を出力する。

## トラブルシューティング

- ビルド失敗: Goのバージョンを確認（`go version`）。`go.mod` の Go バージョンに合わせる
- 探索が終わらない: `-max-time` と `-max-states` を小さくして傾向を確認、または `-algo bidirectional` を試す
- GitHub Pages反映されない: Actionsの `Deploy static content to Pages` が成功したか確認
