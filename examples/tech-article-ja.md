# Go だけで HTML から PDF を作る

PDF を作るのに、ヘッドレスブラウザや外部コマンドは本当に必要でしょうか。この記事では、純 Go のライブラリで HTML から PDF を生成する方法と、その選び方を紹介します。

## はじめに

請求書やレポートを自動生成するとき、よく使われるのは次の 3 つの方法です。

1. ヘッドレスブラウザに印刷させる
2. `wkhtmltopdf` のような外部コマンドを呼ぶ
3. 純 Go のライブラリで直接生成する

3 番目の方法は、バイナリ 1 つで配布でき、コンテナイメージも小さくできます。

## 基本的な使い方

`Convert` 関数に HTML 文字列を渡すと、PDF のバイト列が返ります。

```go
package main

import (
	"log"
	"os"

	"example.com/pdfkit"
)

func main() {
	html := `<h1>Hello</h1><p>This PDF was made with Go.</p>`

	pdf, err := pdfkit.Convert(html, pdfkit.Options{
		PageSize: "A4",
		FontDirs: []string{"./fonts"},
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("out.pdf", pdf, 0o644); err != nil {
		log.Fatal(err)
	}
}
```

`Options.FontDirs` に日本語フォントのフォルダを指定すると、日本語もそのまま描画できます。

> **注意：** ライブラリは JavaScript を実行しません。動的なページは、先に HTML を書き出してから渡してください。

## 方式の比較

| 方式 | 配布のしやすさ | 再現性 | JavaScript | 速度 |
|---|---|---|---|---|
| ヘッドレスブラウザ | 低い（本体が大きい） | 高い | 対応 | 遅め |
| 外部コマンド | 中（インストールが必要） | 中 | 一部対応 | 普通 |
| 純 Go ライブラリ | 高い（単一バイナリ） | 対応範囲内で高い | 非対応 | 速い |

## まとめ

- テンプレートから帳票を作るなら、純 Go が手軽で安定します。
- 複雑な CSS や JavaScript が必要なページには、ブラウザ方式が向いています。
- どの方式でも、まず小さなテンプレートで試してから本番に広げるのがおすすめです。

※ 記事中のパッケージ名 `pdfkit` とコードは説明用の架空のものです。
