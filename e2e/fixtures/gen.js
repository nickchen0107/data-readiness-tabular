const ExcelJS = require('exceljs');
const path = require('path');

async function gen() {
  const wb = new ExcelJS.Workbook();
  const s1 = wb.addWorksheet('Sheet1');
  s1.columns = [
    { header: '訂單編號', key: 'a', width: 15 },
    { header: '客戶名稱', key: 'b', width: 20 },
    { header: '金額', key: 'c', width: 12 },
    { header: '日期', key: 'd', width: 15 },
    { header: '狀態', key: 'e', width: 12 },
  ];
  s1.addRow({ a: 'ORD-001', b: '台灣科技公司', c: 15000, d: '2024-01-15', e: '完成' });
  s1.addRow({ a: 'ORD-002', b: '大同企業', c: 22000, d: '2024-01-16', e: '處理中' });
  s1.addRow({ a: 'ORD-003', b: '', c: 8500, d: '2024-01-17', e: '完成' });
  s1.addRow({ a: 'ORD-004', b: '聯發科技', c: '', d: '2024/01/18', e: '取消' });
  s1.addRow({ a: 'ORD-005', b: '鴻海精密', c: 45000, d: '2024-01-19', e: '完成' });
  s1.addRow({ a: 'ORD-002', b: '大同企業', c: 22000, d: '2024-01-16', e: '處理中' });
  s1.addRow({ a: 'ORD-006', b: '台積電', c: '三萬', d: '2024-01-20', e: '' });
  s1.addRow({ a: 'ORD-007', b: '華碩電腦', c: 12000, d: '2024-01-21', e: '完成' });
  s1.addRow({ a: '', b: '宏碁公司', c: 9000, d: '', e: '處理中' });
  s1.addRow({ a: 'ORD-009', b: '中華電信', c: 18000, d: '2024-01-23', e: '完成' });

  const s2 = wb.addWorksheet('Sheet2');
  s2.columns = [
    { header: 'ID', key: 'id', width: 10 },
    { header: 'Name', key: 'name', width: 20 },
  ];
  s2.addRow({ id: 1, name: 'Test' });
  s2.addRow({ id: 2, name: 'Data' });

  const out = path.join(__dirname, 'test-data.xlsx');
  await wb.xlsx.writeFile(out);
  console.log('Generated:', out);
}

gen().catch((e) => { console.error(e); process.exit(1); });
