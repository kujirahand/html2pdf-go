# html2pdf-go

pure Go の HTML→PDF 変換エンジンです。HTML のパース、CSS の適用、レイアウト、PDF の書き出しまで実装済みです。対応する HTML/CSS の範囲は [docs/reference-ja.md](docs/reference-ja.md) を参照してください。

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

## フォルダ構成

| パス | 内容 |
|---|---|
| `html2pdf.go` | 公開 API（`Convert`, `ConvertFile`, `Options`） |
| `cmd/html2pdf/` | CLI ツール |
| `internal/dom` | HTML パースと文書ツリー |
| `internal/css` | CSS パース、カスケード、計算済みスタイル |
| `internal/text` | フォント計測、折り返し・禁則処理 |
| `internal/layout` | ボックス生成、レイアウト、ページネーション |
| `internal/render` | レイアウト結果の描画 |
| `internal/pdf` | PDF 書き出し |
| `examples/basic/` | 最小の使用例 |
| `testdata/` | テスト用 HTML |
| `docs/` | 設計ドキュメント |

## 対応範囲とロードマップ

[docs/architecture.md](docs/architecture.md) を参照してください。

対応する HTML 要素・CSS の一覧は [docs/reference-ja.md](docs/reference-ja.md)（English: [docs/reference.md](docs/reference.md)）を参照してください。

## ライセンス

MIT ライセンスです。詳細は `LICENSE` ファイルを参照してください。
