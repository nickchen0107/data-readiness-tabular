# T3 TrustChain 平台 — 外部平台串接 API 說明

## 概述

T3 TrustChain 提供 REST API 供外部平台整合，實現資料存證（Evidence）上鏈服務：

- 使用者身份管理
- 檔案上傳至 IPFS
- 資料 hash 寫入區塊鏈（Hyperledger Fabric）
- 存證記錄查詢與追溯

> **注意：** 資料存證不需要錢包（Wallet）。Wallet 僅在 T3 平台使用者要鑄造 NFT 時才需要。

## 基礎資訊

| 項目 | 值 |
|------|-----|
| Base URL | `http://<T3_HOST>:3000`（本機測試：`http://127.0.0.1:3000`）|
| 認證方式 | `X-API-Key` + `Bearer Token` |
| Content-Type | `application/json` |
| 回應格式 | JSON |

> **注意：** Base URL 為可配置項。正式上線時會更換為正式域名。

---

## 串接流程

```
T3 Admin                                       外部平台
────────                                       ────────
0. 建立平台 → 取得 API Key ──→ 提供給外部平台
                                               ↓
                                   1. 註冊使用者（POST /api/evidence/register）
                                               ↓
                                   2. 取得 apiToken
                                               ↓
                                   3. 上傳資料存證（POST /api/evidence/records）
                                      ├─ raw_dataset → IPFS + 鏈上 hash
                                      └─ processed_dataset → IPFS + 鏈上 hash
                                               ↓
                                   4. 查詢記錄（GET /api/evidence/records/:id）
```

---

### 前置步驟：建立平台（T3 Admin 操作）

在 T3 管理端建立一個「平台」，取得 API Key 後提供給外部平台。

```
POST /api/cross-platform/platforms
Authorization: Bearer <admin-token>
Content-Type: application/json
```

**Request Body:**
```json
{
  "platformName": "資料梳理平台",
  "description": "Data curation platform for evidence recording",
  "callbackUrl": "http://your-platform.com/webhook"
}
```

**Response (201):**
```json
{
  "platformId": "uuid-xxx",
  "apiKey": "Zuh3e3P...G2Kc",
  "defaultOrg": "GovernanceOrg",
  "defaultRole": "client",
  "allowedChannels": ["trust-channel"]
}
```

> ⚠️ `apiKey` 只會在建立時回傳一次，請妥善保存。

---

### 1. 註冊使用者

每個使用外部平台的使用者，首次需要在 T3 註冊一個身份。

```
POST /api/evidence/register
X-API-Key: <平台 API Key>
Content-Type: application/json
```

**Request Body:**
```json
{
  "externalUserId": "platform_user_123",
  "displayName": "張三"
}
```

**Response (201):**
```json
{
  "apiToken": "eyJ...",
  "userId": "uuid-xxx",
  "identityRef": "uuid-xxx",
  "expiresIn": 86400
}
```

| 回傳欄位 | 說明 | 是否儲存 |
|---------|------|----------|
| `apiToken` | 後續呼叫用的 JWT（24hr 有效） | ✅ 必須（過期後需重新 register） |
| `userId` | T3 內部使用者 ID | 建議儲存 |
| `identityRef` | Fabric 身份識別碼 | ✅ 必須 |

**注意事項：**
- `externalUserId` 必須在該平台內唯一
- 同一個 `externalUserId` 重複註冊會回傳 `409 USER_ALREADY_EXISTS`
- Token 過期後需重新呼叫 register 取得新 token

---

### 2. 上傳資料存證（Record Evidence）

將處理前/後的資料上傳到 IPFS，metadata hash 寫入區塊鏈。

```
POST /api/evidence/records
X-API-Key: <平台 API Key>
Authorization: Bearer <apiToken>
Content-Type: application/json
```

**Request Body:**
```json
{
  "artifacts": [
    {
      "type": "raw_dataset",
      "hash": "e3fdb8a9...（SHA-256 hex）",
      "storageOption": "ipfs-upload",
      "data": "aWQsdmFs...（base64 encoded file content）",
      "description": "原始感測器資料"
    },
    {
      "type": "processed_dataset",
      "hash": "1a4244c2...（SHA-256 hex）",
      "storageOption": "ipfs-upload",
      "data": "aWQsc2Vu...（base64 encoded file content）",
      "description": "清洗標準化後的資料"
    },
    {
      "type": "cleaning_log",
      "hash": "abc123...（SHA-256 hex）",
      "storageOption": "hash-only",
      "description": "清洗過程日誌（僅記錄 hash）"
    }
  ],
  "toolVersion": "data-curation-v2.1.0",
  "ruleVersion": "normalization-rule-v1.0"
}
```

**Response (201):**
```json
{
  "recordId": "SAFE-EVD-000001",
  "transactionId": "b55e6dcf-7e47-43d2-9f11-6dcea6635f1d",
  "signatureStatus": "confirmed",
  "timestamp": "2026-07-05T19:29:06.394Z",
  "artifacts": [
    { "type": "raw_dataset", "hash": "e3fdb8a9...", "ipfsCid": "QmQdYSSd..." },
    { "type": "processed_dataset", "hash": "1a4244c2...", "ipfsCid": "QmUP869L..." },
    { "type": "cleaning_log", "hash": "abc123...", "ipfsCid": null }
  ]
}
```

#### Artifact Type 說明

| type | 用途 |
|------|------|
| `raw_dataset` | 處理前的原始資料 |
| `processed_dataset` | 處理後的資料 |
| `cleaning_log` | 清洗/處理過程的日誌 |

#### Storage Option 說明

| 值 | 說明 | 是否需要 `data` 欄位 |
|----|------|---------------------|
| `ipfs-upload` | 上傳檔案到 IPFS，並在鏈上記錄 hash + CID | ✅ 需要（base64） |
| `hash-only` | 只在鏈上記錄 hash，不上傳檔案 | ❌ 不需要 |

#### Hash 計算方式

```javascript
const crypto = require('crypto');
const fileBuffer = fs.readFileSync('your-file.csv');
const hash = crypto.createHash('sha256').update(fileBuffer).digest('hex');
```

> **重要：** T3 會驗證 `hash` 與上傳的 `data` 是否一致。不一致會回傳 `400 HASH_MISMATCH`。

---

### 3. 查詢存證記錄

**查詢單筆：**

```
GET /api/evidence/records/:recordId
X-API-Key: <平台 API Key>
Authorization: Bearer <apiToken>
```

**Response (200):**
```json
{
  "recordId": "SAFE-EVD-000001",
  "userId": "uuid-xxx",
  "identityRef": "uuid-xxx",
  "platformId": "uuid-xxx",
  "rawDatasetHash": "e3fdb8a9...",
  "processedDatasetHash": "1a4244c2...",
  "cleaningLogHash": null,
  "rawDatasetCid": "QmQdYSSd...",
  "processedDatasetCid": "QmUP869L...",
  "cleaningLogCid": null,
  "toolVersion": "data-curation-v2.1.0",
  "ruleVersion": "normalization-rule-v1.0",
  "signatureStatus": "confirmed",
  "transactionId": "b55e6dcf-...",
  "timestamp": "2026-07-05T19:29:06.394Z"
}
```

**列出所有記錄（分頁）：**

```
GET /api/evidence/records?page=1&pageSize=20
X-API-Key: <平台 API Key>
Authorization: Bearer <apiToken>
```

---

### 外部平台應儲存的資料

每次呼叫 T3 API 後，外部平台需要儲存回傳值：

| 回傳欄位 | 來源 API | 用途 | 是否必須儲存 |
|---------|----------|------|------------|
| `apiToken` | `/api/evidence/register` | 後續 API 呼叫的認證 | ✅ 必須 |
| `identityRef` | `/api/evidence/register` | Fabric 身份識別 | ✅ 必須 |
| `recordId` | `/api/evidence/records` | 存證記錄 ID，後續查詢用 | ✅ 必須 |
| `transactionId` | `/api/evidence/records` | 區塊鏈交易 ID（上鏈證明） | ✅ 必須 |
| `ipfsCid` | `/api/evidence/records` | IPFS 永久位址，可驗證檔案未被竄改 | ✅ 必須 |
| `signatureStatus` | `/api/evidence/records` | 上鏈狀態 | 建議儲存 |

> **重要：** `transactionId` 是最核心的值。它是區塊鏈上這筆記錄的唯一證明。
> 即使 T3 平台未來停止服務，只要有 `transactionId`，就能在 Fabric 網路上驗證這筆記錄的存在。

---

### DB Schema 建議

外部平台的資料庫建議新增以下表：

```sql
-- 使用者與 T3 的帳號對應
CREATE TABLE t3_user_mapping (
  id SERIAL PRIMARY KEY,
  local_user_id VARCHAR(255) NOT NULL UNIQUE,
  t3_identity_ref VARCHAR(255) NOT NULL,
  t3_api_token TEXT,
  t3_token_expires_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 存證記錄對應
CREATE TABLE t3_evidence_records (
  id SERIAL PRIMARY KEY,
  local_user_id VARCHAR(255) NOT NULL,
  local_job_id VARCHAR(255),
  t3_record_id VARCHAR(255) NOT NULL,
  t3_transaction_id VARCHAR(255),
  raw_dataset_cid VARCHAR(255),
  processed_dataset_cid VARCHAR(255),
  cleaning_log_cid VARCHAR(255),
  signature_status VARCHAR(20),
  recorded_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
```

---

### 整合程式碼範例（TypeScript）

```typescript
import crypto from 'crypto';
import fs from 'fs';

class T3EvidenceClient {
  private baseUrl: string;
  private apiKey: string;

  constructor(baseUrl: string, apiKey: string) {
    this.baseUrl = baseUrl;
    this.apiKey = apiKey;
  }

  // 註冊使用者（每人只需一次）
  async registerUser(externalUserId: string, displayName: string) {
    const res = await fetch(`${this.baseUrl}/api/evidence/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': this.apiKey,
      },
      body: JSON.stringify({ externalUserId, displayName }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json() as Promise<{
      apiToken: string;
      userId: string;
      identityRef: string;
      expiresIn: number;
    }>;
  }

  // 上傳資料存證
  async recordEvidence(
    apiToken: string,
    rawFile: Buffer,
    processedFile: Buffer,
    toolVersion: string,
    ruleVersion?: string,
  ) {
    const rawHash = crypto.createHash('sha256').update(rawFile).digest('hex');
    const processedHash = crypto.createHash('sha256').update(processedFile).digest('hex');

    const res = await fetch(`${this.baseUrl}/api/evidence/records`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': this.apiKey,
        'Authorization': `Bearer ${apiToken}`,
      },
      body: JSON.stringify({
        artifacts: [
          {
            type: 'raw_dataset',
            hash: rawHash,
            storageOption: 'ipfs-upload',
            data: rawFile.toString('base64'),
          },
          {
            type: 'processed_dataset',
            hash: processedHash,
            storageOption: 'ipfs-upload',
            data: processedFile.toString('base64'),
          },
        ],
        toolVersion,
        ruleVersion,
      }),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json() as Promise<{
      recordId: string;
      transactionId: string;
      signatureStatus: string;
      timestamp: string;
      artifacts: Array<{ type: string; hash: string; ipfsCid: string | null }>;
    }>;
  }

  // 查詢記錄
  async getRecord(apiToken: string, recordId: string) {
    const res = await fetch(`${this.baseUrl}/api/evidence/records/${recordId}`, {
      headers: {
        'X-API-Key': this.apiKey,
        'Authorization': `Bearer ${apiToken}`,
      },
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  }
}

// 使用範例
const client = new T3EvidenceClient('http://127.0.0.1:3000', 'your-api-key');

// 1. 註冊
const { apiToken, identityRef } = await client.registerUser('user_001', '張三');

// 2. 存證
const rawFile = fs.readFileSync('raw-data.csv');
const processedFile = fs.readFileSync('processed-data.csv');
const result = await client.recordEvidence(apiToken, rawFile, processedFile, 'tool-v1.0');

console.log('Record ID:', result.recordId);
console.log('TX ID:', result.transactionId);  // 區塊鏈交易證明
console.log('Raw CID:', result.artifacts[0].ipfsCid);  // IPFS 永久位址
```

---

## 錯誤處理

所有錯誤回應格式：

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "人類可讀的錯誤描述"
  }
}
```

### 錯誤碼

| HTTP Status | Code | 說明 |
|-------------|------|------|
| 400 | MISSING_FIELDS | 缺少必要欄位 |
| 400 | VALIDATION_ERROR | artifacts 格式不正確 |
| 400 | HASH_MISMATCH | 上傳的 data 與提供的 hash 不一致 |
| 401 | API_KEY_REQUIRED | 缺少 X-API-Key header |
| 401 | INVALID_API_KEY | API Key 無效 |
| 403 | ACCESS_DENIED | 無權存取該記錄 |
| 404 | RECORD_NOT_FOUND | 記錄不存在 |
| 409 | USER_ALREADY_EXISTS | 使用者已註冊 |
| 502 | IPFS_UPLOAD_FAILED | IPFS 上傳失敗 |
| 502 | CHAIN_COMMIT_FAILED | 區塊鏈提交失敗 |

---

## 健康檢查

```
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "services": ["auth", "mint", "marketplace", "pricing"]
}
```

---

## 安全注意事項

1. **API Key 保密** — API Key 等同於平台身份，洩漏需立即通知 T3 管理員重新產生
2. **Token 快取** — apiToken 有效期 24 小時，建議快取避免頻繁 register
3. **HTTPS** — 正式環境必須使用 HTTPS
4. **Hash 驗證** — T3 會驗證上傳的 data 與 hash 是否一致，防止竄改
5. **使用者隔離** — 每個使用者只能查詢自己的記錄

---

## 環境配置

在外部平台的 `.env` 中加入：

```env
T3_API_URL=http://127.0.0.1:3000       # 本機測試
# T3_API_URL=https://t3.your-domain.com  # 正式環境
T3_API_KEY=your-platform-api-key-here
```
