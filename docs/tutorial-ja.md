# チュートリアル

[English](tutorial.md)

まずは CLI で、HTML ファイルを 1 つ PDF に変換してみます。

## 1. インストールする

Linux・macOS・Windows 向けのビルド済みバイナリは [GitHub Releases ページ](https://github.com/kujirahand/html2pdf-go/releases)にあります。コマンドラインから最新リリースをインストールするには、次のとおりです。

```sh
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.ps1 | iex
```

バイナリは署名されていないため、ブラウザでダウンロードすると OS にブロックされます（Gatekeeper / SmartScreen）。これらのスクリプトでインストールすればブロックを避けられ、インストール前に SHA-256 チェックサムも検証します。Go を使う場合は次のとおりです。

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest
```

## 2. HTML ファイルを作る

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

## 3. PDF に変換する

```sh
html2pdf hello.html
```

同じフォルダに `hello.pdf` ができます。開いてみると、見出しが青く大きくなり、表に罫線が引かれた PDF になっているはずです。

出力先やページ設定を変えたいときは、オプションを指定します。

```sh
html2pdf -o out.pdf -page-size A4 -margin 48 hello.html   # 出力先と余白を指定
html2pdf -scale 0.95 hello.html                            # 全体を 95% に縮小
```

## 4. 次のステップ

- 見た目を変える: `<style>` の中身を変えます。使える HTML 要素と CSS は [reference-ja.md](reference-ja.md) に一覧があります
- 改ページする: 次のページを始めたい位置に `<page-break></page-break>` を置きます（例: `<p>Page 1</p><page-break></page-break><p>Page 2</p>`）。先頭・末尾・連続した `<page-break>` で空白ページはできません
- Go のプログラムから使う: [ライブラリガイド](library-ja.md)を見てください
- 実例を見る: [examples/](../examples/) に住所録、領収書、イベント案内状、コンサートチケット、カフェメニュー、名刺、カレンダー、搭乗券、議事録、レシピなどのサンプル文書（HTML と Markdown、日英両版）があります。`just examples` で全てを `examples/pdf/` に PDF 化できます
