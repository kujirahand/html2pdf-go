# AGENTS.md

このリポジトリで作業する AI エージェント（および開発者）向けの作業ルールです。
迷ったらこのファイルに従ってください。ユーザーからその場で別の指示があった場合は、そちらを優先します。

## プロジェクト概要

- `html2pdf-go` は、純 Go の HTML→PDF 変換エンジンです。HTML のパース、CSS の適用、レイアウト、PDF の書き出しまでを Go だけで行います。
- 公開 API はルートパッケージ `html2pdf`（`Convert` / `ConvertFile` / `Options`）だけです。エンジン本体は `internal/` に置き、API が安定するまでは非公開とします。
- 処理の流れ: `dom`（パース）→ `css`（スタイル適用）→ `layout`（レイアウト・ページ分割）→ `render`（描画）→ `pdf`（書き出し）。`text` はフォント計測と折り返しを担当し、`layout` と `render` から使われます。詳細は `docs/architecture-ja.md` を参照してください。

## ドキュメントの二言語ルール（最重要）

ドキュメントは英語版と日本語版をペアで管理します。**片方だけを変更・追加して終えることは禁止です。**

### 対象と命名規則

| 英語版（無印） | 日本語版（`-ja`） |
|---|---|
| `README.md` | `README-ja.md` |
| `docs/<名前>.md` | `docs/<名前>-ja.md` |

- 無印が英語版、`-ja` が付いたものが日本語版です。
- 現在のペア: `README.md` / `README-ja.md`、`docs/architecture.md` / `docs/architecture-ja.md`、`docs/reference.md` / `docs/reference-ja.md`、`docs/tutorial.md` / `docs/tutorial-ja.md`、`docs/markdown.md` / `docs/markdown-ja.md`

### 同時更新のルール

1. **`README.md` と `README-ja.md` は同時に更新すること。** 片方に加えた変更（チュートリアル、使い方、オプション、リンクなど）は、同じ変更の中で必ずもう片方にも反映する。
2. **`docs/*.md` と `docs/*-ja.md` は同時に作成・更新すること。** `docs/` に新しいドキュメントを作るときは、英語版と日本語版の両方を最初から作る。既存ドキュメントを直すときも両方を直す。
3. **内容は一致させること。** 英語版と日本語版で、見出し構成、記載する事実、コード例、コマンド、数値、対応表の中身が食い違ってはいけない。言語だけが違う状態を保つ。どちらか一方にしか書いていない情報があってはいけない。
4. リンクは各言語版の中で完結させる。英語版からは英語版ドキュメントへ、日本語版からは日本語版ドキュメントへリンクし、相互リンク（言語切替リンク）を先頭付近に置く。
5. コードブロック、コマンド、ファイルパス、識別子は翻訳せず、そのまま一致させる。
6. 作業の最後に、ペアの両ファイルを開いて見比べ、構成と内容が一致していることを確認してから完了とする。

このルールは README と `docs/` が対象です。それ以外の Markdown（例: `fonts/README.md`）は単独で構いませんが、新しく恒常的なドキュメントを置く場合は `docs/` にペアで置くことを検討してください。

## フォルダ構造と役割

| パス | 役割 |
|---|---|
| `html2pdf.go` | 公開 API。`Convert` / `ConvertFile` / `Options` の定義と、変換パイプラインの入口 |
| `html2pdf_test.go` | 公開 API のテスト |
| `markdown.go` | Markdown 変換の公開 API。`MarkdownToHTML` / `ConvertMarkdown` / `ConvertMarkdownFile` / `DefaultMarkdownCSS` と、`<pre>` の空白保持変換 |
| `markdown_test.go` | Markdown 変換のテスト |
| `cmd/html2pdf/` | CLI ツール本体（`main.go`）。フラグはライブラリの `Options` に対応させる |
| `cmd/md2pdf/` | Markdown 用 CLI（`main.go`）。`html2pdf` と同じフラグに `-css` を加えたもの |
| `internal/dom/` | HTML のパースと文書ツリーの構築 |
| `internal/css/` | CSS のパース、セレクタ照合、カスケード、計算済みスタイルの生成 |
| `internal/text/` | フォントの読み込み・選択・計測、文字列の折り返しと禁則処理 |
| `internal/layout/` | ボックス生成、ブロック／テーブルレイアウト、ページ分割（ページネーション） |
| `internal/cliio/` | CLI 用の標準入力判定と入出力ジョブの計画 |
| `internal/fontcfg/` | CLI 用のフォントフォルダ収集（`-font-dirs` → 環境変数 `HTML2PDF_FONT_DIRS` → OS 既定）。ライブラリは使わない |
| `internal/filex/` | 文書が参照するローカルファイル（画像・スタイルシート・フレーム文書）を読む唯一の入口。既定では `BaseDir` の内側に閉じ込め（`os.OpenRoot`）、通常ファイルのみ・サイズ上限つきで読む。`AllowExternalFiles` で外部参照を許可する |
| `internal/raster/` | レイアウト結果を PNG 画像にラスタライズする（`-format png` / `ConvertPNG`）。PDF 経路とは独立 |
| `internal/datauri/` | `data:` URI のデコード（base64／パーセントエンコード）。`layout` と `render` が画像ソースとして使う |
| `internal/render/` | レイアウト結果を `internal/pdf` 経由で描画する |
| `internal/pdf/` | PDF の書き出し。オブジェクト、コンテンツストリーム、フォント（Type3 サブセット埋め込み）・画像の埋め込み |
| `docs/` | 設計ドキュメント。二言語ペアで置く（上記ルール参照）。`architecture` は設計とパイプライン、`reference` は対応する HTML 要素・CSS の仕様一覧、`markdown` は Markdown 変換（md2pdf）の解説 |
| `examples/` | サンプル文書と使用例。`basic/` は最小の Go 使用例、`*.html` は変換サンプル（住所録、領収書、フォント見本など）、`sample.md` は md2pdf 用のダミー Markdown 文書 |
| `testdata/` | テスト・サンプル生成用の入力 HTML と素材（請求書 `invoice.html`、印影のダミー画像など） |
| `fonts/` | `just fonts` で取得する IPAex フォントの置き場。`README.md` 以外は git 管理対象外 |
| `out/` | 生成したサンプル PDF の出力先。git 管理対象外 |
| `bin/` | `just make` で生成する CLI バイナリの出力先。git 管理対象外 |
| `scripts/` | 開発・リリース用のシェルスクリプト。`build-release.sh` は両 CLI を 6 ターゲット（linux/darwin/windows × amd64/arm64）向けにクロスコンパイルし、アーカイブと `checksums.txt` を `dist/` に出力する。`install.sh`（Linux/macOS）と `install.ps1`（Windows）は、curl／irm 経由で最新リリースのバイナリを取得・SHA-256 検証・インストールする利用者向けインストーラ |
| `.github/workflows/` | CI／リリース自動化。`test.yml` は PR と main への push で `just test` を実行する（フォント依存テストが空回りしないよう、Noto CJK・DejaVu・Liberation を入れてから実行）。`release.yml` は `v*` タグの push（または手動実行）で `scripts/build-release.sh` を実行し、成果物を GitHub Release に添付する |
| `dist/` | リリース用アーカイブの出力先。git 管理対象外 |
| `justfile` | 開発用タスクランナーの定義（`test` / `check` / `fonts` / `samples` など） |
| `README.md` / `README-ja.md` | 利用者向けの入口ドキュメント（英語／日本語）。必ず同時に更新する |
| `AGENTS.md` | 本ファイル。エージェント向け作業ルール |
| `LICENSE` | MIT ライセンス |

フォルダ構成は README には載せず、この表で管理します。新しいフォルダや主要ファイルを追加したら、この表を更新してください。

## 開発の進め方

- タスクは `just` を使う。`just test`（`go test ./...`）、`just check`（vet + test）、`just samples`（サンプル PDF 一式を `out/` に生成）、`just fonts`（IPAex フォント取得）。
- 変更後は必ず `just check` を通し、表示に関わる変更では `just samples` でサンプル PDF を生成して目視確認する。
- 対応範囲（HTML 要素・CSS）を変えたら、`docs/reference.md` と `docs/reference-ja.md` の両方に反映する。設計を変えたら `docs/architecture.md` と `docs/architecture-ja.md` の両方に反映する。

## データに関する注意

- **実在の請求書・領収書などの実データは絶対に取り込まないこと。** 本人の氏名・住所・口座・印影・金額などは入れない。
- テストやサンプルには、`testdata/` や `examples/` にあるダミー文書（構造は実物と同じでも、内容はすべて架空のもの）を使う。新しくサンプルを作るときも架空のデータで作る。
- フォントのバイナリはコミットしない（`fonts/` は `just fonts` で取得する）。

## コミット・PR

- PR は作成してよいが、マージはユーザーの承認を得てから行う。
- ドキュメントだけの変更でも、二言語ペアが揃っていることを PR の説明に書く。
