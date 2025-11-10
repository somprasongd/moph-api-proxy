// ฟังก์ชันแฮชรหัสผ่านด้วย HMAC-SHA256 เพื่อใช้กับบริการ MOPH
const { createHmac } = require('crypto');

function hashPassword(password, secretKey) {
  // ใช้ secret ของแต่ละระบบผูกกับรหัสผ่าน เพื่อให้ payload สอดคล้องกับ API ต้นทาง
  return createHmac('sha256', secretKey).update(password).digest('hex');
}

module.exports = {
  hashPassword,
};
