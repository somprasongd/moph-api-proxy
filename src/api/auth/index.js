// จัดการการเปลี่ยนรหัสผ่านและสร้างโทเคนใหม่ให้ระบบภายนอก
const express = require('express');
const http = require('../../http');

const router = express.Router();

router.post('/change-password', async (req, res, next) => {
  const { username, password } = req.body;
  const app = req.query.app || 'mophic'; // mophic or fdh

  if (!username) {
    // ต้องระบุ username เพื่อทราบบัญชีที่จะเปลี่ยนโทเคน
    return res.status(400).json({
      error: {
        message: 'username is required',
      },
    });
  }

  if (!password) {
    // ไม่อนุญาตให้ใช้รหัสผ่านว่าง
    return res.status(400).json({
      error: {
        message: 'password is required',
      },
    });
  }

  // บังคับสร้างโทเคนใหม่ด้วยข้อมูลที่ส่งมาเพื่อรีเซ็ต credential
  const token = await http.getToken({ force: true, username, password, app });
  if (!token) {
    return res.status(401).json({
      error: {
        message: 'Invalid username or password',
      },
    });
  } else {
    return res.sendStatus(204);
  }
});

module.exports = router;
