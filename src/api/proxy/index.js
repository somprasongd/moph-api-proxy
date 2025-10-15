const axios = require('axios');
const express = require('express');
const formidable = require('formidable');
const FormData = require('form-data');
const queryString = require('query-string');
const fs = require('fs');
const http = require('../../http');

const router = express.Router();

router.all('*', async (req, res, next) => {
  // // allow get and post and put
  // if (req.method !== 'GET' && req.method !== 'POST' && req.method !== 'PUT') {
  //   res
  //     .sendStatus(405)
  //     .json({ message: 'proxy error: allow only GET, POST and PUT method.' }); // Method Not Allowed
  //   return;
  // }

  const { query } = req;
  const endpoint = req.header('x-api-endpoint') || query['endpoint'];
  const client = http.getClient(endpoint);

  if (query['endpoint']) {
    delete query['endpoint'];
  }

  const stringified = queryString.stringify(query);
  const url = `${req.params['0']}${
    stringified === '' ? '' : `?${stringified}`
  }`;

  try {
    let respone;

    if (req.method === 'GET') {
      respone = await client.get(url);
    } else if (req.method === 'POST') {
      contentTypes = req.get('Content-Type').split(';');
      if (contentTypes[0] === 'application/json') {
        respone = await client.post(url, req.body);
      } else if (contentTypes[0] === 'multipart/form-data') {
        const reqForm = formidable.formidable({
          multiples: true,
          // maxFileSize: 100 * 1024 * 1024, // 100MB default 200MB
        });
        const [fields, files] = await reqForm.parse(req);

        const form = new FormData();
        if (fields) {
          for (const key in fields) {
            if (Object.hasOwnProperty.call(fields, key)) {
              const element = fields[key];
              form.append(key, element[0]);
            }
          }
        }

        if (files) {
          for (const key in files) {
            if (Object.hasOwnProperty.call(files, key)) {
              const element = files[key];
              if (Array.isArray(element)) {
                for (const v of element) {
                  form.append(key, fs.createReadStream(v.filepath), {
                    filename: v.originalFilename,
                    contentType: v.mimetype,
                  });
                }
              } else {
                form.append(key, element.filepath, {
                  filename: element.originalFilename,
                  contentType: v.mimetype,
                });
              }
            }
          }
        }
        respone = await client.post(url, form, {
          headers: {
            'Content-Type': req.get('Content-Type'),
          },
        });
      } else {
        next(new Error(`Unsupport Content-Type: ${contentTypes[0]}`));
      }
    } else if (req.method === 'PUT') {
      respone = await client.put(url, req.body);
    } else if (req.method === 'PATCH') {
      respone = await client.patch(url, req.body);
    } else if (req.method === 'DELETE') {
      respone = await client.delete(url);
    } else {
      res.sendStatus(405).json({
        message:
          'proxy error: allow only GET, POST, PUT, PATCH and DELETE method.',
      }); // Method Not Allowed
      return;
    }

    return res.send(respone.data);
  } catch (error) {
    if (axios.isAxiosError(error)) {
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
