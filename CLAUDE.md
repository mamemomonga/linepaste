# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 概要

`linepaste` は複数行テキストを1行ずつクリップボードにコピーするCLIツール。stdinまたはファイルからテキストを読み込み、Bubble Teaを使ったTUI上で行を選択してコピーできる。

対応OS: macOS（`pbcopy`）、Linux（`xclip` / `xsel` / `wl-copy`）

## コマンド

```bash
# ビルド（go mod tidy も実行される）
make build

# ~/bin/ にインストール
make install

# アンインストール
make uninstall

# 直接実行
go run . <file>
cat file | go run .
```

テストは存在しない。

## アーキテクチャ

コード全体が `main.go` 1ファイルに収まるシンプルな構造。

**TUIフレームワーク**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)（Elm アーキテクチャ）を使用。`model` 構造体が状態を保持し、`Init` / `Update` / `View` の3メソッドで動作する。

**主要な状態 (`model`)**:
- `lines []string` — 入力された全行
- `cursor int` — 現在選択中の行インデックス
- `topLine int` — ビューポートの先頭行インデックス
- `copiedLine int` — 最後にコピーした行（`-1` は未コピー）

**ビューポート制御** (`adjustViewport`): vim スタイルのスクロール。カーソルがビューポート外に出た時だけスクロールする。`visibleEnd` が折り返し行数を考慮して表示できる最終行を計算する。

**クリップボード** (`toClipboard`): OS に応じて外部コマンド（`pbcopy` / `xclip` / `xsel` / `wl-copy`）にパイプ渡しでコピーする。

**入力処理**: stdin がパイプ（非 CharDevice）なら stdin から読み込み、引数があればファイルから読む。TUI の入力は `/dev/tty` を直接開いて渡すことでパイプと共存させている。

**テキスト折り返し** (`wrapText`): バイト単位で折り返す（マルチバイト文字非考慮）。現在行はガター幅 + `▶ ` マーカー分だけ有効幅が狭くなる。
