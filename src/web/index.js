// กำหนดเส้นทางเว็บ UI สำหรับตั้งค่าระบบและแสดงข้อมูลต่าง ๆ
const express = require('express');
const config = require('../config');
const home = require('./home');
const apiKey = require('./api-key');
const changePwd = require('./change-password');

function init(appName) {
  const router = express.Router();

  // หน้าแรกแสดงข้อมูลรวมของระบบ
  router.use(home.init(appName));
  // หน้าเปลี่ยนรหัสผ่าน/โทเคน
  router.use(changePwd);
  if (config.USE_API_KEY) {
    // เฉพาะเมื่อเปิดใช้ API Key จึงให้เข้าหน้าขอดูคีย์ได้
    router.use(apiKey);
  }

  return router;
}

module.exports = { init };
