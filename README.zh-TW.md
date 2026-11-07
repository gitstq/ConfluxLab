# ConfluxLab 🧬

> **零相依的 CRDT 收斂性工具箱與隨機併發驗證沙盤**
> 把「最終一致性」從一句口號，變成可以跑進 CI 的回歸測試。

[簡體中文](README.md) · **繁體中文** · [English](README.en.md)

[![CI](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml/badge.svg)](https://github.com/gitstq/ConfluxLab/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-%E2%89%A5%201.21-00ADD8)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Dependencies](https://img.shields.io/badge/runtime%20dependencies-0-brightgreen)](#-核心特性)

> 📦 **不想裝 Go？** 直接下載 [**v1.0.0 Release**](https://github.com/gitstq/ConfluxLab/releases/tag/v1.0.0)：Linux / macOS / Windows 五平台靜態執行檔，附 `SHA256SUMS.txt` 校驗和，開箱即用。

---

## 🎉 專案介紹

寫分散式系統時，我們常說「用 CRDT 就會自動收斂」。但**要怎麼證明它真的收斂**？靠讀論文，還是等線上出一次事故？

ConfluxLab 的答案是：**把收斂性變成可執行、可量測、可重播的指標**。

它是以純 Go 標準函式庫實作的程式庫，內建 5 種狀態型 CRDT 與向量時鐘；更關鍵的是附帶一個**確定性的隨機併發沙盤**——把多個副本的本機寫入、雙向同步、網路分區、同步受阻、訊息重複投遞交錯執行，接著自動癒合分區、逐副本比對快照，同時檢查單調性、墓碑完整性，以及交換律／結合律／冪等律。任何一條被破壞，程式就回傳非零結束碼，CI 直接失敗。

| 你想回答的問題 | ConfluxLab 給出的答案 |
| --- | --- |
| 訊息亂序會不會出錯？ | 交換律門禁：`a⊔b = b⊔a` |
| 不同的合併拓撲等價嗎？ | 結合律門禁：`(a⊔b)⊔c = a⊔(b⊔c)` |
| 訊息重傳安全嗎？ | 冪等律門禁：`a⊔a = a`，並在沙盤中真實重複投遞 |
| 分區恢復後副本一致嗎？ | 沙盤癒合後逐副本比對快照 |
| 偶發問題怎麼重現？ | 輸入 `-seed` 即可逐位重播整條排程序列 |

**解決的痛點**：CRDT 的缺陷往往只在「併發＋分區＋重傳」的特定交錯下才會浮現，手寫幾個單元測試根本覆蓋不到。ConfluxLab 用數百輪隨機排程取代人工窮舉，而且**失敗可精確重現**。

**自研差異化亮點**：市面上多數專案是「再實作一遍 CRDT」，本專案把重心放在**驗證器**——代數律檢查、不變量追蹤、確定性重播、機器可讀報告、CI 門禁，全部自研且零相依。

**靈感來源**：2026-11-08 GitHub Trending 上的 `cloudflare/security-audit-skill`（把安全結論做成可獨立驗證、機器可讀的發現）以及 `mbillow/localthings`（本機優先、不仰賴雲端）。前者啟發了「讓結論可被驗證」，後者啟發了「零相依、離線可跑」。本專案的程式碼、演算法與文件均為獨立自研，未複製任何參考專案的實作。

---

## ✨ 核心特性

- 🧬 **5 種狀態型 CRDT 全自研**：`GCounter`（只增計數器）、`PNCounter`（可增減計數器）、`LWWRegister`（最後寫入者獲勝）、`MVRegister`（多值暫存器）、`ORSet`（觀察-刪除集合）。
- 🕰️ **向量時鐘**：完整判定 `before / after / equal / concurrent` 四種因果序，為 `MVRegister` 提供併發版本識別。
- ⚖️ **三條代數律可執行驗證**：隨機產生狀態組，逐一檢查交換律、結合律、冪等律——它們是 join-semilattice 的定義性質，也是收斂性的充分條件。
- 🌐 **隨機併發沙盤**：本機寫入、雙向合併、分區切換、同步受阻、訊息重複投遞交錯進行；自動癒合後逐副本比對。
- 🎯 **確定性可重現**：自研 SplitMix64 產生器，不用 `math/rand` 全域來源，也不讓 `time.Now()` 參與排程決策；**同種子必得逐位一致的結果**，可用 `-json` 直接 diff 兩次執行。
- 🛡️ **衝突顯式化**：`LWWRegister` 用 `(時戳, 節點)` 全序決定勝者；`MVRegister` 保留所有併發版本交給上層；`Merge` 遇到同識別碼卻不同值時直接報錯，絕不靜默吞掉。
- 📊 **機器可讀報告＋門禁**：`-json` 輸出結構化報告，含每種類型的操作數、合併數、受阻數、分區次數、違反項與最終快照；任一異常結束碼為 1。
- 🪶 **零第三方相依**：執行期與測試期都只用 Go 標準函式庫，`go mod download` 無事可做。
- 📦 **5 個目標的靜態執行檔**：Linux (amd64/arm64)、macOS (amd64/arm64)、Windows (amd64)，`CGO_ENABLED=0`，附 SHA256 校驗和。

---

## 🚀 快速開始

**環境需求**：Go ≥ 1.21（或直接使用預先編譯好的執行檔），其餘無需安裝。

```bash
# 方式一：直接由原始碼執行，不必安裝任何套件
git clone https://github.com/gitstq/ConfluxLab.git
cd ConfluxLab
go run ./cmd/confluxlab verify

# 方式二：編譯成單一靜態執行檔
make build
./dist/confluxlab version

# 方式三：作為程式庫引入
go get github.com/gitstq/ConfluxLab
```

三條最常用的指令：

```bash
# ① 隨機併發沙盤：驗證多副本最終收斂
go run ./cmd/confluxlab verify -rounds 500 -replicas 5

# ② 代數律門禁：驗證交換／結合／冪等
go run ./cmd/confluxlab laws -rounds 300

# ③ 三個真實併發語意展示（add-wins、LWW 平手、衝突曝光）
go run ./cmd/confluxlab demo
```

首次執行 `verify` 的典型輸出：

```text
ConfluxLab · 隨機併發收斂性驗證
──────────────────────────────────────────────────────────────────────────────
  seed=20261108 rounds=300 replicas=4 localOps=661 merges=373 kinds=5 status=CONVERGED
──────────────────────────────────────────────────────────────────────────────
  類型             收斂           本機操作       合併       重複       受阻       分區     違反
  gcounter       ✅ 是           138       66        6       60       36      0
  pncounter      ✅ 是           141       71        7       54       34      0
  lwwregister    ✅ 是           131       77       17       58       34      0
  mvregister     ✅ 是           119       80       15       52       49      0
  orset          ✅ 是           132       79       14       60       29      0
──────────────────────────────────────────────────────────────────────────────
  結論: 全部收斂，三條代數律與全部不變量通過 ✅
```

---

## 📖 詳細使用指南

### `verify` — 隨機併發收斂性驗證

| 參數 | 預設值 | 說明 |
| --- | --- | --- |
| `-seed` | `20261108` | 隨機種子；**同種子必得逐位一致結果** |
| `-rounds` | `400` | 每種類型的排程事件數 |
| `-replicas` | `4` | 副本數量（≥ 2） |
| `-kinds` | `all` | 逗號分隔的類型，例如 `orset,mvregister` |
| `-json` | `false` | 輸出機器可讀的 JSON 報告 |
| `-timeline` | `false` | 輸出事件時間軸（含分區、受阻、重複投遞） |

```bash
# 只跑兩種類型，檢視分區時間軸
go run ./cmd/confluxlab verify -seed 42 -rounds 120 -kinds orset,mvregister -timeline

# 輸出 JSON，供指令碼取用或做確定性重播比對
go run ./cmd/confluxlab verify -rounds 200 -json > run-a.json
go run ./cmd/confluxlab verify -rounds 200 -json > run-b.json
diff run-a.json run-b.json   # 同種子應完全相同
```

時間軸範例（節錄）：

```text
  [orset]  收斂=true  本機操作=58  合併=20  重複=2  受阻=27  分區切換=15
  │ #001  分區切換  n1 → 分區 1
  │ #002  同步受阻  n1 ↔ n2  (分區 1|0)
  │ #003  雙向同步  n2 ↔ n3
  │ #006  本機寫入  remove(epsilon)@n1#1         → n1
  │ #010  本機寫入  add(beta)@n1#2               → n1
  │ #011  本機寫入  remove(beta)@n0#3            → n0
  │ ...  重複投遞 n2 ↔ n3  (驗證冪等)
  ──── 網路癒合：全連通反熵 ────
  │ 收斂=true  最終狀態={[alpha beta delta epsilon gamma] 38}
```

> 💡 **為什麼「受阻」很重要**：只在全連通下做合併，幾乎測不出問題。分區讓不同副本各自演化出**真正併發**的版本，癒合階段才是收斂性的考場。

### `laws` — 代數律門禁

```bash
go run ./cmd/confluxlab laws -seed 20261108 -rounds 300
```

```text
  類型                     檢查次數         違反 結果
  gcounter                900          0 ✅ 通過
  pncounter               900          0 ✅ 通過
  lwwregister             900          0 ✅ 通過
  mvregister              900          0 ✅ 通過
  orset                   900          0 ✅ 通過
  ────────────────────────────────────────────────────────────────
  交換律 a⊔b = b⊔a ｜ 結合律 (a⊔b)⊔c = a⊔(b⊔c) ｜ 冪等律 a⊔a = a
```

### `demo` — 三個真實語意

```text
  ① ORSet add-wins：分區兩側分別對同一元素執行 remove 與 add
     合併後可見元素 = [task]  → 併發 add 存活（新 tag 未被觀察到）
  ② LWWRegister 平手打破：相同時戳的兩次併發寫入
     兩個副本獨立合併後均得到 "來自 n1"（節點名字典序決定，結果確定）
  ③ MVRegister 不靜默丟棄衝突：兩個併發寫入同時保留
     併發版本 = [配置 A 配置 B]  → 由上層決策，而非悄悄覆蓋
```

### 作為程式庫使用

```go
package main

import (
    "fmt"
    "github.com/gitstq/ConfluxLab"
)

func main() {
    left, right := confluxlab.NewORSet(), confluxlab.NewORSet()
    left.Add("deploy", "n0:1")
    _ = right.Merge(left.Clone())
    left.Remove("deploy")       // n0 觀察到 tag n0:1
    right.Add("deploy", "n1:9") // 分區期間併發 add，tag 未被觀察

    _ = left.Merge(right.Clone())
    _ = right.Merge(left.Clone())
    fmt.Println(left.Elements()) // [deploy] —— add-wins
}
```

完整可執行範例請見 [`examples/basic/main.go`](examples/basic/main.go)。

### 結束碼

| 結束碼 | 含義 |
| --- | --- |
| `0` | 全部類型收斂、三條代數律與全部不變量通過 |
| `1` | 存在未收斂、不變量違反或執行錯誤 |
| `2` | 參數／用法錯誤 |

因此可直接寫進指令碼或 CI：

```bash
go run ./cmd/confluxlab verify -rounds 500 -replicas 5 || exit 1
```

### 常見問題

**Q：不同種子結果不同，是不是不穩定？**
A：恰好相反。種子決定整條排程序列，同種子逐位一致；多種子是用來擴大覆蓋率。CI 會同時跑多種子與傳統單元測試。

**Q：`MVRegister` 的快照裡出現多個值，是 bug 嗎？**
A：不是，這是設計取捨——它**不擅自丟棄**併發寫入，把衝突顯式交給上層。若需要自動選出勝者，請改用 `LWWRegister`。

**Q：新增自己的 CRDT 會被驗證器覆蓋嗎？**
A：會。依 [`CONTRIBUTING.md`](CONTRIBUTING.md) 的清單登錄到 `AllKinds()` 後，`laws` 與 `verify` 會自動納入。

---

## 💡 設計思路與迭代規劃

### 三條代數律為什麼是收斂性的根基

狀態型 CRDT 的 `Merge` 本質上是偏序集合上的**最小上界（join）**。只要它滿足：

| 定律 | 含義 | 對應的現實問題 |
| --- | --- | --- |
| 交換律 `a⊔b = b⊔a` | 結果與合併順序無關 | 訊息亂序送達 |
| 結合律 `(a⊔b)⊔c = a⊔(b⊔c)` | 結果與合併拓撲無關 | 星狀／gossip／鏈狀同步 |
| 冪等律 `a⊔a = a` | 重複合併無副作用 | 訊息重傳、至少一次投遞 |

三者共同構成 **join-semilattice**，副本不需要任何協調就能最終收斂到同一狀態。ConfluxLab 沒有把這三條寫在註解裡，而是寫成**每次 CI 都執行的隨機化回歸測試**。

### 沙盤的事件模型

```text
副本狀態 × N + 向量時鐘 × N + 分區分組 × N
        ↓ 每輪隨機選擇一種事件
  ├─ 本機寫入（45%）  → 推進本機 Lamport 時戳與向量時鐘，施加 Op
  ├─ 雙向同步（43%）  → 分區內允許；跨分區記「受阻」；含機率重複投遞
  └─ 分區切換（12%）  → 把某個副本移到另一分區
        ↓ 排程結束
  網路癒合：撤銷全部分區，反覆兩兩反熵合併直至穩定
        ↓
  逐副本快照比對 + 單調性／墓碑完整性檢查 + 結構化報告
```

### 關鍵設計決策

1. **確定性優先於「更隨機」**：所有隨機性來自 `RNG`，`time.Now()` 不參與任何排程決策。失敗時只要一個種子就能完整重播，這在除錯收斂問題時價值極高。
2. **寫入識別碼全域唯一是硬前提**：`LWWRegister` 的 `(時戳, 節點)`、`ORSet` 的 tag 都必須唯一。開發過程中，代數律驗證器正是抓到產生器違反此前提所造成的交換律假失敗；現在 `Merge` 會顯式報錯而非靜默挑選（詳見 [`CHANGELOG.md`](CHANGELOG.md)）。
3. **衝突顯式化**：`Merge` 不允許靜默丟失資訊。要嘛用全序決定勝者，要嘛保留全部併發版本。
4. **零相依**：不引入 CLI 框架、測試框架或斷言庫，降低供應鏈風險，也讓 `go test ./...` 在任何環境都能直接跑。

### 後續迭代方向

1. 序列型 CRDT（RGA / LSEQ）文字協同與墓碑壓縮
2. 操作型 delta 傳輸與頻寬統計，量化反熵全量交換的代價
3. 最小重現序列自動收縮（delta debugging）：失敗時輸出最短排程
4. 拜占庭副本注入：模擬惡意節點送出非法狀態
5. 匯出 JSON Schema，讓編輯器可補全報告欄位
6. 與真實網路層（如 WebSocket / NATS）對接做端到端驗證

> 更完整的推導請見 [`docs/design.md`](docs/design.md)。

---

## 📦 打包與部署指南

本專案是**程式庫 + CLI**（不是需要安裝程式的桌面應用），提供兩種交付型態。

### 作為程式庫引入

```bash
go get github.com/gitstq/ConfluxLab
```

無外部相依，不需額外設定；模組路徑為 `github.com/gitstq/ConfluxLab`。

### 全平台靜態執行檔

```bash
make cross                  # 等同於 bash scripts/build.sh
# 或指定版本
VERSION=v1.0.0 bash scripts/build.sh
```

產出目錄 `dist/`：

| 目標 | 檔案 |
| --- | --- |
| Linux amd64 / arm64 | `confluxlab-v1.0.0-linux-amd64` / `...-linux-arm64` |
| macOS amd64 / arm64 | `confluxlab-v1.0.0-darwin-amd64` / `...-darwin-arm64` |
| Windows amd64 | `confluxlab-v1.0.0-windows-amd64.exe` |
| 校驗和 | `SHA256SUMS.txt` |

```bash
# 校驗
sha256sum -c dist/SHA256SUMS.txt

# 直接執行（Linux 範例，靜態連結，無執行期相依）
./dist/confluxlab-v1.0.0-linux-amd64 verify -rounds 500
```

> ⚠️ macOS / Windows 產物由 Linux 交叉編譯產生，建置與檔案格式已校驗，但執行期行為未在對應平台實機驗證；請在目標平台跑一次 `version` 與 `verify` 冒煙。

### 接入 CI

倉庫自帶 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)：Go 1.21/1.22/1.23 三版本矩陣，依序執行格式檢查、`go vet`、`go test -race`、建置、代數律門禁、5 副本 500 事件收斂性門禁、確定性重播 diff，以及**硬編碼憑證掃描**。任一步失敗即阻斷合併。

若不是使用 GitHub，最小接入方式就是兩行指令：

```bash
go test ./... -count=1
go run ./cmd/confluxlab verify -rounds 500 -replicas 5
```

---

## 🤝 貢獻指南

歡迎新增 CRDT 類型、強化不變量檢查或補齊語言版本。請先閱讀 [`CONTRIBUTING.md`](CONTRIBUTING.md)，重點如下：

- 提交訊息遵循 **Angular 規範**：`feat:` / `fix:` / `docs:` / `refactor:` / `test:`
- 新增類型須實作 `State` 介面、登錄 `AllKinds()`、撰寫單元測試，並跑通 `make all`
- 三條紅線：**不引入執行期相依**、**確定性優先**、**衝突顯式化**
- 提交 Issue 請附上 `-seed / -rounds / -replicas` 與 `-json` 輸出，便於精確重現

---

## 📄 開源授權說明

本專案採用 [MIT License](LICENSE)。

- ✅ 允許商用、修改、散佈、再授權
- ✅ 無需支付費用，無附加限制
- ⚠️ 需保留版權聲明與授權聲明
- ⚠️ 軟體按「原樣」提供，不含任何明示或暗示擔保

完整條款請見 [`LICENSE`](LICENSE)。Copyright (c) 2026 gitstq。
