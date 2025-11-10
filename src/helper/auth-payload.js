// ฟังก์ชันสร้าง payload สำหรับขอรับโทเคนและตรวจสอบข้อมูลใน cache
const config = require('../config');
const cache = require('../cache');
const { hashPassword } = require('./password');

function createAuthPayload(username, password, secretKey) {
  // แปลงข้อมูลผู้ใช้ให้กลายเป็น payload ที่ API ภายนอกต้องการ
  return {
    user: username,
    password_hash: hashPassword(password, secretKey),
    hospital_code: config.MOPH_HCODE,
  };
}
async function isCurrentAuthPayload(app = 'mophic', username, password) {
  // ตรวจสอบว่าข้อมูล username/password ที่กรอกตรงกับ payload ล่าสุดหรือไม่
  const strPayload = await cache.get(`${app}${config.AUTH_PAYLOAD_KEY}`);
  // console.log('strPayload', strPayload);
  if (!strPayload) {
    return false;
  }
  const secretKey =
    app === 'mophic' ? config.MOPH_IC_AUTH_SECRET : config.FDH_AUTH_SECRET;
  // console.log('payload', JSON.stringify(createAuthPayload(username, password)));
  return (
    strPayload ===
    JSON.stringify(createAuthPayload(username, password, secretKey))
  );
}

module.exports = {
  createAuthPayload,
  isCurrentAuthPayload,
};
