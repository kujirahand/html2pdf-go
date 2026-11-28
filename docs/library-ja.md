# Go ライブラリとしての使い方

[English](library.md)

## 1. はじめに

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

`Options` の全フィールドは [reference-ja.md](reference-ja.md) を参照してください。

## 2. HTML の変換

`html2pdf.Convert` は HTML のバイト列を受け取り、PDF を `io.Writer` に書き出します。PDF の代わりに PNG 画像が欲しいときは `html2pdf.ConvertPNG(src, opts, scale)` を使います。ページごとの PNG を返します。各ページは 72dpi のポイントサイズの `scale` 倍で描画されます（0 以下なら 2、つまり 144dpi）。

## 3. Markdown の変換

`ConvertMarkdown` は [goldmark](https://github.com/yuin/goldmark) で Markdown を HTML にし、同じエンジンに渡します（Markdown → HTML → PDF）。`ConvertMarkdownPNG(src, opts, scale)` はその PNG 版で、ページごとの PNG を返します。

```go
// Markdown のバイト列から PDF のバイト列へ
err := html2pdf.ConvertMarkdown(src, &buf, &html2pdf.Options{PageSize: "A4", MarkdownCSS: "h1 { color: #205090; }"})

// Markdown ファイルから PDF ファイルへ（BaseDir の既定値は Markdown ファイルのあるフォルダ）
err = html2pdf.ConvertMarkdownFile("input.md", "out.pdf", nil)

// 既定のスタイルシートを埋め込んだ HTML 文書だけを得る
htmlDoc, err := html2pdf.MarkdownToHTML(src)
```

`Options.MarkdownCSS` は既定のスタイルシートの後ろに追加されます。画像の相対パスは Markdown ファイルのあるフォルダを基準に解決されます。対応する Markdown と制限は [markdown-ja.md](markdown-ja.md) を参照してください。

## 4. 知っておきたいオプション

- `BaseDir` と `AllowExternalFiles`: 既定では、文書が読めるファイル（画像・スタイルシート・フレーム文書）は文書自身のフォルダ（`BaseDir`）の内側にあるものだけです。フォルダの外を指す絶対パスや `..`、`//host/share` のようなネットワークパスは拒否されます。信頼できる文書では、`Options.AllowExternalFiles` でこの制限を解除できます。
- `Scale`: 内容の縮小・拡大率です（既定 1.0）。wkhtmltopdf の zoom / smart shrinking や Chrome の printToPDF の `scale` に相当する調整機能で、実寸ではわずかに次のページへ溢れる文書を収めたいときに使います。
