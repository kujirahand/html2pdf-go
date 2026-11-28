![html2pdf-go — HTML/CSS/Markdown から PDF へ](docs/html2pdf-go-logo800.png)

# html2pdf-go - HTML/CSS/Markdown→PDF

[English README](README.md)

HTML/CSS/Markdown→PDFの変換ライブラリとCLIです。HTML/CSSの主要タグをサポートすることで、それなりに見栄えの良いPDFを手軽に作成できます。Pure Go言語で実装しているので、実行ファイル1つで、ブラウザ依存もなく、手軽に使える小さなツールです。

## インストール

```sh
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.ps1 | iex
```

バイナリは署名されていないため、ブラウザでダウンロードすると OS にブロックされます（Gatekeeper / SmartScreen）。これらのスクリプトでインストールすればブロックを避けられ、インストール前に SHA-256 チェックサムも検証します。

## サンプル

どんな PDF が作れるかは、サンプル文書のカタログ [examples/README.md](examples/README.md)（日本語）をご覧ください。

## CLI の使い方

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest

html2pdf input.html                    # input.pdf を出力
html2pdf -o out.pdf -page-size A4 input.html
html2pdf -page-width 91 -page-height 55 input.html   # カスタムページサイズ（ミリメートル）
html2pdf -scale 0.95 input.html        # 内容を 95% に縮小
html2pdf -e '<h1>Hello</h1>'           # HTML を直接指定。output.html と output.pdf を出力
html2pdf -format png input.html        # input.png を出力（PNG 画像出力）
```

Markdown ファイルは `md2pdf` で変換します。

```sh
go install github.com/kujirahand/html2pdf-go/cmd/md2pdf@latest

md2pdf input.md                        # input.pdf を出力
md2pdf -o out.pdf -margin 48 input.md
md2pdf -css style.css input.md         # 既定のスタイルシートの後ろに自分の CSS を追加
```

フラグの一覧は [docs/reference-ja.md](docs/reference-ja.md)、Markdown の詳細は [docs/markdown-ja.md](docs/markdown-ja.md) を参照してください。

## 対応する HTML/CSS/Markdown

- HTML: 文書構造（`div`、`section`、`center`、`page-break` など）、見出しと段落、リスト、表（入れ子を含む）、画像（PNG/JPEG/GIF）、フレーム（`frameset`、`iframe`）。全一覧: [docs/reference-ja.md](docs/reference-ja.md)。
- CSS: クラス・ID・要素・子孫セレクタ、フォント・色・背景・margin と padding・罫線（`border-radius` を含む）・`width`・`text-align`・`line-height`、`page-break-after` による改ページ。全一覧: [docs/reference-ja.md](docs/reference-ja.md)。
- Markdown: 見出し、太字、リスト（入れ子・タスクリストを含む）、引用、コードブロック、GFM の表、水平線、リンク、画像、生の HTML。全一覧: [docs/markdown-ja.md](docs/markdown-ja.md)。

未対応: `float`、`position`、flex・grid レイアウト、`@media` / `@font-face`、属性セレクタと擬似クラス、斜体、JavaScript。

## ドキュメント

- [チュートリアル](docs/tutorial-ja.md): CLI で HTML ファイルを 1 つ PDF に変換する手順。
- [ライブラリガイド](docs/library-ja.md): Go ライブラリとしての使い方。
- [対応範囲リファレンス](docs/reference-ja.md): 対応する HTML 要素・CSS・変換オプション。
- [Markdown](docs/markdown-ja.md): Markdown の変換（md2pdf）の解説。
- [アーキテクチャ](docs/architecture-ja.md): 設計と変換パイプライン。
- [開発](docs/development-ja.md): `just` による開発タスク。

## ライセンス

MIT ライセンスです。詳細は `LICENSE` ファイルを参照してください。
