// จัดการสร้างและตรวจสอบ API Key ที่ใช้ป้องกันการเข้าถึงระบบ
const fs = require('fs');
const path = require('path');
const util = require('util');
const uuidAPIKey = require('uuid-apikey');

const readFile = util.promisify(fs.readFile);
const writeFile = util.promisify(fs.writeFile);

const keygenFile = path.join(
  __dirname,
  '..',
  '..',
  '.authorized_key',
  '.access.key'
);

if (!fs.existsSync(path.dirname(keygenFile))) {
  // สร้างโฟลเดอร์เก็บไฟล์ key ล่วงหน้าหากยังไม่มี
  fs.mkdirSync(path.dirname(keygenFile));
}

let secret;
let apiKey;
async function init() {
  try {
    // พยายามอ่าน secret ที่เคยสร้างไว้ก่อน
    secret = await readFile(keygenFile, { encoding: 'utf8' });
    apiKey = uuidAPIKey.toAPIKey(secret);
  } catch (readError) {
    try {
      // ถ้าไม่มีให้สุ่ม key ใหม่แล้วบันทึกเป็นไฟล์
      const key = uuidAPIKey.create();
      await writeFile(keygenFile, key.uuid, { encoding: 'utf8' });
      secret = key.uuid;
      apiKey = key.apiKey;
    } catch (writeError) {
      console.error('Unable to initialize API key store', writeError);
      throw writeError;
    }
  }
  // console.log(`Your api key is ${apiKey}`);
  console.log(`Enable API Key: true`);
}

function verify(apiKey) {
  // ตรวจสอบรูปแบบและเทียบ UUID เพื่อยืนยันว่า key ถูกต้อง
  if (!apiKey || !uuidAPIKey.isAPIKey(apiKey)) {
    return false;
  }

  if (uuidAPIKey.toUUID(apiKey) !== secret) {
    return false;
  }

  return true;
}

function getApiKey() {
  // ใช้ในหน้าเว็บเพื่อแสดงคีย์ปัจจุบันให้ผู้ดูแล
  return apiKey;
}

module.exports = {
  init,
  verify,
  getApiKey,
};
