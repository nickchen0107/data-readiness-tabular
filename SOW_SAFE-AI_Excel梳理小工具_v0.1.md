# SAFE-AI Excel 梳理小工具 — Statement of Work (SoW) v0.1

## 1. 專案概述

### 1.1 產品名稱
SAFE-AI 資料梳理平台 — Excel 梳理小工具

### 1.2 目標
建構一個 Docker-based、前後端分離、模組化的 MVP 工具，讓使用者可以上傳 Excel 檔案，經由 AI Data Readiness 評估與資料梳理流程後，產出品質提升的資料集，並透過 LLM 問答對比展現梳理前後的差異。

### 1.3 設計原則
- Docker 容器化部署
- 前後端分離（React + Go）
- 模組化架構，各模組可獨立替換
- RESTful API 統一介面
- 評估指標全部為公式計算（不含 AI），僅模組 F 問答串接 LLM

### 1.4 參考依據
- ISO/IEC 25012:2008 Data Quality Model（維度定義）
- AIDRIN (AI Data Readiness Inspector)，arXiv 2024
- Snowflake AI-Ready Data Framework（開源）
- ISO/IEC 5259-1:2024（AI/ML 資料品質標準）
- Tidy Data, Hadley Wickham, 2014

---

## 2. 技術架構

### 2.1 技術選型

| 層級 | 技術 | 說明 |
|------|------|------|
| 前端 | React + TypeScript | SPA，Nginx serve + reverse proxy |
| 後端 | Go (Gin/Echo) | RESTful API server |
| 資料庫 | PostgreSQL | 使用者帳號、檔案 metadata、評估結果、cleaning log |
| 檔案儲存 | Local volume (Docker) | 上傳的 Excel 檔案持續保留 |
| LLM | Gemini API | 僅模組 F 使用 |
| 區塊鏈 | REST API 接口 | 本專案只定義接口，不實作鏈端 |
| 部署 | Docker Compose | 所有服務容器化 |
| 語系 | 繁體中文 | v0.1 僅繁中 |

### 2.2 服務架構

```
┌────────────────────────────────────────────────────────┐
│  Docker Compose                                        │
│                                                        │
│  ┌─────────────────┐     ┌───────────────────────────┐│
│  │ frontend        │     │ backend (Go)              ││
│  │ React + Nginx   │────▶│                           ││
│  │ Port: 80        │     │  /api/auth/*              ││
│  └─────────────────┘     │  /api/upload/*            ││
│                          │  /api/assess/*            ││
│                          │  /api/clean/*             ││
│                          │  /api/export/*            ││
│                          │  /api/evidence/*          ││
│                          │  /api/qa/*                ││
│                          │  /api/settings/*          ││
│                          └──────────┬────────────────┘│
│                                     │                  │
│  ┌───────────────────────────────────┼────────────┐   │
│  │ PostgreSQL                        │            │   │
│  │ Port: 5432                        │            │   │
│  └───────────────────────────────────┘            │   │
│                                                    │   │
└────────────────────────────────────────────────────┘   │
          │                                │             │
          ▼                                ▼             │
    Gemini API                    Blockchain API         │
    (外部 LLM)                   (另一個 Kiro 專案)      │
```

### 2.3 Nginx 的角色
- 提供 React 打包後的靜態檔案（HTML、JS、CSS）
- 反向代理：將 `/api/*` 請求轉發至 Go 後端容器
- 單頁應用路由：所有非 API 路由回傳 `index.html`（React Router 需要）
- 未來可加入壓縮、速率限制、HTTPS 等功能

---

## 3. 功能模組規格

### 3.1 Landing Page

| 項目 | 說明 |
|------|------|
| 範圍 | 一個簡潔的入口頁面 |
| 內容 | AICM 提示文字（一行，僅提示存在，不實作）+ 工具入口按鈕 |
| 邊界 | 不實作 AICM 問卷、計分或報告 |

---

### 3.2 帳號系統

| 項目 | 說明 |
|------|------|
| 功能 | 註冊、登入、登出 |
| 實作 | 帳號（email）+ 密碼（bcrypt hash）+ JWT token |
| 權限 | 本版不實作權限管控，所有登入使用者功能相同 |
| 頁面 | 登入頁、註冊頁 |
| 邊界 | 不做 OAuth、不做角色管理、不做忘記密碼 |

---

### 3.3 模組 A — 上傳 + AI Data Readiness 評估

#### 3.3.1 上傳

| 項目 | 說明 |
|------|------|
| 支援格式 | xlsx, csv |
| 不支援 | xls（本版暫不支援） |
| 檔案限制 | 最大 50MB / 100,000 rows |
| Sheet 處理 | 多 sheet 時讓使用者選擇一個 sheet 進行評估 |
| 合併儲存格 | 偵測並標記，不自動拆解 |
| 公式欄位 | 取計算後的值（不解析公式本身） |
| 留存策略 | 上傳後持續保留，刪除機制後續定義 |

#### 3.3.2 六項評估指標

所有計算皆為規則式公式，不含 AI。

##### 指標 1: 列完整度（Row Completeness）— 權重 20%

**演算法說明：**

衡量每一列資料的填寫完整程度。一份資料如果大量的列都有空欄位，代表這些資料在進入 AI 分析時容易產生偏差或無法被正確彙總。

**計算公式：**

```
分數 = Σ(每列非空欄位數 / 總欄位數) / 總列數 × 100
```

範例：一份有 10 個欄位、100 列的資料，若每列平均有 7 個欄位有值，則分數 = 70。

依據：ISO 25012 完整性（Completeness）維度

##### 指標 2: 欄位完整度（Column Completeness）— 權重 20%

**演算法說明：**

衡量每個欄位的缺漏比例。與指標 1 互補 — 指標 1 看「每列」，指標 2 看「每欄」。某些欄位若缺漏嚴重（例如毛利欄 60% 為空），會嚴重影響該欄相關問題的回答能力。

**計算公式：**

```
分數 = Σ(每欄非空值數 / 總列數) / 總欄位數 × 100
```

範例：10 個欄位中，3 個欄位的填寫率為 40%，其餘 7 個為 95%，則分數 ≈ (0.4×3 + 0.95×7) / 10 × 100 = 78.5。

依據：ISO 25012 完整性（Completeness）維度（欄位層級視角）

##### 指標 3: 格式一致性（Format Consistency）— 權重 15%

**演算法說明：**

衡量同一欄位中，資料格式是否統一。常見問題如日期欄混用 `2024/1/5`、`2024-01-05`、`113.1.5` 三種格式，或數字欄中混入文字。格式不一致會導致程式解析錯誤或排序/比較結果不正確。

**計算公式：**

```
對每欄：
  1. 取所有非空值，偵測每個值的格式類型
  2. 找出該欄的主要格式（出現次數最多的格式類型）
  3. 該欄得分 = 符合主要格式的值數 / 非空值總數

分數 = 所有欄位得分的平均值 × 100
```

**格式類型偵測規則：**
- 日期類：匹配常見日期格式（`yyyy/MM/dd`、`yyyy-MM-dd`、民國年格式 `yyy.M.d`）
- 數字類：匹配純數字、含千分位逗號、含小數點
- 布林類：匹配 true/false、是/否、Y/N
- 文字類：不符合以上類型的其餘值

範例：某欄有 100 個非空值，其中 80 個是 `yyyy-MM-dd` 格式、15 個是 `yyyy/MM/dd`、5 個是文字。主要格式為 `yyyy-MM-dd`，該欄得分 = 80/100 = 0.8。

依據：ISO 25012 一致性（Consistency）維度

##### 指標 4: 重複與近似（Duplicate / Similar）— 權重 10%

**演算法說明：**

本指標偵測兩種問題：
1. **完全重複** — 所有欄位值完全相同的列（以整列 hash 比對）
2. **近似重複** — 文字內容高度相似，但因書寫方式不同而被視為不同資料

**近似比對方法 — Levenshtein 距離：**

Levenshtein 距離是計算「一個字串需要經過多少次基本編輯（插入、刪除、替換一個字元）才能變成另一個字串」的演算法。距離越小代表兩個字串越相似。

範例：
- `ABC Co.` 與 `ABC Company` → 距離 = 5（差異較大，不算近似）
- `台基電子` 與 `台基電子股份有限公司` → 距離 = 6（不算近似）
- `P-001` 與 `P001` → 距離 = 1（算近似）
- `王大明` 與 `王大名` → 距離 = 1（算近似）

本版閾值設為 ≤ 2，代表「編輯 2 次以內即可變為相同」才判定為近似。

**近似偵測的欄位選擇邏輯：**

不是所有欄位都需要做近似比對。系統會自動篩選適合比對的欄位：
1. 該欄為文字類型（非數字、非日期）
2. 該欄的唯一值數量（cardinality）介於總列數的 5%–80%（太低代表分類欄、太高代表流水號）
3. 最多取符合條件的前 5 欄（效能限制：近似比對為 O(n²) 運算）

**計算公式：**

```
完全重複列數 = 全欄位 hash 完全相同的列數
近似重複組數 = 符合條件的欄位中，Levenshtein 距離 ≤ 2 的值組數

分數 = (1 - (完全重複列數 + 近似重複組數 × 0.5) / 總列數) × 100
最低分 = 0
```

- 完全重複的影響權重為 1.0（確定是問題）
- 近似重複的影響權重為 0.5（可能是問題，但需人工確認）

依據：ISO 25012 唯一性（Uniqueness）維度

##### 指標 5: 表格結構品質（Table Structure Quality）— 權重 15%

**演算法說明：**

本指標評估 Excel 的結構是否為「機器可直接解析的乾淨表格」。起始分數 100，每偵測到一種結構問題就扣分。

**扣分項目與理由：**

| 問題類型 | 偵測方式 | 扣分 | 為何扣分 |
|----------|----------|------|----------|
| 合併儲存格 | Excel 解析時偵測 merged cells | -20 | 合併格讓資料無法逐列/逐欄正確讀取，程式解析時會產生空值或錯位 |
| 多層標題 | 前 5 列中有 >1 列全為文字且內容不重複 | -20 | 多層標題代表欄位名稱不明確，程式無法確定哪一列是真正的欄位名稱 |
| 小計/合計列 | 含「小計」「合計」「total」「subtotal」關鍵字的列 | -15 | 這些列是彙總值，混在資料中會導致重複計算或統計錯誤 |
| 多表同 sheet | 偵測連續空白列（≥2 列）分隔的多塊資料區 | -25 | 多表混在同一 sheet 讓程式無法判斷表格邊界，解析極易出錯 |
| 備註混入資料欄 | 某欄文字長度標準差 > 平均值 × 3 | -10 | 代表該欄混入了超長備註文字，不符合單一欄位單一用途的結構原則 |

**計算公式：**

```
分數 = max(0, 100 - 各扣分項加總)
```

依據：Tidy Data 原則（Hadley Wickham, 2014）— 每列一筆觀測、每欄一個變數、每個表格一種觀測類型

##### 指標 6: AI 問答可用性（AI Query Readiness）— 權重 20%

**演算法說明：**

本指標評估「這份資料是否具備讓大語言模型有效回答問題的結構條件」。大語言模型在做結構化問答時，需要資料中存在特定角色的欄位才能穩定回答。本指標以五項子條件加分制計算。

**加分項目與理由：**

| 子條件 | 偵測邏輯 | 加分 | 為何加分 |
|--------|----------|------|----------|
| 識別欄存在 | 有至少一欄的唯一值比率 > 80%（去空值後計算） | +20 | 識別欄（如客戶名、訂單號）讓模型能區分每筆資料是什麼，否則無法做個體查詢 |
| 時間欄存在 | 有至少一欄可被規則式日期解析器解析（取樣前 100 列，成功率 > 60%） | +20 | 時間欄讓模型能回答「哪一年」「哪個月」等時序問題，是商業問答最常見的維度 |
| 分類欄存在 | 有至少一欄唯一值數量 < 總列數 × 20% 且唯一值數量 > 1 | +20 | 分類欄讓模型能做分群統計（如「按業務員分」「按產品類別分」） |
| 數值欄存在 | 有至少一欄為數字類型或可解析為數字的比例 > 80% | +20 | 數值欄讓模型能做加總、平均、排名等量化分析 |
| 欄位名稱品質 | 所有欄位名稱非空、無重複、長度 > 1 字元 | +20 | 模型靠欄位名稱理解資料語意，名稱不清會導致回答偏離 |

**計算公式：**

```
分數 = 各子條件得分加總（滿分 100）
```

每項子條件為「全有或全無」（滿足條件 = 得滿分，不滿足 = 0 分）。

**依據：**
- Snowflake AI-Ready Data Framework 的 Contextual factor 要求資料具備明確的 entity identifier 與 semantic documentation
- AIDRIN 框架的 queryability 維度
- 大語言模型做表格問答的實務需求：需要 entity key（識別）、time axis（時間）、category（分類）、measure（數值）四種欄位角色

#### 3.3.3 總分計算

```
AI 資料就緒分數 = 
    列完整度 × 0.20 +
    欄位完整度 × 0.20 +
    格式一致性 × 0.15 +
    重複與近似 × 0.10 +
    表格結構品質 × 0.15 +
    AI 問答可用性 × 0.20
```

所有子指標分數為 0–100，加權後總分亦為 0–100。

#### 3.3.4 分級

| 分數 | 狀態 | 說明 |
|------|------|------|
| 80–100 | Ready（就緒） | 可直接進入梳理與問答 |
| 60–79 | Conditionally Ready（有條件就緒） | 建議先梳理再進入 AI 問答 |
| 0–59 | Not Ready（尚未就緒） | 建議補齊高風險缺漏或修正結構 |

#### 3.3.5 輸出

- AI 資料就緒分數（總分 + 六項子分數 + 各分數對應的說明）
- 每列就緒度統計（High / Medium / Low 各多少列）
- 問題清單（含嚴重度、影響列數、處理建議文字）

---

### 3.4 模組 B — 分流

| 項目 | 說明 |
|------|------|
| 功能 | 依分數與問題清單，讓使用者選擇路徑 |
| 路徑 1 | 以現況梳理 → 進入模組 C1 |
| 路徑 2 | 補齊後重新上傳 → 回到模組 A |
| UI | 兩張卡片選擇 + 風險提示 |
| 邊界 | Not Ready 需提示風險，但不阻擋使用者選擇 |

---

### 3.5 模組 C1 — 以現況梳理

| 項目 | 說明 |
|------|------|
| 功能 | 人工確認 + 批次規則處理 |
| 批次規則（v0.1） | 4 種：統一日期格式、移除重複列、客戶名正規化（簡易規則）、移除小計列 |
| 單列處理 | 兩種選項：填入「N/A」、刪除該列 |
| 處理紀錄 | 每個操作寫入 cleaning log（JSON 格式），含操作類型、影響列、時間戳、操作者 |
| 介面 | 資料表格檢視 + 問題標示 + 批次規則設定 + 確認按鈕 |
| 邊界 | 不做進階模糊比對、不做 AI 輔助補值 |

#### 批次規則細節

| 規則 | 邏輯 |
|------|------|
| 統一日期格式 | 偵測到的日期欄位，全部轉為 `yyyy-MM-dd` 格式 |
| 移除重複列 | 完全相同的列僅保留第一筆 |
| 客戶名正規化 | 移除公司後綴變體（Co./Company/公司/股份有限公司）後比對，相同者統一為最長版本 |
| 移除小計列 | 含「小計」「合計」「total」「subtotal」關鍵字的列標記移除 |

---

### 3.6 模組 C2 — 補齊後重新評估

| 項目 | 說明 |
|------|------|
| 功能 | 使用者重新上傳修正後的 Excel，重跑模組 A 評估 |
| 實作 | 後端邏輯等同模組 A，前端多一個「重新上傳」入口 |
| 邊界 | 不要求完全無空缺，依 readiness score 判斷 |

---

### 3.7 模組 D — 產出

| 項目 | 說明 |
|------|------|
| 功能 | 產出梳理完成的資料版本，提供下載 |
| 輸出檔案 | |
| ├ refined.xlsx | 梳理後資料集 |
| ├ report.pdf | AI 資料就緒報告（美觀排版） |
| └ cleaning.log | JSON 格式處理紀錄 |
| PDF 報告內容 | 總分、六項指標分數與說明、問題摘要、前後對比統計、梳理規則摘要 |
| PDF 風格 | 品牌色系、清晰排版、含圖表（環形分數圖、長條圖） |
| 介面 | 進度條 + 前後統計對比 + 下載按鈕 |
| 邊界 | 不承諾「資料絕對正確」，僅保證「依規則處理」 |

---

### 3.8 模組 E — Evidence 存證（API 接口）

| 項目 | 說明 |
|------|------|
| 本專案職責 | 計算 hash + 呼叫區塊鏈 API + 顯示存證結果 |
| 不實作 | 區塊鏈端邏輯（由另一個 Kiro 專案處理） |

#### 存證 API 接口定義

**送出存證（本專案 → 區塊鏈服務）:**

```
POST /api/evidence/submit
Content-Type: application/json

Request:
{
  "dataset_hash": "sha256 of refined.xlsx",
  "cleaning_log_hash": "sha256 of cleaning.log",
  "report_hash": "sha256 of report.pdf",
  "timestamp": "2026-06-10T14:32:07Z",
  "tool_version": "excel-tool-0.1",
  "rule_version": "rules-0.1.0",
  "operator_id": "user-uuid",
  "metadata": {
    "original_filename": "翔立光_業績原始資料.xlsx",
    "original_rows": 2043,
    "refined_rows": 218,
    "readiness_before": 47,
    "readiness_after": 91
  }
}

Response:
{
  "record_id": "SAFE-EVD-002187",
  "transaction_hash": "0x...",
  "signature_status": "confirmed | pending",
  "verification_url": "https://..."
}
```

**查詢存證（本專案 → 區塊鏈服務）:**

```
GET /api/evidence/{record_id}

Response:
{
  "record_id": "SAFE-EVD-002187",
  "dataset_hash": "...",
  "cleaning_log_hash": "...",
  "timestamp": "...",
  "signature_status": "confirmed",
  "verification_url": "..."
}
```

#### 前端顯示

- Evidence Record 卡片（hash、timestamp、record ID、status）
- 明確標示 Demo Mode（待區塊鏈正式上線後切換）
- 顯示「No sensitive data on-chain」「Integrity verifiable」

---

### 3.9 模組 F — 前後對比問答

| 項目 | 說明 |
|------|------|
| 大語言模型 | Gemini API |
| 資料注入方式 | 將結構化資料以 CSV 格式片段直接放入提示詞 |
| 回應方式 | 整批回應（非串流） |
| 介面 | 左右並排：原始資料回答 vs 梳理後資料回答 |
| 資料不足防護 | 若使用的資料欄缺漏率 > 50%，直接回傳「資料不足」+ 原因說明，不呼叫模型 |
| 建議問題 | 系統根據欄位名稱自動產生 3 個建議問題 |
| 自由輸入 | 使用者可自行輸入問題 |
| 資料保護提示 | 告知使用者資料會送至外部模型，需同意後才使用 |
| 邊界 | 不做向量搜尋、不做串流回應、不做多輪對話記憶 |

#### 資料不足防護規則

1. 偵測問題中涉及的欄位（關鍵字匹配）
2. 若該欄位在原始資料中缺漏率 > 50% → 回傳「資料不足」訊息，說明缺少什麼
3. 若無法判斷涉及哪個欄位 → 仍送出至模型，但附加系統提示詞要求模型在資料不足時明確說明

---

### 3.10 權重設定頁面

| 項目 | 說明 |
|------|------|
| 功能 | 管理者可調整六項指標的權重比例 |
| 介面 | 六個滑桿，總和需 = 100%（介面即時計算並顯示） |
| 儲存 | PostgreSQL 系統設定表 |
| 預設值 | 列完整度 20%、欄位完整度 20%、格式一致性 15%、重複與近似 10%、表格結構品質 15%、AI 問答可用性 20% |
| 規則 | 變更權重不影響歷史報告（每份報告內嵌當時使用的權重值） |

---

## 4. API 端點清單

| Method | Endpoint | 模組 | 說明 |
|--------|----------|------|------|
| POST | /api/auth/register | 帳號 | 註冊 |
| POST | /api/auth/login | 帳號 | 登入，回傳 JWT |
| POST | /api/auth/logout | 帳號 | 登出 |
| GET | /api/auth/me | 帳號 | 取得當前使用者資訊 |
| POST | /api/upload | A | 上傳 Excel |
| GET | /api/upload/{id}/sheets | A | 取得 sheet 列表 |
| POST | /api/assess | A | 執行 readiness 評估 |
| GET | /api/assess/{id} | A | 取得評估結果 |
| GET | /api/assess/{id}/issues | A | 取得問題清單 |
| POST | /api/clean/apply | C1 | 套用梳理規則 |
| GET | /api/clean/{id}/preview | C1 | 預覽梳理結果 |
| GET | /api/clean/{id}/log | C1/D | 取得 cleaning log |
| GET | /api/export/{id}/xlsx | D | 下載 refined Excel |
| GET | /api/export/{id}/pdf | D | 下載 PDF 報告 |
| GET | /api/export/{id}/log | D | 下載 cleaning log |
| POST | /api/evidence/submit | E | 送出存證（proxy 到區塊鏈 API） |
| GET | /api/evidence/{record_id} | E | 查詢存證狀態 |
| POST | /api/qa/ask | F | 送出問題 |
| GET | /api/qa/suggestions/{assess_id} | F | 取得建議問題 |
| GET | /api/settings/weights | 設定 | 取得當前權重 |
| PUT | /api/settings/weights | 設定 | 更新權重 |

---

## 5. 資料庫 Schema（概要）

### users
| Column | Type | Note |
|--------|------|------|
| id | UUID | PK |
| email | VARCHAR | UNIQUE |
| password_hash | VARCHAR | bcrypt |
| created_at | TIMESTAMP | |

### uploads
| Column | Type | Note |
|--------|------|------|
| id | UUID | PK |
| user_id | UUID | FK → users |
| filename | VARCHAR | 原始檔名 |
| file_path | VARCHAR | 儲存路徑 |
| file_size | BIGINT | bytes |
| row_count | INT | |
| col_count | INT | |
| selected_sheet | VARCHAR | 使用者選擇的 sheet |
| created_at | TIMESTAMP | |

### assessments
| Column | Type | Note |
|--------|------|------|
| id | UUID | PK |
| upload_id | UUID | FK → uploads |
| total_score | FLOAT | 0-100 |
| row_completeness | FLOAT | |
| column_completeness | FLOAT | |
| format_consistency | FLOAT | |
| duplicate_similar | FLOAT | |
| table_structure | FLOAT | |
| ai_query_readiness | FLOAT | |
| weights_snapshot | JSONB | 評估時使用的權重 |
| status | VARCHAR | ready / conditional / not_ready |
| issues | JSONB | 問題清單 |
| created_at | TIMESTAMP | |

### cleaning_sessions
| Column | Type | Note |
|--------|------|------|
| id | UUID | PK |
| assessment_id | UUID | FK → assessments |
| user_id | UUID | FK → users |
| rules_applied | JSONB | 使用的規則 |
| rows_before | INT | |
| rows_after | INT | |
| score_after | FLOAT | 梳理後重新評分 |
| cleaning_log | JSONB | 完整 log |
| created_at | TIMESTAMP | |

### evidence_records
| Column | Type | Note |
|--------|------|------|
| id | UUID | PK |
| cleaning_session_id | UUID | FK → cleaning_sessions |
| dataset_hash | VARCHAR | SHA-256 |
| log_hash | VARCHAR | SHA-256 |
| report_hash | VARCHAR | SHA-256 |
| record_id | VARCHAR | 區塊鏈回傳的 ID |
| signature_status | VARCHAR | confirmed / pending / demo |
| created_at | TIMESTAMP | |

### system_settings
| Column | Type | Note |
|--------|------|------|
| key | VARCHAR | PK |
| value | JSONB | 設定值 |
| updated_at | TIMESTAMP | |
| updated_by | UUID | FK → users |

---

## 6. 本版不做項目

- AICM 問卷、計分或報告
- xls 格式支援
- 合併儲存格自動拆解
- AI 輔助補值或語意推測
- 向量搜尋（RAG）
- 串流式模型回應
- 多輪對話記憶
- 權限管控（角色、資料隔離）
- OAuth / SSO / 忘記密碼
- 區塊鏈端實作
- 多語系（僅繁中）
- 資料自動刪除機制
- 進階模糊比對演算法（如 Jaro-Winkler）
- 資料匯入匯出格式轉換
- 正式量子安全驗證宣告

---

## 7. 驗收標準

1. 系統可接受不同類型 Excel（業績、庫存、BOM、訂單等）
2. 六項指標計算結果可重現（同一份檔案多次評估結果一致）
3. 梳理操作正確執行且 cleaning log 完整記錄
4. PDF 報告可正常開啟且排版美觀
5. Evidence API 呼叫正確（即使區塊鏈端未上線，本端邏輯正確）
6. 模組 F 問答可正確串接 Gemini，且 Guardrail 在資料不足時阻擋
7. 所有 API 回應符合 RESTful 規範，錯誤碼正確
8. Docker Compose 一鍵啟動所有服務
9. 權重設定可正確儲存並影響後續評估

---

## 8. 交付物

1. 完整原始碼（Git repository）
2. Docker Compose 設定檔
3. API 文件（OpenAPI/Swagger）
4. 資料庫 migration scripts
5. README（建置、部署、設定說明）
