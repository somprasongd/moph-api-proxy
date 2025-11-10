// รวมเส้นทาง API ทั้งหมดก่อนนำไปใช้ในแอปหลัก
const express = require('express');
const auth = require('./auth');
const proxy = require('./proxy');

function init() {
  const router = express.Router();

  // กลุ่มที่เกี่ยวกับการจัดการ credential
  router.use('/api/auth', auth);
  // ส่วน proxy จะรับทุกเส้นทาง API ที่เหลือ
  router.use(proxy);

  return router;
}

module.exports = { init };
