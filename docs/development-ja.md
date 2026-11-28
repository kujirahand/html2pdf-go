# 開発

[English](development.md)

タスクランナー [just](https://github.com/casey/just) を使います。`just` だけでレシピ一覧を表示します。

```sh
just test      # go test ./...
just make      # ネイティブの CLI バイナリを bin/ に生成
just fonts     # IPAex フォントを fonts/ に取得（取得済みなら何もしない）
just samples   # サンプル PDF 一式を out/ に生成
```
