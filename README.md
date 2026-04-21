# linepaste

複数テキストを一行ずつコピーするツール

## 機能

- stdinまたはファイルから行を読み込み、1行ずつクリップボードにコピー
- 現在行のハイライト + 前後の文脈表示
- 絶対行番号付き
- 長い行のターミナル幅での折り返し

## インストール

```bash
git clone ... && cd linepaste
make install
```

## 使い方

```bash
# ファイルから
linepaste commands.sh

# パイプで
cat commands.sh | linepaste

# ヒアドキュメントで
linepaste <<'EOF'
aws s3 ls
aws ec2 describe-instances --query 'Reservations[].Instances[].InstanceId'
echo "done"
EOF
```

## キーバインド

| キー | 動作 |
|------|------|
| Enter / c | 現在行をコピー |
| j / ↓ / Space / n | 次の行へ移動 |
| k / ↑ / p | 前の行へ移動 |
| g / Home | 先頭へ |
| G / End | 末尾へ |
| q / Esc / Ctrl+C | 終了 |

## ワークフロー

1. `linepaste` で行一覧を表示
2. Enter/c で現在行をクリップボードにコピー
3. CloudShellに Cmd+V で貼り付け → 実行
4. linepaste に戻って j で次の行へ移動、Enter でコピー
5. 繰り返し

## 対応OS

- macOS（pbcopy）
- Linux（xclip / xsel / wl-copy）

## License

MIT

## 備考

このツールはClaude Opus 4.6で開発しました。