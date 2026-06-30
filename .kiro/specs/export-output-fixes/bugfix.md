# 匯出模組錯誤修復需求文件

## Introduction

SAFE-AI Excel 梳理小工具的匯出模組（export package）存在三個影響使用者體驗的缺陷：
1. 匯出的 Excel 檔案使用固定檔名 `refined.xlsx`，未保留原始上傳檔名
2. PDF 報告在字型檔案不存在時中文顯示為亂碼
3. 清理日誌（cleaning.log）匯出為原始 JSON 格式，不易閱讀

這三個缺陷影響匯出檔案的可用性與專業度，需要一併修復。

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN 使用者匯出 Excel 檔案 THEN 系統在 Content-Disposition 標頭使用固定檔名 `refined.xlsx`，無論原始上傳檔案名為何
1.2 WHEN 使用者上傳名為「客戶名單.xlsx」的檔案並完成梳理後匯出 THEN 系統下載的檔案名稱為 `refined.xlsx`，使用者無法辨識該檔案對應哪份原始資料
1.3 WHEN Docker 容器中 `/app/assets/fonts/NotoSansTC-Regular.ttf` 字型檔案不存在 THEN PDF 報告回退使用 Helvetica 字型，導致所有中文文字顯示為亂碼
1.4 WHEN 字型檔案路徑設定錯誤或字型未安裝 THEN 系統無聲地降級至無法渲染中文的字型，不產生任何錯誤提示
1.5 WHEN 使用者匯出 cleaning.log THEN 系統輸出原始 JSON 格式（使用 `json.MarshalIndent`），包含程式化欄位名如 `operation_type`、`affected_rows` 陣列等，一般使用者無法直接閱讀

### Expected Behavior (Correct)

2.1 WHEN 使用者匯出 Excel 檔案 THEN 系統 SHALL 使用格式 `{原始檔名}_refined.xlsx` 作為下載檔名（例如原始檔案為「客戶名單.xlsx」，匯出為「客戶名單_refined.xlsx」）
2.2 WHEN Session 未記錄原始檔名（歷史資料） THEN 系統 SHALL 回退使用 `refined.xlsx` 作為預設檔名
2.3 WHEN Docker 容器建置時 THEN 系統 SHALL 確保中文字型檔案（NotoSansTC-Regular.ttf、NotoSansTC-Bold.ttf）已打包至 `/app/assets/fonts/` 路徑
2.4 WHEN 字型檔案在執行期間不存在 THEN 系統 SHALL 回傳明確錯誤訊息（如「中文字型檔案未安裝，無法產生 PDF 報告」），而非產出亂碼 PDF
2.5 WHEN 使用者匯出 cleaning.log THEN 系統 SHALL 輸出人類可讀的逐行格式，每行包含時間戳記與操作描述（格式範例：`[2024-03-15 14:23:01] 移除重複列：第 5, 12, 23 列`）
2.6 WHEN LogEntry 包含 AffectedRows 列表 THEN 系統 SHALL 將列號以逗號分隔方式嵌入操作描述中

### Unchanged Behavior (Regression Prevention)

3.1 WHEN 使用者匯出 Excel 檔案 THEN 系統 SHALL CONTINUE TO 產生包含正確欄位標題、資料列、及欄位寬度的有效 .xlsx 檔案
3.2 WHEN 使用者匯出 PDF 報告且字型檔案存在 THEN 系統 SHALL CONTINUE TO 正常產生包含分數摘要、六項指標表格、問題摘要、梳理前後對比的完整品牌 PDF 報告
3.3 WHEN 使用者匯出 cleaning.log THEN 系統 SHALL CONTINUE TO 包含所有 LogEntry 資訊（OperationType、AffectedRows、Timestamp、OperatorID、Details）
3.4 WHEN 匯出任一格式 THEN 系統 SHALL CONTINUE TO 驗證 session 所有權（JWT user_id 必須匹配）
3.5 WHEN 匯出快取檔案已存在 THEN 系統 SHALL CONTINUE TO 直接回傳快取檔案而非重新產生

---

## Bug 條件推導

### Bug 1：Excel 檔名問題

```pascal
FUNCTION isBugCondition_ExcelFilename(X)
  INPUT: X of type ExportRequest
  OUTPUT: boolean

  // 當使用者匯出 Excel 時，檔名總是固定的 refined.xlsx
  RETURN X.exportType = "xlsx"
END FUNCTION
```

```pascal
// Property: Fix Checking — Excel 檔名應保留原始檔名
FOR ALL X WHERE isBugCondition_ExcelFilename(X) DO
  result ← DownloadExcel'(X)
  IF X.session.OriginalFilename ≠ "" THEN
    ASSERT result.ContentDisposition contains (stripExtension(X.session.OriginalFilename) + "_refined.xlsx")
  ELSE
    ASSERT result.ContentDisposition contains "refined.xlsx"
  END IF
END FOR
```

```pascal
// Property: Preservation Checking — Excel 內容不受影響
FOR ALL X WHERE NOT isBugCondition_ExcelFilename(X) DO
  ASSERT F(X) = F'(X)
END FOR
```

### Bug 2：PDF 中文亂碼

```pascal
FUNCTION isBugCondition_PDFFont(X)
  INPUT: X of type PDFGenerationContext
  OUTPUT: boolean

  // 字型檔案不存在時觸發 bug
  RETURN NOT fileExists(X.config.Report.FontPath)
END FUNCTION
```

```pascal
// Property: Fix Checking — 字型不存在時應回傳錯誤而非亂碼 PDF
FOR ALL X WHERE isBugCondition_PDFFont(X) DO
  result ← GeneratePDF'(X)
  ASSERT result.error ≠ nil
  ASSERT result.error.message contains "字型"
END FOR
```

```pascal
// Property: Preservation Checking — 字型存在時 PDF 正常產出
FOR ALL X WHERE NOT isBugCondition_PDFFont(X) DO
  ASSERT F(X) = F'(X)
END FOR
```

### Bug 3：cleaning.log 格式問題

```pascal
FUNCTION isBugCondition_LogFormat(X)
  INPUT: X of type ExportRequest
  OUTPUT: boolean

  // 所有 log 匯出都受此 bug 影響
  RETURN X.exportType = "log"
END FUNCTION
```

```pascal
// Property: Fix Checking — Log 應為人類可讀格式
FOR ALL X WHERE isBugCondition_LogFormat(X) DO
  result ← GenerateLog'(X.session)
  lines ← splitLines(result.content)
  FOR EACH line IN lines DO
    ASSERT line matches pattern "^\[YYYY-MM-DD HH:MM:SS\] .+"
  END FOR
  ASSERT result.content does NOT contain "operation_type"
  ASSERT result.content does NOT contain "affected_rows"
END FOR
```

```pascal
// Property: Preservation Checking — Log 仍包含所有操作資訊
FOR ALL X WHERE isBugCondition_LogFormat(X) DO
  result ← GenerateLog'(X.session)
  FOR EACH entry IN X.session.CleaningLog DO
    ASSERT result.content contains entry.Timestamp formatted as "YYYY-MM-DD HH:MM:SS"
    ASSERT result.content contains humanReadableDescription(entry)
  END FOR
END FOR
```
