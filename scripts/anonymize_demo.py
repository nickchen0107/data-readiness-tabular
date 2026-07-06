"""
基於「合併_戎克訂單系統(新).xlsx」製作模糊化的 demo 檔案。
保留完整 6600 行結構，只模糊化個資相關欄位。
"""
import openpyxl
import random
import os

random.seed(42)

# 模糊化用的假資料池
FAKE_CLIENTS = [
    'Alpha Corp\n(ALPHA)', 'Beta Systems', 'Gamma Electronics\n(GAMMA)',
    'Delta Tech', 'Epsilon IoT\n(EPS)', 'Zeta Solutions',
    'Eta Industries', 'Theta Computing', 'Iota Devices\n(IOTA)',
    'Kappa Networks', 'Lambda Micro', 'Mu Semiconductor',
]
FAKE_SALES = ['Alice', 'Bob', 'Carol', 'David', 'Eve', 'Frank', 'Grace', 'Henry']
FAKE_SUPPLIERS = [
    '供應商A', '供應商B\n代工', '供應商C', '代工廠D',
    '供應商E', 'OEM廠F', '供應商G\n組裝',
]

# 用 hash 做 deterministic 替換（同一個原始值永遠對應同一個假值）
client_map = {}
sales_map = {}
supplier_map = {}

def get_fake(original, fake_pool, mapping):
    if not original or not str(original).strip():
        return original
    key = str(original).strip()[:20]
    if key not in mapping:
        mapping[key] = random.choice(fake_pool)
    return mapping[key]

def anonymize_po(value):
    if not value:
        return value
    return f'PO-{random.randint(2000000, 9999999)}'

def anonymize_pi(value):
    if not value:
        return value
    return f'PI-{random.randint(20190101, 20251231)}'

def anonymize_erp(value):
    if not value:
        return value
    return f'WO-{random.randint(10000, 99999)}'

print("Loading workbook...")
src = os.path.join(os.path.dirname(__file__), '..', '測試資料', '合併_戎克訂單系統(新).xlsx')
wb = openpyxl.load_workbook(src)

# Sheet ALL: columns to anonymize (0-indexed)
# 1=Client, 2=PO, 3=PI, 8=Sales, 13=Supplier, 14=ERP
ws = wb['ALL']
print(f"Processing ALL sheet: {ws.max_row} rows x {ws.max_column} cols")

for row_idx in range(2, ws.max_row + 1):
    # Client (col B = index 2)
    cell = ws.cell(row=row_idx, column=2)
    if cell.value:
        cell.value = get_fake(cell.value, FAKE_CLIENTS, client_map)
    
    # PO No (col C = index 3)
    cell = ws.cell(row=row_idx, column=3)
    if cell.value:
        cell.value = anonymize_po(cell.value)
    
    # PI No (col D = index 4)
    cell = ws.cell(row=row_idx, column=4)
    if cell.value:
        cell.value = anonymize_pi(cell.value)
    
    # Sales (col I = index 9)
    cell = ws.cell(row=row_idx, column=9)
    if cell.value:
        cell.value = get_fake(cell.value, FAKE_SALES, sales_map)
    
    # Supplier (col N = index 14)
    cell = ws.cell(row=row_idx, column=14)
    if cell.value:
        cell.value = get_fake(cell.value, FAKE_SUPPLIERS, supplier_map)
    
    # ERP (col O = index 15)
    cell = ws.cell(row=row_idx, column=15)
    if cell.value:
        cell.value = anonymize_erp(cell.value)

# Sheet OA List: anonymize client and sales
ws2 = wb['OA List']
print(f"Processing OA List sheet: {ws2.max_row} rows")
for row_idx in range(3, ws2.max_row + 1):
    cell = ws2.cell(row=row_idx, column=2)
    if cell.value:
        cell.value = get_fake(cell.value, FAKE_CLIENTS, client_map)
    cell = ws2.cell(row=row_idx, column=3)
    if cell.value:
        cell.value = get_fake(cell.value, FAKE_SALES, sales_map)

output = os.path.join(os.path.dirname(__file__), '..', 'frontend', 'public', 'demo-sample.xlsx')
print(f"Saving to {output}...")
wb.save(output)
size = os.path.getsize(output)
print(f"Done! File size: {size / 1024:.0f} KB ({size / 1024 / 1024:.1f} MB)")
