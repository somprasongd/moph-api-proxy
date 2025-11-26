// ตัวกลางสำหรับส่งต่อคำขอไปยังปลายทางต่าง ๆ พร้อมจัดการไฟล์ multipart
const axios = require('axios');
const express = require('express');
const formidable = require('formidable');
const FormData = require('form-data');
const queryString = require('query-string');
const fs = require('fs');
const http = require('../../http');

const createFormidable = (options = {}) => {
  if (typeof formidable === 'function') {
    return formidable(options);
  }
  if (typeof formidable.formidable === 'function') {
    return formidable.formidable(options);
  }
  return new formidable.IncomingForm(options);
};

const router = express.Router();

router.all('*', async (req, res, next) => {
  const { query } = req;
  // เลือก client ตาม header หรือ query endpoint เพื่อยิงไปปลายทางที่ถูกต้อง
  const endpoint = req.header('x-api-endpoint') || query['endpoint'];
  const client = http.getClient(endpoint);

  if (query['endpoint']) {
    delete query['endpoint'];
  }

  // ประกอบ URL ใหม่จาก path เดิมและ query ที่เหลืออยู่
  const stringified = queryString.stringify(query);
  const url = `${req.params['0']}${
    stringified === '' ? '' : `?${stringified}`
  }`;

  try {
    let response;

    if (req.method === 'GET') {
      // ส่งผ่าน GET โดยตรงไม่มีการดัดแปลง body
      response = await client.get(url);
    } else if (req.method === 'POST') {
      // ตรวจสอบชนิดข้อมูลเพื่อแยกจัดการ JSON และ multipart
      const contentTypeHeader = req.get('Content-Type') || '';
      const mimeType = contentTypeHeader.split(';')[0].trim().toLowerCase();

      if (mimeType === 'application/json') {
        response = await client.post(url, req.body);
      } else if (mimeType === 'multipart/form-data') {
        // ใช้ formidable อ่านฟอร์มหรือไฟล์จากคำขอ
        const reqForm = createFormidable({
          multiples: true,
          // maxFileSize: 100 * 1024 * 1024, // 100MB default 200MB
        });
        const [fields, files] = await reqForm.parse(req);

        const form = new FormData();
        if (fields) {
          for (const key in fields) {
            if (Object.hasOwnProperty.call(fields, key)) {
              const element = fields[key];
              const value = Array.isArray(element) ? element[0] : element;
              // append ฟิลด์ธรรมดาเหมือนผู้ใช้ส่งมา
              form.append(key, value);
            }
          }
        }

        if (files) {
          for (const key in files) {
            if (Object.hasOwnProperty.call(files, key)) {
              const element = files[key];
              const appendFile = (file) => {
                // ใช้ stream เพื่อไม่โหลดไฟล์ทั้งหมดเข้าเมมโมรี
                form.append(key, fs.createReadStream(file.filepath), {
                  filename: file.originalFilename,
                  contentType: file.mimetype,
                });
              };

              if (Array.isArray(element)) {
                element.forEach(appendFile);
              } else {
                appendFile(element);
              }
            }
          }
        }

        response = await client.post(url, form, {
          // ใช้ header ใหม่จาก FormData เพื่อให้ boundary ถูกต้อง
          headers: form.getHeaders(),
        });
      } else {
        return res.status(415).json({
          message: `Unsupported Content-Type${
            mimeType ? `: ${mimeType}` : ''
          }`,
        });
      }
    } else if (req.method === 'PUT') {
      response = await client.put(url, req.body);
    } else if (req.method === 'PATCH') {
      response = await client.patch(url, req.body);
    } else if (req.method === 'DELETE') {
      response = await client.delete(url);
    } else {
      // ป้องกันการใช้ method แปลกเพื่อให้ behavior ชัดเจน
      return res.status(405).json({
        message:
          'proxy error: allow only GET, POST, PUT, PATCH and DELETE method.',
      }); // Method Not Allowed
    }

    return res.send(response.data);
  } catch (error) {
    if (axios.isAxiosError(error)) {
      if (error.code === 'ECONNABORTED') {
        // แยกกรณี timeout ให้ตอบ 504 ชัดเจน
        console.error('axios timeout:', error.message);
        return res.status(504).json({
          message: 'upstream request timeout',
          detail: error.message,
          endpoint,
          url,
          timeoutMs: error.config?.timeout,
        });
      }

      // log รายละเอียด axios error ให้ตรวจสอบ trace ได้ง่าย
      console.error('axios error.message:', error.message);
      console.error('axios error.code:', error.code);
      console.error('axios error.syscall:', error.syscall);
      console.error('axios error.errno:', error.errno);
      console.error('axios error.response?.status:', error.response?.status);
      console.error('axios error.response?.data:', error.response?.data);
      console.error('axios error.cause:', error.cause);
      if (typeof error.toJSON === 'function') {
        console.error('axios error.toJSON():', error.toJSON());
      }
    } else {
      console.error('non-axios error:', error);
    }
    next(error);
  }
});

module.exports = router;
