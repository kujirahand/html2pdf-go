# アーキテクチャ

[English version](architecture.md)

html2pdf-go は、HTML+CSS を純粋な Go だけで PDF に変換するエンジンです。以下のパイプラインは実装済みで、対応範囲と制限事項は後述のとおりです。

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
| `internal/datauri` | `data:` URI のデコード（base64／パーセントエンコード）。`layout` と `render` が画像ソースとして使う |
| `internal/render` | レイアウト済みページを `pdf` パッケージ経由で描画する |
| `internal/pdf` | PDF ファイルの書き出し（オブジェクト、コンテンツストリーム、フォント・画像の埋め込み） |

## Scale（縮小・拡大率）

`Options.Scale`（CLI では `-scale`）で内容全体を縮小・拡大できます。wkhtmltopdf の zoom / smart shrinking や、Chrome の printToPDF の `scale` に相当する調整機能です。0 以下は 1.0 として扱います。

- `layout` は「実ページ寸法 / Scale」の仮想ページでレイアウトし、返す `Page` の `Width`/`Height` は実寸に戻して `Scale` を持たせる
- `render` は `Page.Scale` を `pdf.Page.SetScale` に渡し、`pdf` が描画時に座標・サイズ・線幅・フォントサイズへ Scale を掛ける（Y 軸の反転は実寸のページ高さで行う）
- Scale が 1 のときは従来と同じ結果になる

## 当面の対応範囲

実装済みの対応範囲の詳細は [reference-ja.md](reference-ja.md)（English: [reference.md](reference.md)）を参照してください。

請求書 HTML の実測に基づきます。

- **タグ**: `h1`, `p`, `div`, `br`, `table`, `tr`, `th`, `td`, `img`, `center`, `ul`, `li`, `ol`（`ul`, `li`, `ol` は請求書では未使用だが、一般的な文書で必須のため初期対応範囲に含める）
- **セレクタ**: `*`、要素、`.class`、`#id`、それらの複合（`要素.class`、`要素#id`、`.class#id` など）、子孫（`.total td` など）。specificity は（ID 数, クラス数, 要素数）の 3 つ組
- **プロパティ**: `font-size`, `font-weight`, `font-family`（`FontFile` / `FontFileBold` や、インストール済みフォントのファミリー名で指定）, `color`, `background-color`, `text-align`, `text-decoration`, `line-height`, `margin`（および各辺）, `padding`（および各辺）, `border`（および各辺。スタイルは solid / dashed / dotted / double、角丸は `border-radius` と各隅のロングハンド）, `width`, `page-break-after`
- **単位**: 長さ系プロパティ（`width`, `margin`, `padding`, `font-size` など）でサポートする単位は `px`, `em`, `rem`, `%`, `pt`, `mm`, `cm`, `in`、および単位なしの `0`
  - 換算基準: 1px = 0.75pt（96dpi → 72pt）、1in = 25.4mm = 72pt
  - `em` は当該要素の `font-size` 基準（ただし `font-size` 自身の指定では親要素の `font-size` 基準）、`rem` はルート要素の `font-size` 基準
  - `%` はプロパティごとに基準が異なる（`width` 系は包含ブロックの幅）

`rowspan` でつながる行は一つのページ分割単位になります。表レイアウトは `pageSim` でページ分割をシミュレートし、最初の `thead` と次のブロックが収まる継続ページでは flow 座標のまま見出しを再描画します。

## 制限事項

- 1ページより高いテーブル行、`rowspan` ブロックや画像は分割されず、はみ出した部分は描画されない
- 入れ子のテーブルは深さに対して処理時間が指数的に増えるため、深い入れ子には向かない
- テーブルの列幅は自動レイアウト方式で、`width` 未指定の表ではセルの `%` 指定は、列幅の合計が収まる場合は使われず、縮小が必要な場合にのみ使われる
- `text-align: center` のブロック内では表が中央寄せされる（ブラウザとの相違点）

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
