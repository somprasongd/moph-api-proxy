// ตรวจสอบ API Key ที่ผู้ใช้ส่งมาก่อนจะเข้าถึงเส้นทาง API
const config = require('../config');
const keygen = require('../helper/keygen');

function validateApikey(req, res, next) {
  if (config.USE_API_KEY) {
    // รับค่า key จาก header หรือ query ให้รองรับทั้งสองรูปแบบ
    const apiKey = req.header('x-api-key') || req.query['x-api-key'];

    if (!apiKey) {
      // กรณีไม่ส่งมาให้หยุดก่อนถึง controller
      const err = new Error('Access denied. No x-api-key provided');
      err.status = 401; // 401 Unauthorized
      err.statusCode = 401;
      return next(err);
    }

    if (!keygen.verify(apiKey)) {
      // เมื่อ key ไม่ตรงไฟล์ที่กำหนด ให้แจ้ง error ชัดเจน
      const err = new Error('Invalid x-api-key');
      err.status = 400; // Bad Request
      err.statusCode = 400;
      return next(err);
    }

    // ป้องกันไม่ให้ query key หลุดไปยังระบบปลายทาง
    delete req.query['x-api-key'];
  }
  next();
}

module.exports = {
  validateApikey,
};
