// Seed translations into DB from fallback JSON files
const fs = require('fs');
const path = require('path');

const zhPath = path.join(__dirname, '..', 'frontend', 'src', 'i18n', 'fallback', 'zh-TW.json');
const enPath = path.join(__dirname, '..', 'frontend', 'src', 'i18n', 'fallback', 'en.json');

const zh = JSON.parse(fs.readFileSync(zhPath, 'utf8'));
const en = JSON.parse(fs.readFileSync(enPath, 'utf8'));

let sql = '';
for (const [key, value] of Object.entries(zh)) {
  const escaped = String(value).replace(/'/g, "''");
  sql += `INSERT INTO translations (locale, key, value) VALUES ('zh-TW', '${key}', '${escaped}') ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value;\n`;
}
for (const [key, value] of Object.entries(en)) {
  const escaped = String(value).replace(/'/g, "''");
  sql += `INSERT INTO translations (locale, key, value) VALUES ('en', '${key}', '${escaped}') ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value;\n`;
}

fs.writeFileSync(path.join(__dirname, '..', 'backend', 'migrations', '009_reseed_from_json.sql'), sql);
console.log(`Generated ${Object.keys(zh).length} zh-TW + ${Object.keys(en).length} en = ${Object.keys(zh).length + Object.keys(en).length} entries`);
