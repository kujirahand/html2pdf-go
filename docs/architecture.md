# アーキテクチャ

html2pdf-go は、HTML+CSS を純粋な Go だけで PDF に変換するエンジンです。現在は骨組みのみで、変換処理は未実装です。

## パイプライン

```
HTML
  │
  ▼
dom      HTML をパースして文書ツリーを構築
  │
  ▼
css      スタイルシートを適用（セレクタマッチ → カスケード → 計算済みスタイル）
  │
  ▼
layout   ボックスツリー生成 → ブロック/テーブルレイアウト → ページ分割
  │
  ▼
render   レイアウト済みページを描画命令に変換
  │
  ▼
pdf      PDF オブジェクト・コンテンツストリームを書き出し
  │
  ▼
PDF
```

`text` パッケージは、フォントの計測と折り返しのために `layout` と `render` から利用されます。

## パッケージの責務

| パッケージ | 責務 |
|---|---|
| `internal/dom` | HTML をパースし、スタイル適用済みの文書ツリーを構築する（将来 `x/net/html` を利用） |
| `internal/css` | CSS のパース、セレクタマッチング、カスケード、計算済みスタイルの生成 |
| `internal/text` | フォントの読み込み・計測、文字列の整形（折り返し・禁則処理） |
| `internal/layout` | ボックスツリーの生成、ブロック/テーブルレイアウト、ページネーション |
| `internal/render` | レイアウト済みページを `pdf` パッケージ経由で描画する |
| `internal/pdf` | PDF ファイルの書き出し（オブジェクト、コンテンツストリーム、フォント・画像の埋め込み） |

## 当面の対応範囲

請求書 HTML の実測に基づきます。

- **タグ**: `h1`, `p`, `div`, `br`, `table`, `tr`, `th`, `td`, `img`, `center`, `ul`, `li`, `ol`（`ul`, `li`, `ol` は請求書では未使用だが、一般的な文書で必須のため初期対応範囲に含める）
- **セレクタ**: `*`、要素、`.class`、`要素.class`、子孫（`.total td` など）
- **プロパティ**: `font-size`, `font-weight`, `color`, `background-color`, `text-align`, `text-decoration`, `line-height`, `margin`（および各辺）, `padding`（および各辺）, `border`（および各辺）, `width`, `page-break-after`
- **単位**: 長さ系プロパティ（`width`, `margin`, `padding`, `font-size` など）でサポートする単位は `px`, `em`, `rem`, `%`, `pt`, `mm`, `cm`, `in`、および単位なしの `0`
  - 換算基準: 1px = 0.75pt（96dpi → 72pt）、1in = 25.4mm = 72pt
  - `em` は当該要素の `font-size` 基準（ただし `font-size` 自身の指定では親要素の `font-size` 基準）、`rem` はルート要素の `font-size` 基準
  - `%` はプロパティごとに基準が異なる（`width` 系は包含ブロックの幅）

## 非目標（初期）

- JavaScript
- flex
- grid
- float
- position

## ロードマップ

| バージョン | 内容 |
|---|---|
| v0 | テキストのみ描画 |
| v1 | ブロックと罫線（リスト `ul`/`ol`/`li` のマーカー・番号・インデントを含む） |
| v2 | テーブル |
| v3 | 画像と改ページ |
