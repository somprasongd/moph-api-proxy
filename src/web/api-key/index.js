// จัดการฟอร์มให้ผู้ใช้ขอดู API Key ด้วยการยืนยันตัวตน
const express = require('express');
const keygen = require('../../helper/keygen');
const { isCurrentAuthPayload } = require('../../helper/auth-payload');

const router = express.Router();

router.all('/api-key', async (req, res) => {
  const payload = {
    status: '',
    message: '',
  };

  if (req.method === 'POST') {
    const { username, password } = req.body;
    // ตรวจสอบว่าข้อมูลเข้าสอดคล้องกับ payload ที่เคยสร้าง token ไว้
    let isMatch = await isCurrentAuthPayload('mophic', username, password);
    if (!isMatch) {
      isMatch = await isCurrentAuthPayload('fdh', username, password);
    }
    if (!isMatch) {
      payload.status = 'error';
      payload.message = 'Invalid username or password';
    } else {
      // เมื่อยืนยันตัวตนได้ให้ดึง API Key ปัจจุบันมาส่งให้ผู้ใช้
      payload.status = 'success';
      payload.message = keygen.getApiKey();
    }
  }

  res.render('api-key', payload);
});

module.exports = router;
