# 対応範囲リファレンス

[English](reference.md)

このマニュアルは html2pdf-go が対応する範囲の一覧です。内容はすべてソースコードで裏付けたものです。ここに書かれていない機能は**未対応**として扱ってください。

## 1. 概要

- 描画されるのは `<body>` の中身だけです（`<frameset>` 文書ではフレームの内容。「フレーム」を参照）。それ以外（`<head>`、`<title>`、`<meta>` など）は表示されません。
- スタイルの指定方法は次の 3 つです。
  - 文書内のどこにある `<style>` 要素でも読み込まれます（すべて、出現順）。
  - `<link rel="stylesheet" href="...">` 要素。`<style>` 要素と合わせて出現順に適用されます。読み込まれるのはローカルファイルだけで、相対パスはベースディレクトリが基準です。`AllowExternalFiles` を指定しない限りファイルはベースディレクトリの内側になければならず（`BaseDir` と `AllowExternalFiles` のオプションを参照）、通常のファイルで、サイズは 32 MiB 以下でなければなりません。`href` に `://` を含むものと `data:` で始まるものは無視され（リモート CSS は取得しません）、`rel` に `alternate` を含むものも無視されます。読み込めないローカルファイルを指す場合は、変換がエラーになります。
  - 要素の `style` 属性。
- `@import` はスキップされます（外部 CSS ファイルの中でも同じです）。
- `<script>` は破棄され、JavaScript は実行されません。
- CSS コメント（`/* ... */`）は除去されます。未知のプロパティや不正な値は黙って無視されます。
- アットルールのブロック（`@media`、`@page`、`@font-face`、`@keyframes` など）は、中に入れ子になったルールも含めて丸ごと読み飛ばされます。
- `!important` は値から取り除かれますが、優先順位には**影響しません**。
- キーワード `inherit`・`initial`・`unset`、`var()`、`calc()` は未対応です。

## 2. 対応 HTML 要素

ここに載っていない要素や未知の要素は `display: inline` となり、インラインのテキストとして描かれます（テキストは表示されますが、特別なスタイルは付きません）。

### 文書構造

| 要素 | 描画方法 | 備考 |
|---|---|---|
| `body` | ブロック、margin 8px | 描画の起点。 |
| `div`, `section`, `article`, `header`, `footer`, `address` | ブロック | `address` は斜体になりません。 |
| `center` | ブロック、`text-align: center` | |
| `page-break` | ブロック、その位置で強制改ページ | 次のページを始めたい位置に `<page-break></page-break>`（または `<page-break>` だけ）を置きます。閉じていない `<page-break>` の後ろに書いた内容は、要素の後ろへ移されます。9 章を参照。 |
| `frameset` | フレームのグリッド | 下の「フレーム」を参照。 |
| `frame` | グリッドの 1 セル | 下の「フレーム」を参照。`noframes` の中身は描画されません。 |
| `iframe` | ブロックレベルの埋め込み文書 | 下の「インラインフレーム」を参照。 |
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `template`, `object`, `embed`, `svg` | 非表示（`display: none`） | 子要素を含めて一切描画されません。 |

### フレーム

`<frameset>` を含む文書（複数ある場合は最初の 1 つが使われます）は、`<body>` の代わりにフレームを並べた 1 ページとして描画されます。

- `rows` と `cols` はカンマ区切りのリストです。各トークンは px 数値（× 0.75 ポイント）、全体に対する割合（`30%`）、相対長（`*` は重み 1、`2*` は重み 2。ほかのトークンの残りを分け合う）のいずれかです。属性が無い方向は `100%` として扱います。領域は余白の内側のページ全体です。
- セルの数は行数 × 列数で、`frame` 要素が出現順に割り当てられます。余ったセルは空になり、余った frame は無視されます。
- `frame` の `src` は、ベースディレクトリ基準のローカルファイルとして読み込まれます。読めない場合は変換がエラーになります。URL である `src`（`//` で始まるもの、または `https:`、`data:`、`about:`、`blob:` などのスキーム付きのもの）と、`src` の無い frame は空のセルになります。
- 各フレーム文書は、セルの大きさをページサイズ、余白 0、外側と同じ Scale で個別にレイアウトされます。その文書の外部 CSS や画像は、そのファイルのあるディレクトリが基準になり、フレーム文書自体が frameset でもかまいません。
- フレーム文書の最初のページだけが表示され、内容はセルでクリップされます。
- `px` と `%` のトラックの合計が領域を超える場合は、すべてが比例して縮んで領域に収まり、相対長のトラックは 0 になります。
- 上限（入れ子 8 段、フレーム文書の総数 1000。frame、iframe、`srcdoc` を数える）を超えると変換エラーになります。

### インラインフレーム

ブロックレベルの子要素として現れた `iframe` は、文書を埋め込んだボックスとして描画されます。

- ボックスの大きさは、CSS の `width` / `height`、次に `width` / `height` 属性（ピクセル。`width` の % は親のコンテンツ幅に対する割合で、`height` の % は既定値にフォールバック）、最後に HTML の既定値 300x150 px の順で決まります。マージン、パディング、罫線は他のブロックと同じ扱いで、フレーム文書はコンテンツ領域に収まります。
- `srcdoc` があればその HTML を文書として使います（HTML の仕様どおり `src` より優先）。ローカルの `src` は基準ディレクトリから読み、読めない場合は変換エラーになります。URL である `src`（「フレーム」を参照）や、ソースが無い場合は空のボックスになります。
- フレーム文書はコンテンツ領域の大きさでレイアウトされ、最初のページだけがボックスでクリップされて表示されます。フレームと iframe は、入れ子 8 段、フレーム文書の総数 1000 の上限を共有し、どちらかを超えると変換エラーになります。
- `span` や `a` などのインライン要素の中にある `iframe` は描画されません。ブロックの流れの直接の子である `iframe`（`p` の直接の子を含む）は、ブロックとして描画されます。
- 表のセル内の `iframe` は、列幅の計算で解決済みの幅（上のサイズの決め方）として数えられます。

### ブロックのテキスト

| 要素 | 描画方法 | 備考 |
|---|---|---|
| `p` | ブロック、margin `1em 0` | |
| `h1` – `h6` | ブロック、太字。サイズと余白は組み込みの既定値（8 章） | |
| `blockquote` | ブロック、margin `1em 0` | インデントなし。 |
| `pre` | ブロック、等幅 | 等幅フォントで描かれます。空白・改行は引き続き畳まれます。 |
| `code`, `kbd`, `samp`, `tt` | インライン、等幅、背景 `#f0f0f0` | |
| `br` | 強制改行 | |
| `b`, `strong` | 太字 | |
| `u`, `ins` | 下線 | |
| `a` | 青色・下線 | `href` は使われません（リンクにはなりません）。 |

### リスト

| 要素 | 描画方法 | 備考 |
|---|---|---|
| `ul`, `ol` | ブロック、margin `1em 0`、`padding-left: 40px` | 別のリストの中では margin 0。 |
| `li` | マーカー付きのリスト項目 | マーカーの規則は 9 章。 |

### 表

| 要素 | 描画方法 | 備考 |
|---|---|---|
| `table` | 表 | 入れ子の表も使えます。 |
| `thead`, `tbody`, `tfoot` | 行グループ | 3 つとも同じ扱いで、行はソース順に配置されます（`tfoot` は末尾に移動しません）。 |
| `tr` | 表の行 | |
| `td` | セル、padding 1px | |
| `th` | セル、padding 1px、太字、中央寄せ | |
| `caption` | — | 描画されません（表レイアウトが無視します）。 |
| `colgroup`, `col` | — | 無視されます。 |

`table`・行グループ・`tr` の直下に置かれたテキストや、行以外の要素は描画されません。

### 画像

| 要素 | 描画方法 | 備考 |
|---|---|---|
| `img` | インライン画像 | `src` を使用。PNG・JPEG・GIF。9 章を参照。 |

### 非表示要素

「文書構造」の `display: none` の一覧を参照してください。CSS の `display: none` で任意の要素を非表示にもできます。

## 3. 対応属性

実装が読む属性は次のものだけです。それ以外の属性（`href`、`alt`、`align`、`bgcolor`、`border`、`cellpadding`、`cellspacing`、`rowspan`、`start` など）は効果がありません。属性名は大文字小文字を区別しません。

| 属性 | 対象要素 | 効果 |
|---|---|---|
| `class` | すべての要素 | `.class` セレクタのマッチに使われます（空白区切りの複数指定可）。 |
| `id` | すべての要素 | `#id` セレクタのマッチに使われます（大文字小文字を区別する）。 |
| `style` | すべての要素 | インラインの宣言。優先順位は最高です。 |
| `src` | `iframe`、`frame` | ローカルの HTML ファイルのパス（`BaseDir` 基準。`AllowExternalFiles` を指定しない限り `BaseDir` の内側に限られ、サイズは 32 MiB 以下）。URL（`//` で始まるもの、または `https:`、`data:`、`about:` などのスキーム付きのもの）は読み込まれず、空のボックス／セルになります。読めないファイルはエラーです。`iframe` では `srcdoc` が優先されます。 |
| `srcdoc` | `iframe` | 埋め込む文書の HTML ソース。`src` より優先されます。 |
| `rows`、`cols` | `frameset` | カンマ区切りのトラックの大きさ（px、割合、相対長 `*` / `2*`）。`px` と `%` のトラックが領域を超える場合は、比例して縮んで領域に収まります。 |
| `src` | `img` | 画像ファイルのパス、または `data:` URI。相対パスは `BaseDir` を基準に解決され、`AllowExternalFiles` を指定しない限り `BaseDir` の内側に限られます。`//` や `\\` で始まるパスは読み込まれません。1 つの PDF に含まれるデコード後の画像は合計 1 億ピクセルまでで、予算を超える画像はスキップされます。`data:` URI は base64 形式（`data:image/png;base64,...`）とパーセントエンコード形式（`data:image/gif,...`）に対応し、メディアタイプを指定する場合は `image/*` でなければなりません。デコードできるのは PNG・JPEG・GIF です。`http(s)://` の URL（`://` を含むもの）は読み込まれません。 |
| `width` | すべての要素（意味があるのは `img`、`iframe`、`table`、`td`、`th`、ブロック要素） | CSS の長さとして解釈され、単位なしの数値は px。最も優先順位の低い `width` として扱われ、CSS の `width` があればそちらが優先されます。負の値や不正な値は無視されます。 |
| `height` | `iframe` | 単位なしの px 数値のみ（% は無視）。最も優先順位の低い `height` として扱われ、CSS の `height` があればそちらが優先されます。 |
| `colspan` | `td`, `th` | 結合する列数。1 以下や数値でない値は 1 として扱われます。上限は 1000。 |

## 4. CSS セレクタ

| 対応するもの | 例 |
|---|---|
| ユニバーサル | `*` |
| 要素（大文字小文字を区別しない） | `p` |
| クラス（大文字小文字を区別する） | `.note` |
| 要素 + クラス | `p.note` |
| 複合クラス | `.a.b`, `td.a.b` |
| ID（大文字小文字を区別する。1 つの複合セレクタに最大 1 つ） | `#main` |
| 要素 + ID、クラス + ID | `div#main`, `.a#b`, `div.note#main` |
| 子孫（空白） | `.total td`, `div p span` |
| セレクタリスト（カンマ） | `h1, h2, .title` |

| 対応しないもの | 例 |
|---|---|
| 属性セレクタ | `[type=text]` |
| 擬似クラス・擬似要素 | `:first-child`, `::before` |
| 子・隣接・間接兄弟コンビネータ | `>`, `+`, `~` |

`[ ] : > + ~ ( )` のいずれかを含む複合セレクタは**無効**となり、どの要素にもマッチしません。空の断片（`#`、`div#`、`a..b`）や 2 つ以上の `#`（`#a#b`）を含むものも無効です。無効なセレクタはエラーにならず、カンマ区切りの他のセレクタは有効なままです。`div>p` も `div > p` も、`>` の部分が無効になるため無効です。

## 5. カスケード

宣言は次の順に適用され、後の段階が前の段階を上書きします。

1. `width` 属性のヒント
2. 組み込み（UA）スタイルシート（8 章）
3. 作者スタイルシート（`<style>` 要素）。specificity の昇順、同じなら出現順（後のルールが勝つ）。作者のルールは specificity に関係なく UA のルールより優先されます。
4. インラインの `style` 属性

1 つのルール内では後の宣言が勝ちます。`!important` は無視されます。

**specificity** は左から順に比較する 3 つの数の組です。

1. セレクタ内の ID の数（すべての複合セレクタの合計）
2. クラス名の数（同じく合計）
3. 要素名の数（同じく合計）。`*` は 0 と数えます。

例: `#a` = (1, 0, 0)、`div#a.b` = (1, 1, 1)、`td.a.b` = (0, 2, 1)、`.total td` = (0, 1, 1)、`p` = (0, 0, 1)。ID はクラスがいくつあっても常に優先されます。

**継承。** 継承されるプロパティは `font-size`、`font-weight`、`font-family`、`color`、`text-align`、`text-decoration`、`line-height` だけです。それ以外は要素ごとに初期値から始まります。

## 6. 値の形式

### 長さ

| 単位 | 意味 |
|---|---|
| `px` | 0.75pt（1/96 インチ） |
| `pt` | 1/72 インチ |
| `mm` | ミリメートル |
| `cm` | センチメートル |
| `in` | インチ（72pt） |
| `em` | その要素自身の `font-size`。`font-size` プロパティでは親の `font-size` |
| `rem` | ルートのフォントサイズ（16px = 12pt） |
| `%` | プロパティごとに基準が異なる。`width`・`margin`・`padding`・罫線の太さは包含ブロックの幅、`font-size` は親の `font-size`、`line-height` はフォントサイズ、`border-radius` はボックスの幅（X）・高さ（Y） |
| 単位なし | px として扱う（例: `margin: 5`）。例外は `line-height`（倍率）、`font-weight`（太さ）、`font-size`（拒否される） |

それ以外の単位（`vw`、`vh`、`ex`、`pc`、`ch` など）は未対応です。先頭の `+` / `-` は受け付けます。`2.em` のような不正な数値は拒否されます。

### 色

| 形式 | 備考 |
|---|---|
| `#rgb`, `#rrggbb` | 16 進。`#rgba` と `#rrggbbaa` は拒否されます。 |
| `rgb(r, g, b)`, `rgba(r, g, b, a)` | 各成分は 0–255 またはパーセント。区切りはカンマ・空白・`/`。範囲外は丸められます。アルファ値は**無視**されます。 |
| 名前付き色 | `black`, `white`, `navy`, `red`, `blue`, `green`, `gray`, `grey`, `silver`, `maroon`, `purple`, `yellow`, `orange`, `lime`, `aqua`, `cyan`, `fuchsia`, `magenta`, `teal`, `olive`, `brown`, `pink`, `lightgray`, `lightgrey`, `darkgray`, `darkgrey` |

未対応: `hsl()`、色の値としての `transparent`、`currentColor`、その他すべての CSS 色名。文字色の既定は黒（`color` の初期値）です。

## 7. 対応 CSS プロパティ

ここに載っていないプロパティは無視されます。

| プロパティ | 値 | 備考 |
|---|---|---|
| `display` | 任意のキーワード | レイアウトが認識するのは `none`、`block`、`list-item`、`table`、`table-row-group`、`table-row`、`table-cell`、`table-caption`。それ以外の値（`inline`、`inline-block`、`flex`、`grid` など）はすべてインライン扱い。値の検証はありません。 |
| `font-size` | 長さ、または `xx-small` `x-small` `small` `medium` `large` `x-large` `xx-large` `smaller` `larger` | キーワードは 9, 10, 13, 16, 18, 24, 32px。`smaller` = 0.8333em、`larger` = 1.2em。単位なしの数値や負の値は無視。継承されます。 |
| `font-weight` | `bold`, `bolder`, `normal`, `lighter`, 数値 | `bold`/`bolder` = 700、`normal`/`lighter` = 400。数値は 700 以上なら太字、未満なら通常（フェイスは 2 種類のみ）。継承されます。 |
| `font-family` | カンマ区切りのファミリー名 | 各名前から前後の空白と引用符（`'` または `"`）を取り除きます。名前は先頭から順に、インストール済みフォントのファミリー名（大文字小文字を区別せず、name テーブルのみ。ファイル名では照合しない）と照合され、最初に一致したものが使われます（太字なら その太字フェイス）。汎用名 `monospace` は、Webブラウザの等幅フォントのデフォルトと同じ考え方で、OSと言語環境によって解決されます。日本語の Windows では MS Gothic、英語の Windows では Courier New、日本語の macOS では Osaka-Mono、英語の macOS では Courier、Linux では fontconfig の一般的な選択（日本語なら Noto Sans Mono CJK JP、それ以外は DejaVu Sans Mono や Liberation Mono）です。言語環境は LC_ALL、LC_CTYPE、LANG の順に読み、判定できない場合はその OS の日本語候補と欧文候補の両方を試します。ブラウザのデフォルト候補がどれもインストールされていない場合は、ファミリー名に "Mono" を含むインストール済みフォント（日本語対応のものを優先）を使い、等幅フォントが全く無ければ基本フォントにフォールバックします。解決された等幅フォントに日本語グリフが無い場合、そのフォントで描く日本語は表示されません。それ以外の汎用名（`serif`、`sans-serif`、`cursive`、`fantasy`、`system-ui`、`ui-serif`、`ui-sans-serif`、`ui-monospace`、`ui-rounded`）は読み飛ばされます。一致がなければ基本フォントを使います。継承されます。 |
| `color` | 色 | 継承されます。罫線色の既定値でもあります。 |
| `background-color` | 色 | ブロック、表、行（`tr`）、セルに描かれます。インライン要素にも、行ごとにテキストの背面へ描かれます。 |
| `background` | ショートハンド | 最初に見つかった色のトークンだけを使い、他（画像・位置など）は無視します。値が `transparent` / `none` なら背景をクリアします。 |
| `text-align` | `left`, `right`, `center` | `justify` と `start` は `left`、`end` は `right` になります。継承されます。親より狭い表の中央寄せにも使われます。 |
| `text-decoration`, `text-decoration-line` | `underline` または `none` を含む値 | 下線のみ対応。`line-through` と `overline` は無視されます。継承されます。 |
| `line-height` | `normal`, 数値, 長さ | 数値はフォントサイズの倍率、長さ（`%` を含む）は絶対値。継承されます。 |
| `margin` | 1〜4 個の長さ、`auto` | `auto` は 0 として扱われます（自動中央寄せはありません）。負の値も可。 |
| `margin-top`, `margin-right`, `margin-bottom`, `margin-left` | 長さ、`auto` | `margin` の各辺のロングハンド。 |
| `padding` | 1〜4 個の長さ | 負の値は 0 として扱われます。 |
| `padding-top`, `padding-right`, `padding-bottom`, `padding-left` | 長さ | `padding` の各辺のロングハンド。 |
| `width` | 長さ、`auto` | ブロックでは content 幅。表では表自身の padding と罫線を含む幅。パーセントは包含ブロックの幅が基準。 |
| `border` | 太さ・スタイル・色（順不同） | 4 辺すべてに設定。太さ省略 = `medium`（3px）、色省略 = 要素の `color`、スタイル省略 = `none` なので、**`border: 1px red` は何も描きません**。 |
| `border-top`, `border-right`, `border-bottom`, `border-left` | `border` と同じ | 1 辺だけ。 |
| `border-width` | `thin`（1px）、`medium`（3px）、`thick`（5px）、長さを 1〜4 個 | |
| `border-style` | `none`, `hidden`, `solid`, `dashed`, `dotted`, `double`, `groove`, `ridge`, `inset`, `outset` を 1〜4 個 | `hidden` は `none`。`groove`・`ridge`・`inset`・`outset` は `solid` として描かれます。1 つでも不正な値があれば宣言全体が無視されます。 |
| `border-color` | 色を 1〜4 個 | 1 つでも不正な値があれば宣言全体が無視されます。 |
| `border-top-width`, `border-right-width`, `border-bottom-width`, `border-left-width` | 太さ | `border-width` の各辺のロングハンド。 |
| `border-top-style`, ... `border-left-style` | スタイル | `border-style` の各辺のロングハンド。 |
| `border-top-color`, ... `border-left-color` | 色 | `border-color` の各辺のロングハンド。 |
| `border-radius` | 長さ 1〜4 個、必要なら `/` に続けてさらに 1〜4 個 | 水平半径、`/` の後に垂直半径（省略時は水平と同じ）。順序は左上・右上・右下・左下。負の値は無視。パーセントはボックスの幅（X）・高さ（Y）が基準。 |
| `border-top-left-radius`, `border-top-right-radius`, `border-bottom-right-radius`, `border-bottom-left-radius` | 長さ [長さ] | `border-radius` の各隅のロングハンド。 |
| `page-break-after`, `break-after` | `always`, `page`, `left`, `right`, `auto`, `avoid` | 最初の 4 つは要素の後ろで強制改ページ、`auto` と `avoid` は取り消し。それ以外の値は無視。継承されません。 |

**ショートハンドとロングハンド。** 1〜4 個の値を取るショートハンド（`margin`、`padding`、`border-width`、`border-style`、`border-color`）は、上・右・下・左の順に展開されます。1 個なら全辺、2 個なら上下 / 左右、3 個なら 上 / 左右 / 下です。`border` は全辺の太さ・スタイル・色を設定し、それ以前のロングハンドを上書きします。後から書いたロングハンドは、1 辺の 1 要素だけを上書きします。`border-radius` も同様に、左上・右上・右下・左下の順に展開されます。

**罫線と角丸の描画。** 各辺は罫線の太さの中心に線として描かれます。角丸がある場合、4 辺の太さと色が同じで、すべて `solid` の罫線は、角丸の 1 本の輪郭線として描かれます。太さ・色・スタイルが辺ごとに異なる場合や、`dashed`・`dotted`・`double` の辺がある場合は、直線 4 本が描かれ、角丸は背景の形にだけ反映されます。`double` は太さの 3 分の 1 の線 2 本で描かれます。半径がボックスに収まらないときは全体が縮小されます。表のセルは `border-radius` に対応しません。

## 8. 組み込み（UA）の既定スタイル

| セレクタ | 既定値 |
|---|---|
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `template`, `object`, `embed`, `svg` | `display: none` |
| `iframe` | `display: block` |
| `body`, `div`, `p`, `h1`–`h6`, `blockquote`, `pre`, `center`, `section`, `article`, `header`, `footer`, `address`, `page-break`, `ul`, `ol` | `display: block` |
| `li` | `display: list-item` |
| `table` | `display: table` |
| `thead`, `tbody`, `tfoot` | `display: table-row-group` |
| `tr` | `display: table-row` |
| `td`, `th` | `display: table-cell` |
| `caption` | `display: table-caption`（描画されない） |
| `body` | `margin: 8px` |
| `p`, `blockquote` | `margin: 1em 0` |
| `h1` | `font-size: 2em`、太字、`margin: 0.67em 0` |
| `h2` | `font-size: 1.5em`、太字、`margin: 0.83em 0` |
| `h3` | `font-size: 1.17em`、太字、`margin: 1em 0` |
| `h4` | 太字（1em）、`margin: 1.33em 0` |
| `h5` | `font-size: 0.83em`、太字、`margin: 1.67em 0` |
| `h6` | `font-size: 0.67em`、太字、`margin: 2.33em 0` |
| `ul`, `ol` | `margin: 1em 0`、`padding-left: 40px` |
| `ul ul`, `ul ol`, `ol ul`, `ol ol` | `margin: 0` |
| `b`, `strong` | 太字 |
| `u`, `ins` | 下線 |
| `a` | `color: blue`、下線 |
| `center` | `text-align: center` |
| `th` | 太字、`text-align: center` |
| `td`, `th` | `padding: 1px` |
| `page-break` | `page-break-after: always` |
| `code`, `kbd`, `samp`, `tt` | `font-family: monospace`、`background-color: #f0f0f0` |
| `pre` | `font-family: monospace` |

初期フォントサイズは 16px（12pt）です。これらのルールにない要素（例: `nav`、`main`、`aside`、`figure`、`dl`、`dt`、`dd`、`form`、`hr`、`fieldset`）はインラインになり、前後の改行や余白は付きません。`hr` は何も描きません。

## 9. レイアウトの挙動

**ブロック。** `width` は content 幅で、padding と罫線はその外側に加わります。縦方向の margin は隣り合うブロック同士で相殺されます（大きい方が採用）。親子間の margin 相殺はなく、負の margin は相殺されません。`height`、`min-*`、`max-*` は存在しない（ただし `iframe` のボックスでは `height` が効く）ため、高さは常に内容で決まります。背景や罫線を持つブロックは、1 ページに収まるなら 1 ページにまとめられます。

**インライン内容。** インライン要素が影響するのは、テキストの `color`、`font-size`、`font-weight`、下線、`line-height` だけです。インライン要素の margin、padding、罫線は無視されますが、`background-color` はテキストの背面に描かれます（折り返す場合は行ごとに分割されます）。空白（改行を含み、`pre` の中でも）は 1 個の空白に畳まれ、行頭の空白は捨てられます。`&nbsp;` は保持され、そこでは改行されません。インライン内容の中のブロック要素は新しい行を始めます。

**改行と禁則処理。**
- 空白の位置と、CJK 文字（CJK 漢字・かな、ハングル、CJK 互換文字・全角形、U+20000–3FFFF）の間で改行されます。
- 欧文の単語は、1 語が行幅を超える場合を除き、途中で分割されません。
- 行頭禁則文字: `、。，．・：；？！ー〜）〕］｝」』】〉》`、小書きのかな（`ゃゅょぁぃぅぇぉっャュョァィゥェォッ`）、`ゝゞ々`、および `, . : ; ? ! ) ] }`。最大 2 文字までは前の行の末尾にぶら下がり、それ以上は改行位置が前に移ります。
- 行末禁則文字: `（〔［｛「『【〈《` と `( [ {`。
- `word-break`、`overflow-wrap`、`white-space`、ハイフネーションは未対応です。

**リスト。** `ul` の項目は入れ子の深さに関係なく `• ` のマーカーが付きます（`list-style` は未対応）。`ol` の項目は、その項目までの `ol` の `display: list-item` の子を数えて `1. `、`2. `、… と番号が付きます（`start` と `reversed` は未対応）。親が `ol` でない `li` には黒丸が付きます。マーカーは項目の最初の行に 1 回だけ、項目の content ボックスの左側（リストの padding 部分）に、項目のフォントサイズ・太さ・色で描かれます。インライン内容のない項目にはマーカーが付きません。

**画像。** `img` はインラインで、下端がテキストのベースラインに揃います。既定のサイズは、画像（ファイル、またはデコードした `data:` URI）のピクセルサイズ × 0.75 です。CSS または属性の `width` で幅を指定でき、高さは縦横比から決まります（`height` は無視）。1 辺が 8192 ピクセルを超える、または 3300 万ピクセルを超える画像、読み込めない・不正な・非対応（SVG を含む）の画像は黙ってスキップされます（代替表示も `alt` テキストもありません）。

**表。**
- セルは左から順に配置され、`colspan` で列を結合できます。`rowspan` はありません。列数が 10000 を超える表や、レイアウトの作業量が固定の予算を超える文書（たとえば深くネストした表）は、変換がエラーになります。
- 列幅: セルの `width`（属性を含む）でその列が固定されます（content 幅 + padding + 罫線）。`%` 指定は、表に `width` がある場合、または `width` 未指定の表が縮小を迫られた場合に使われます。それ以外の列は最大内容幅になり、余りは均等に配分され、足りないときは最小内容幅に向けて縮みます。結合セルは、幅が足りない列だけを広げます。
- `width` のない表は内容に合わせた幅で、利用可能幅が上限です。`width` があり、すべての列が固定幅のときは、表幅いっぱいに比例拡大されます。
- セルの内容は常に垂直方向の中央に置かれます。行の高さは最も高いセルで決まります。
- 罫線はセルごとに描かれ、`border-collapse` や `border-spacing` はないため、隣り合うセルの罫線は統合されません。
- 表は、（継承された）`text-align` が `center` で、かつ包含ブロックより狭いときに中央寄せされます。`margin: auto` では中央寄せされません。

**改ページ。**
- テキストの行（インライン画像は行の中に含まれます）、表の行、1 ページに収まる罫線・背景付きのボックスは、ページをまたいで分割されません。1 ページより高い行やボックスも分割されず、はみ出した部分は描画されません。
- `page-break-after` / `break-after` は、ブロックまたは表の後ろで強制的に改ページします。表のセルの内側、インライン要素、表の行では無視されます。`page-break-before`、`page-break-inside`、`break-before`、`break-inside` は未対応です。
- `<page-break>` は強制改ページ専用の要素で、後ろの内容が次のページから始まります。`page-break-after: always`（8 章）を使って実現しているため、制限も同じです。閉じていない `<page-break>` の後ろの内容はパーサーによって要素の子になってしまうため、子要素を順序を保ったまま要素の直後へ移します。したがって `<p>a</p><page-break><p>b</p>` と書いても `b` は次のページになります。文書の先頭・末尾に置いた場合や連続して置いた場合も、空白ページはできません。インライン要素の中に置いた `<page-break>` は改ページしません。
- テキストも画像も含まないページは破棄されるため、空白ページは作れません。

**テキスト。** 基本のフェイスは通常と太字の 2 種類で、変換時に見つかった日本語対応フォント 1 つから選ばれます（Noto Sans CJK / Noto Sans JP を優先）。`FontFile` / `FontFileBold` を指定した場合はそのファイルが使われます。太字のフェイスがなければ、太字も通常のフェイスで描かれます。`font-family` では、ファミリー名でほかのインストール済みフォントも選べます（7 章）。インストール済みフォントは、初めて名前検索したときにフォントディレクトリ（`FontDirs`、なければ標準のディレクトリ）を走査して索引化されます。フォントコレクション（`.ttc`）は 1 つのフェイスがファイルを代表します。斜体はありません。

## 10. 未対応

- フォントとテキスト: `font-style`（斜体）、`font` ショートハンド、`letter-spacing`、`word-spacing`、`text-indent`、`text-transform`、`white-space`、`vertical-align`、`text-shadow`、`line-through` / `overline`、`justify`（両端揃え）。
- ボックス: `height`（`iframe` を除く）、`min-width`、`max-width`、`min-height`、`max-height`、`box-sizing`、`overflow`、`outline`、`box-shadow`、`opacity`、`visibility`、`margin: auto` による中央寄せ。
- レイアウト: `float`、`clear`、`position`（`absolute`、`fixed`、`relative`、`sticky`）、`z-index`、flex、grid、マルチカラム、`inline-block` の挙動。
- 表: `rowspan`、`border-collapse`、`border-spacing`、`cellspacing` / `cellpadding`、表の `caption`、`colgroup` / `col`、セルの `vertical-align`、`table-layout`、セルの `border-radius`。
- リスト: `list-style`、`list-style-type`、`ol` の `start`、独自マーカー。
- 背景: `background-image`、グラデーション、`background-position`、`background-repeat`。
- セレクタとルール: 属性セレクタ、擬似クラス、擬似要素、`>` `+` `~`、`@media`、`@import`、`@page`、`@font-face`、`!important` による優先、`inherit` / `initial`、`var()`、`calc()`。
- その他: リモート CSS、リモートのフレーム、リモート画像、SVG、リンク（`href`）、フォーム、JavaScript、`page-break-before` / `page-break-inside`、ヘッダー・フッター、ページ番号。

## 11. 変換オプション

### 警告

`-warn` は未対応の CSS プロパティ（`css-property`）、値（`css-value`）、at-rule（`css-at-rule`）、効果のない HTML 属性（`html-attribute`）、無視されたリモートリソース（`remote-resource`）、使用できない画像（`image-decode`）、不足するグリフ（`missing-glyph`）を報告します。対応 CSS の一覧は `internal/css/supported.go` で管理します。組み込みスタイルシート自体は警告を出しません。

警告は種類とキーで集計し、その順にソートして `html2pdf: warning: unsupported CSS property: font-style (3 times)` の形式で標準エラーに表示します（`md2pdf` では `md2pdf:`）。1 回の場合は回数を省略します。警告があっても変換は続き、成功時の終了コードは 0 です。`-warn` は `-verbose` と独立しており併用できます。`-warn-error` は今後の検討事項です。

### ライブラリ（`html2pdf.Options`）

| フィールド | 型 | 意味 |
|---|---|---|
| `PageSize` | string | `A4`（595.28 × 841.89pt）、`A5`（419.53 × 595.28pt）、`Letter`（612 × 792pt）。大文字小文字は区別しません。空なら `A4`。それ以外の名前はエラー。 |
| `PageWidth`, `PageHeight` | float64（pt） | 任意のサイズ。両方が 0 より大きいときはそのまま使われ、`PageSize` と `Landscape` は無視されます。片方だけ指定したときは、もう片方を `PageSize` から取ります。 |
| `Landscape` | bool | `PageSize` のサイズの幅と高さを入れ替えます。 |
| `BaseDir` | string | 相対パス（画像・スタイルシート・フレーム文書）を解決する基準ディレクトリ。`AllowExternalFiles` を指定しない限り、これらのファイルはこのディレクトリの内側になければなりません。 |
| `AllowExternalFiles` | bool | 文書が `BaseDir` の外のファイルを、絶対パスや `..` で参照することを許可します。既定は false で、画像・スタイルシート・フレーム文書は `BaseDir` の内側からしか読めません（`..` やシンボリックリンクで外へ出ることはできません。絶対パスは `BaseDir` の内側を指す場合のみ使えます）。`//` や `\\` で始まるパス（ネットワークパス）は、どちらのモードでも拒否されます。信頼できる文書でのみ有効にしてください。 |
| `FontDirs` | []string | フォントを探すディレクトリ（深さ 4 階層まで、`.ttf`、`.ttc`、`.otf`）。指定したディレクトリだけを探し、空なら標準のシステムフォントディレクトリ。ライブラリは環境変数も設定ファイルも読みません。使えるフォントが見つからないとエラーを返します。`font-family` の名前検索の対象にもなります。 |
| `OnFontUse` | func(FontUse) | 変換中に解決されたフォントのフェイスごとに、重複を除いて 1 回呼ばれます。`FontUse` は `Family`（一致した `font-family` の名前。照合用に正規化され小文字になります。既定のフェイスなら空）、`Bold`（太字が要求されたことを表します。太字フェイスが無い場合、`Path` は通常体のファイルで、`Bold` は true のままです）、`Path`（フォントファイル）を持ちます。nil ならコールバックなし。 |
| `FontFile` | string | 自動選択の代わりに通常フェイスとして使うフォントファイルのパス。読み込めない・解析できない場合、`Convert` はエラーを返します。空なら自動選択。 |
| `FontFileBold` | string | 太字フェイスのファイルパス。`FontFile` だけを指定した場合、太字は通常フェイスで代用されます。読み込めないファイルはエラーです。 |
| `Scale` | float64 | 内容を縮小（1 未満）・拡大（1 超）します。0 以下は 1.0 として扱い、NaN と無限大はエラー。 |
| `Margin` | float64（pt） | 4 辺共通のページ余白。0 は余白なし（`body` 自身の 8px の margin は残ります）。 |
| `MarkdownCSS` | string | `DefaultMarkdownCSS` の後ろに追加される CSS。`ConvertMarkdown` と `ConvertMarkdownFile` だけが使います。[markdown-ja.md](markdown-ja.md) を参照。 |

`Options.OnWarning` を設定すると、変換後（エラー時も）呼び出し元の goroutine で集計済み `Warning`（`Kind`、`Key`、`Message`、`Count`）を受け取れます。`Warning.String()` で整形でき、コールバックが nil なら収集しません。PNG と Markdown の変換にも適用されます。

関数: `Convert(src []byte, w io.Writer, opts *Options) error` と `ConvertFile(htmlPath, pdfPath string, opts *Options) error`（`opts` は nil 可。`ConvertFile` は変換が成功したときだけファイルを書き出します）。

### CLI

```
html2pdf [flags] input.html
html2pdf [flags] -e '<html>'
```

| フラグ | 既定値 | 意味 |
|---|---|---|
| `-e` | なし | 入力ファイルの代わりに直接指定する HTML ソース。カレントディレクトリの `output.html` にも書き出されます。相対パスはカレントディレクトリを基準に解決されます。入力ファイルとは同時に指定できません。 |
| `-o` | 入力ファイル名の拡張子を `.pdf` にしたもの（`-e` のときは `output.pdf`） | 出力 PDF のパス。 |
| `-page-size` | `A4` | `A4`、`A5`、`Letter`。 |
| `-page-width` | `0` | ページ幅（mm、`Options.PageWidth`）。`-page-size` の幅を上書きします。`0` なら名前付きサイズのままです。正の値のみ指定できます。 |
| `-page-height` | `0` | ページ高さ（mm、`Options.PageHeight`）。`-page-size` の高さを上書きします。`0` なら名前付きサイズのままです。正の値のみ指定できます。 |
| `-landscape` | オフ | 横向き。 |
| `-scale` | `1.0` | 内容の拡大縮小率。 |
| `-margin` | `0` | 4 辺共通のページ余白（pt）。 |
| `-font` | 自動検出 | 通常フェイスのフォントファイル（`Options.FontFile`）。 |
| `-font-bold` | 通常フォント | 太字フェイスのフォントファイル（`Options.FontFileBold`）。 |
| `-font-dirs` | なし | 先に探すフォントディレクトリのカンマ区切り一覧。ほかのフォントディレクトリの前に追加されます（下記参照）。 |
| `-allow-external-files` | オフ | 文書が自身のフォルダの外のファイルを読むことを許可します（`Options.AllowExternalFiles`）。信頼できる文書専用です。 |
| `-format` | 自動 | 出力形式: `pdf` または `png`。このフラグを省略すると、`-o` の名前が `.png` で終わる場合は PNG、それ以外は PDF になります。`png` ではページごとに PNG 画像を書き出し、複数ページの場合は `.png` の前に `-1`、`-2`、... が付きます。 |
| `-png-scale` | 2 | `-format png` のラスタライズ倍率。72dpi のポイントに対する倍率です（2 = 144dpi）。 |
| `-warn` | オフ | 未対応または無視された項目を標準エラーに表示します。 |
| `-verbose` | オフ | 検索したフォントディレクトリと使われたフォントを標準エラーに表示します。 |

CLI はフォントディレクトリを集め、次の順で `Options.FontDirs` に渡します。`-font-dirs`、環境変数 `HTML2PDF_FONT_DIRS`（OS のパスリスト形式。Unix は `:`、Windows は `;` 区切り）、OS のフォントディレクトリの順です。先に集めたものが優先され、重複したディレクトリは最初の位置だけが残ります。`-font-dirs` は標準のディレクトリを置き換えるのではなく、先に探すディレクトリを追加します。`-verbose` を付けると、変換前に各ディレクトリを `font dir [flag|env|os]: <path>` の形で、変換後に解決された各フォントを `font: <family or (default)> (bold) -> <path>` の形で表示します。

CLI は `BaseDir` に入力ファイルのあるディレクトリを設定します。`PageWidth`、`PageHeight` には `-page-width`、`-page-height` の値をミリメートルからポイントに換算して設定します。フラグは入力ファイルの前後どちらにも置けます。
