# Markdown から PDF へ（md2pdf）

[English](markdown.md)

`md2pdf` は、`html2pdf` と同じエンジンで Markdown ファイルを PDF に変換します。エンジンが対応する HTML と CSS は [reference-ja.md](reference-ja.md) を参照してください。

## 1. 処理の流れ

```
Markdown → (goldmark) → HTML → コードブロック変換 → + 既定 CSS → Convert → PDF
```

1. [goldmark](https://github.com/yuin/goldmark) で Markdown を HTML に変換します（CommonMark に加えて、GitHub Flavored Markdown の表・取り消し線・タスクリストに対応）。ソース中の生の HTML はそのまま通します。
2. すべての `<pre>` ブロックを書き換え、改行とインデントが残るようにします（4 章）。
3. 結果を完全な HTML 文書で包みます。`<style>` 要素には `DefaultMarkdownCSS` と、指定があれば追加の CSS を続けて入れます。
4. 文書を `Convert` に渡します。そのため、通常の `Options`（ページサイズ、余白、フォント、拡大率）がそのまま使えます。

関数:

- `MarkdownToHTML(src []byte) ([]byte, error)` は HTML 文書を返します（`DefaultMarkdownCSS` のみ入ります）。
- `ConvertMarkdown(src []byte, w io.Writer, opts *Options) error` は Markdown を PDF にして `w` へ書き出します。
- `ConvertMarkdownFile(mdPath, pdfPath string, opts *Options) error` はファイルを読み込み、成功したときだけ PDF を書き出します。`BaseDir` の既定値は Markdown ファイルのあるフォルダです（`opts` はコピーして使い、呼び出し側の値は変更しません）。相対パスの画像はここを基準に解決されます。

## 2. CSS

`DefaultMarkdownCSS` は、見出し（`h1` と `h2` は下罫線付き）、段落、リンク、リスト、引用（左罫線と薄い背景）、`pre` と `code`（薄い灰色の背景）、表（`table`・`th`・`td` に罫線、`th` に背景色）、`hr` の見た目を決めます。エンジンが対応する CSS だけを使っています。

上書きしたいときは、追加の CSS を渡します。既定の CSS の後ろに追加されるため、詳細度が同じなら後のルールが優先されます。

- CLI: `-css style.css`
- ライブラリ: `Options.MarkdownCSS`（`ConvertMarkdown` と `ConvertMarkdownFile` だけが使います）

```css
h1 { color: #205090; }
body { font-size: 12pt; }
```

## 3. 対応する要素

| Markdown | 結果 |
|---|---|
| `#` から `######` | `h1` から `h6` |
| 段落、`**bold**` | `p`、太字 |
| `*emphasis*` | `em`。通常の文字で描かれます（斜体のフェースがないため） |
| `~~strikethrough~~` | `del`。通常の文字で描かれます |
| `` `code` `` | `code`。背景は描かれません（5 章参照） |
| フェンス付き・インデント形式のコードブロック | `pre`。改行とインデントを保持 |
| `-` / `1.` のリスト、入れ子のリスト | `ul` / `ol` |
| `- [x]` のタスクリスト | リスト項目。チェックボックスの代わりに `[x]` / `[ ]` の文字マーカーで表示します |
| `>` の引用 | `blockquote` |
| GFM の表 | 罫線付きの `table` |
| `---` | `hr`（水平線。CSS で描画） |
| `[text](url)` | `a`。青い下線付きの文字（対応する URL はクリック可能） |
| `![alt](path)` | `img`。相対パスは `BaseDir` を基準に解決 |
| 生の HTML | そのまま通します |

## 4. コードブロックの空白変換

レイアウトエンジンは、改行を含むすべての空白を 1 つに畳み込み、空白を保持する `pre` も持ちません。そのままではコードブロックが 1 行につぶれてしまいます。そこで、goldmark が出力した各 `<pre>` ブロックを、変換の前に次のように書き換えます。

1. 内側の `<code>` タグを取り除きます。テキストは goldmark が HTML エスケープ済みのものをそのまま残します。
2. テキストを行に分割し、末尾の空行を 1 つだけ捨てます。
3. 各行で、タブを 4 つの空白にし、すべての空白を `&nbsp;`（U+00A0。レイアウトエンジンが保持し、空白の幅で描きます）に置き換えます。空行は `&nbsp;` 1 つにして、行の高さを保ちます。
4. 行を `<br>` でつなぎ、もう一度 `<pre>` で包みます。

## 5. 制限

- 斜体はありません。強調は通常の文字で描かれます。
- コードブロックとインラインコードは等幅フォントになりません。エンジンは `monospace` などの総称フォントファミリーを読み飛ばすためです。インデントと改行は保たれますが、文字は桁揃えになりません。
- インライン要素には背景色が描かれないため、インラインの `code` に背景は付きません。コードブロック（ブロック要素）には付きます。
- PDF のリンクは `http`、`https`、`mailto`、同一文書のフラグメントに対応し、他のスキームは無視します。
- 長いコード行は通常の文章と同じように折り返されます。
- ソース中の生の HTML はそのまま通ります（goldmark を unsafe レンダリングで動かしています）。信頼できる Markdown だけを変換してください。

PDF のしおりは既定で有効です（`-bookmarks=false` で無効化）。`-title`、`-author`、`-subject` でメタデータを上書きできます。タイトルが指定されず生成 HTML に `<title>` がない場合、最初の空でない `h1` をタイトルにします。PNG 出力ではこれらの PDF 機能を無視します。
