# html2pdf-go タスクランナー（https://github.com/casey/just）

# 引数なしでレシピ一覧を表示
default:
    @just --list

# テスト
test:
    go test ./...

# ビルド
build:
    go build ./...

# go vet
vet:
    go vet ./...

# gofmt で整形
fmt:
    gofmt -w .

# vet + test をまとめて実行
check: vet test

# IPAex フォントを fonts/ に取得（既にあれば何もしない）
# ダウンロード元: https://moji.or.jp/wp-content/ipafont/IPAexfont/IPAexfont00401.zip
fonts:
    #!/usr/bin/env sh
    set -eu
    if [ -f fonts/ipaexg.ttf ] && [ -f fonts/ipaexm.ttf ]; then
        echo "fonts/: IPAex フォントは取得済みです"
    else
        mkdir -p fonts
        tmp=$(mktemp -d)
        trap 'rm -rf "$tmp"' EXIT
        curl -fsSL -o "$tmp/ipaexfont.zip" https://moji.or.jp/wp-content/ipafont/IPAexfont/IPAexfont00401.zip
        # -j: zip 内のフォルダ階層を潰して fonts/ 直下に展開
        # unzip のワイルドカードの * は / にもマッチするので、階層の有無を問わず取り出せる
        unzip -o -j "$tmp/ipaexfont.zip" '*ipaexg.ttf' '*ipaexm.ttf' '*IPA_Font_License_Agreement_v1.0.txt' '*Readme_IPAexfont00401.txt' -d fonts
    fi

# ダミー請求書 PDF を out/invoice.pdf に生成
invoice:
    mkdir -p out
    go run ./cmd/html2pdf testdata/invoice.html -o out/invoice.pdf

# サンプル一式を out/ に生成
samples: invoice fonts
    mkdir -p out
    go run ./cmd/html2pdf examples/address-book.html -o out/address-book.pdf -landscape -margin 36
    go run ./cmd/html2pdf examples/border-styles.html -o out/border-styles.pdf -margin 48
    go run ./cmd/html2pdf examples/font-family.html -o out/font-family.pdf -margin 48
    go run ./cmd/html2pdf examples/receipt.html -o out/receipt.pdf -margin 48 -font-dirs fonts
    go run ./cmd/md2pdf examples/sample.md -o out/sample-md.pdf -margin 48 -font-dirs fonts

# Markdown サンプル PDF を out/sample-md.pdf に生成（fonts が必要）
markdown: fonts
    mkdir -p out
    go run ./cmd/md2pdf examples/sample.md -o out/sample-md.pdf -margin 48 -font-dirs fonts

# 領収書 PDF を out/receipt.pdf に生成（fonts が必要）
receipt: fonts
    mkdir -p out
    go run ./cmd/html2pdf examples/receipt.html -o out/receipt.pdf -margin 48 -font-dirs fonts

# 各 OS 向けのリリース用バイナリを dist/ に生成（例: just release-binaries v0.1.0）
release-binaries version='dev':
    sh scripts/build-release.sh {{version}}

# 生成物 out/ と dist/ を削除
clean:
    rm -rf out dist
