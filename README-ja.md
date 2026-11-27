# html2pdf-go

[English README](README.md)

pure Go の HTML→PDF 変換エンジンです。HTML のパース、CSS の適用、レイアウト、PDF の書き出しまで実装済みです。対応する HTML/CSS の範囲は [docs/reference-ja.md](docs/reference-ja.md) を参照してください。

## チュートリアル

まずは CLI で、HTML ファイルを 1 つ PDF に変換してみます。所要時間は数分です。

### 1. インストールする

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest
```

### 2. HTML ファイルを作る

`hello.html` という名前で、次の内容のファイルを作ります。見出し・段落・箇条書き・表と、`<style>` での見た目の指定を含む、最小の例です。

```html
<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="utf-8">
<style>
  body { font-size: 12pt; margin: 48px; }
  h1 { font-size: 20pt; color: #205090; }
  table { border: 1px solid #333; }
  th, td { border: 1px solid #333; padding: 4px 8px; }
</style>
</head>
<body>
<h1>はじめての PDF</h1>
<p>これは html2pdf-go で作った PDF です。日本語もそのまま表示できます。</p>
<ul>
  <li>見出しと段落</li>
  <li>箇条書き</li>
  <li>表</li>
</ul>
<table>
  <tr><th>項目</th><th>内容</th></tr>
  <tr><td>ページサイズ</td><td>A4</td></tr>
  <tr><td>フォント</td><td>自動で選ばれます</td></tr>
</table>
</body>
</html>
```

### 3. PDF に変換する

```sh
html2pdf hello.html
```

同じフォルダに `hello.pdf` ができます。開いてみると、見出しが青く大きくなり、表に罫線が引かれた PDF になっているはずです。

出力先やページ設定を変えたいときは、オプションを指定します。

```sh
html2pdf -o out.pdf -page-size A4 -margin 48 hello.html   # 出力先と余白を指定
html2pdf -scale 0.95 hello.html                            # 全体を 95% に縮小
```

### 4. 次のステップ

- 見た目を変える: `<style>` の中身を変えます。使える HTML 要素と CSS は [docs/reference-ja.md](docs/reference-ja.md) に一覧があります
- Go のプログラムから使う: 次の「ライブラリとしての使い方」を見てください
- 実例を見る: [examples/](examples/) に住所録や領収書などのサンプル HTML があります

## ライブラリとしての使い方

```sh
go get github.com/kujirahand/html2pdf-go
```

```go
package main

import (
	"bytes"
	"log"
	"os"

	html2pdf "github.com/kujirahand/html2pdf-go"
)

func main() {
	var buf bytes.Buffer
	src := []byte("<h1>Hello</h1><p>こんにちは</p>")
	if err := html2pdf.Convert(src, &buf, &html2pdf.Options{PageSize: "A4"}); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("out.pdf", buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
```

## CLI の使い方

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest

html2pdf input.html                    # input.pdf を出力
html2pdf -o out.pdf -page-size A4 input.html
html2pdf -scale 0.95 input.html        # 内容を 95% に縮小
```

`-scale`（ライブラリでは `Options.Scale`）は内容の縮小・拡大率です（既定 1.0）。wkhtmltopdf の zoom / smart shrinking や Chrome の printToPDF の `scale` に相当する調整機能で、実寸ではわずかに次のページへ溢れる文書を収めたいときに使います。

## 開発

タスクランナー [just](https://github.com/casey/just) を使います。`just` だけでレシピ一覧を表示します。

```sh
just test      # go test ./...
just fonts     # IPAex フォントを fonts/ に取得（取得済みなら何もしない）
just samples   # サンプル PDF 一式を out/ に生成
```

## 対応範囲とロードマップ

[docs/architecture-ja.md](docs/architecture-ja.md)（English: [docs/architecture.md](docs/architecture.md)）を参照してください。

対応する HTML 要素・CSS の一覧は [docs/reference-ja.md](docs/reference-ja.md)（English: [docs/reference.md](docs/reference.md)）を参照してください。

## ライセンス

MIT ライセンスです。詳細は `LICENSE` ファイルを参照してください。
