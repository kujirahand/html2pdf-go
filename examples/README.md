# サンプル文書カタログ

html2pdf-go で作れる PDF のサンプル集です。各サンプルの入力ファイル（HTML / Markdown）と、変換後の PDF へのリンクをまとめました。PDF は `just examples` で `examples/pdf/` に生成されます（このフォルダの PDF は生成済みのものです）。サンプルに登場する人名・団体名・金額などは、すべて架空のものです。

## HTML サンプル

| サンプル | 内容 | 入力ファイル | PDF |
| --- | --- | --- | --- |
| 住所録 | 横向き A4 に表組みで並べた住所録 | [address-book.html](address-book.html) | [address-book.pdf](pdf/address-book.pdf) |
| 領収書 | 金額欄と宛名のある領収書 | [receipt.html](receipt.html) | [receipt.pdf](pdf/receipt.pdf) |
| 罫線スタイル見本 | dashed / dotted / double などの罫線一覧 | [border-styles.html](border-styles.html) | [border-styles.pdf](pdf/border-styles.pdf) |
| フォント見本 | フォントファミリーの表示見本 | [font-family.html](font-family.html) | [font-family.pdf](pdf/font-family.pdf) |
| イベントチラシ | 余白なし全面レイアウトのライブ案内チラシ | [日本語](event-flyer-ja.html) / [English](event-flyer-en.html) | [日本語](pdf/event-flyer-ja.pdf) / [English](pdf/event-flyer-en.pdf) |
| コンサートチケット | ミシン目と半券のあるチケット | [日本語](concert-ticket-ja.html) / [English](concert-ticket-en.html) | [日本語](pdf/concert-ticket-ja.pdf) / [English](pdf/concert-ticket-en.pdf) |
| カフェメニュー | 2 段組風のカフェメニュー表 | [日本語](cafe-menu-ja.html) / [English](cafe-menu-en.html) | [日本語](pdf/cafe-menu-ja.pdf) / [English](pdf/cafe-menu-en.pdf) |
| 名刺 | 91×55mm の名刺（表裏） | [日本語](business-card-ja.html) / [English](business-card-en.html) | [日本語](pdf/business-card-ja.pdf) / [English](pdf/business-card-en.pdf) |
| カレンダー | 2026 年 12 月の月間カレンダー | [日本語](calendar-ja.html) / [English](calendar-en.html) | [日本語](pdf/calendar-ja.pdf) / [English](pdf/calendar-en.pdf) |
| 搭乗券 | バーコード付きの航空券 | [日本語](boarding-pass-ja.html) / [English](boarding-pass-en.html) | [日本語](pdf/boarding-pass-ja.pdf) / [English](pdf/boarding-pass-en.pdf) |

## Markdown サンプル

Markdown から変換したサンプルです。PDF のファイル名には `-md` が付きます。

| サンプル | 内容 | 入力ファイル | PDF |
| --- | --- | --- | --- |
| Markdown 基本 | 見出し・リスト・表・コードの基本要素 | [sample.md](sample.md) | [sample-md.pdf](pdf/sample-md.pdf) |
| 議事録 | 会議の議事録（決定事項・宿題つき） | [日本語](meeting-minutes-ja.md) / [English](meeting-minutes-en.md) | [日本語](pdf/meeting-minutes-ja-md.pdf) / [English](pdf/meeting-minutes-en-md.pdf) |
| レシピ | 材料表と手順のあるレシピ | [日本語](recipe-ja.md) / [English](recipe-en.md) | [日本語](pdf/recipe-ja-md.pdf) / [English](pdf/recipe-en-md.pdf) |
| 旅行日程表 | 2 泊 3 日の旅程表 | [日本語](travel-itinerary-ja.md) / [English](travel-itinerary-en.md) | [日本語](pdf/travel-itinerary-ja-md.pdf) / [English](pdf/travel-itinerary-en-md.pdf) |
| 技術記事 | コードブロックを含む技術記事 | [日本語](tech-article-ja.md) / [English](tech-article-en.md) | [日本語](pdf/tech-article-ja-md.pdf) / [English](pdf/tech-article-en-md.pdf) |
| 週報 | 週次報告書 | [日本語](weekly-report-ja.md) / [English](weekly-report-en.md) | [日本語](pdf/weekly-report-ja-md.pdf) / [English](pdf/weekly-report-en-md.pdf) |
| 献立表 | 1 週間の献立表 | [日本語](meal-plan-ja.md) / [English](meal-plan-en.md) | [日本語](pdf/meal-plan-ja-md.pdf) / [English](pdf/meal-plan-en-md.pdf) |
| カフェメニュー | Markdown 版のカフェメニュー | [日本語](cafe-menu-ja.md) / [English](cafe-menu-en.md) | [日本語](pdf/cafe-menu-ja-md.pdf) / [English](pdf/cafe-menu-en-md.pdf) |
| カレンダー | Markdown 版の月間予定表 | [日本語](calendar-ja.md) / [English](calendar-en.md) | [日本語](pdf/calendar-ja-md.pdf) / [English](pdf/calendar-en-md.pdf) |
| イベントチラシ | Markdown 版のイベント案内 | [日本語](event-flyer-ja.md) / [English](event-flyer-en.md) | [日本語](pdf/event-flyer-ja-md.pdf) / [English](pdf/event-flyer-en-md.pdf) |

## 自分で変換するには

```sh
just examples   # examples/ の全サンプルを examples/pdf/ に PDF 化
```

個別に変換する場合は、各 CLI を直接使います。

```sh
html2pdf examples/receipt.html -o receipt.pdf
md2pdf examples/recipe-ja.md -o recipe.pdf
```
