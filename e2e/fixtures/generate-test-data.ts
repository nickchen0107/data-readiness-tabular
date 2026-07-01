/**
 * Script to generate the test Excel fixture file.
 * Run with: npx ts-node fixtures/generate-test-data.ts
 *
 * Creates a simple 10-row Excel file with intentional data quality issues:
 * - Empty cells (row completeness issues)
 * - Duplicate rows (uniqueness issues)
 * - Inconsistent formatting (format consistency issues)
 */
import ExcelJS from 'exceljs'
import path from 'path'

async function generate() {
  const workbook = new ExcelJS.Workbook()

  // Sheet 1: Main data with issues
  const sheet1 = workbook.addWorksheet('Sheet1')
  sheet1.columns = [
    { header: '訂單編號', key: 'order_id', width: 15 },
    { header: '客戶名稱', key: 'customer', width: 20 },
    { header: '金額', key: 'amount', width: 12 },
    { header: '日期', key: 'date', width: 15 },
    { header: '狀態', key: 'status', width: 12 },
  ]

  const rows = [
    { order_id: 'ORD-001', customer: '台灣科技公司', amount: 15000, date: '2024-01-15', status: '完成' },
    { order_id: 'ORD-002', customer: '大同企業', amount: 22000, date: '2024-01-16', status: '處理中' },
    { order_id: 'ORD-003', customer: '', amount: 8500, date: '2024-01-17', status: '完成' }, // Empty cell
    { order_id: 'ORD-004', customer: '聯發科技', amount: '', date: '2024/01/18', status: '取消' }, // Empty + inconsistent date
    { order_id: 'ORD-005', customer: '鴻海精密', amount: 45000, date: '2024-01-19', status: '完成' },
    { order_id: 'ORD-002', customer: '大同企業', amount: 22000, date: '2024-01-16', status: '處理中' }, // Duplicate row
    { order_id: 'ORD-006', customer: '台積電', amount: '三萬', date: '2024-01-20', status: '' }, // Text in numeric + empty
    { order_id: 'ORD-007', customer: '華碩電腦', amount: 12000, date: '2024-01-21', status: '完成' },
    { order_id: '', customer: '宏碁公司', amount: 9000, date: '', status: '處理中' }, // Multiple empties
    { order_id: 'ORD-009', customer: '中華電信', amount: 18000, date: '2024-01-23', status: '完成' },
  ]

  rows.forEach((row) => sheet1.addRow(row))

  // Sheet 2: Clean data (for sheet selection test)
  const sheet2 = workbook.addWorksheet('Sheet2')
  sheet2.columns = [
    { header: 'ID', key: 'id', width: 10 },
    { header: 'Name', key: 'name', width: 20 },
  ]
  sheet2.addRow({ id: 1, name: 'Test' })
  sheet2.addRow({ id: 2, name: 'Data' })

  const outputPath = path.join(__dirname, 'test-data.xlsx')
  await workbook.xlsx.writeFile(outputPath)
  console.log(`✓ Generated test fixture: ${outputPath}`)
}

generate().catch(console.error)
