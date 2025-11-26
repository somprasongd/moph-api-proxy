// รวมฟังก์ชันจัดการการเรียก HTTP ไปยังระบบภายนอกพร้อมรีทรายและการจัดการโทเคน
const axios = require('axios');
const axiosRetry = require('axios-retry');
const https = require('https');
const jwt_decode = require('jwt-decode');

const { createAuthPayload } = require('../helper/auth-payload');
const cache = require('../cache');

const {
  MOPH_CLAIM_API,
  MOPH_PHR_API,
  EPIDEM_API,
  FDH_API,
  FDH_AUTH,
  FDH_AUTH_SECRET,
  MOPH_IC_API,
  MOPH_IC_AUTH,
  MOPH_IC_AUTH_SECRET,
  HTTP_TIMEOUT_MS,
  TOKEN_KEY,
  AUTH_PAYLOAD_KEY,
} = require('../config');

const httpsAgent = new https.Agent({
  keepAlive: true,
});

const NETWORK_ERROR_CODES = new Set(['ECONNRESET', 'ETIMEDOUT', 'EAI_AGAIN']);

// applyNetworkRetry ใช้ axios-retry เพื่อรีทรายเมื่อเครือข่ายสะดุดหรือ call ซ้ำได้
function applyNetworkRetry(client) {
  axiosRetry(client, {
    retries: 3,
    retryDelay: axiosRetry.exponentialDelay,
    shouldResetTimeout: true,
    retryCondition: (error) => {
      const isTimeout = error.code === 'ECONNABORTED';

      return (
        axiosRetry.isNetworkOrIdempotentRequestError(error) ||
        (error.code && NETWORK_ERROR_CODES.has(error.code)) ||
        isTimeout
      );
    },
  });
}

const commonHeaders = {
  'Content-Type': 'application/json',
  Accept: 'application/json',
  'Accept-Encoding': 'gzip, deflate',
};

const defaultClientOptions = {
  headers: commonHeaders,
  timeout: Number(HTTP_TIMEOUT_MS) || 15000,
  httpsAgent,
  maxRedirects: 0,
};

const getTokenClient = axios.create({
  baseURL: MOPH_IC_AUTH,
  ...defaultClientOptions,
});
applyNetworkRetry(getTokenClient);

const getTokenClientFDH = axios.create({
  baseURL: FDH_AUTH,
  ...defaultClientOptions,
});
applyNetworkRetry(getTokenClientFDH);

// getToken รับผิดชอบดึง JWT สำหรับแต่ละระบบ พร้อม cache และ refresh ให้อัตโนมัติ
async function getToken(
  options = { force: false, username: '', password: '', app: 'mophic' }
) {
  const {
    force = false,
    username = '',
    password = '',
    app = 'mophic',
  } = options;
  // console.log('gettoken with options:', options);
  const tokenKey = `${app}${TOKEN_KEY}`;
  const authPayloadKey = `${app}${AUTH_PAYLOAD_KEY}`;
  const secretKey = app === 'mophic' ? MOPH_IC_AUTH_SECRET : FDH_AUTH_SECRET;

  let token = null;
  if (force) {
    cache.del(tokenKey);
  } else {
    token = await cache.get(tokenKey);
  }
  if (token === null || token === '') {
    try {
      const url = `/token?Action=get_moph_access_token`;
      let payload = {};
      if (username !== '' && password !== '') {
        // กรณีมี username/password ใหม่ ให้สร้าง payload แล้วเก็บไว้
        payload = createAuthPayload(username, password, secretKey);
      } else {
        const strPayload = await cache.get(authPayloadKey);
        // not logged in
        if (!strPayload) {
          return null;
        }
        payload = JSON.parse(strPayload);
      }

      // console.log('get token with payload', payload);
      const client = app === 'mophic' ? getTokenClient : getTokenClientFDH;
      const response = await client.post(url, payload);
      token = response.data;
      const decoded = jwt_decode(token);
      console.log(`New ${app} token expires at`, decoded.exp);

      cache.setex(tokenKey, token, decoded.exp - 60); // set expire before 60s
      // เก็บ payload ที่ใช้สร้างโทเคนไว้เพื่อง่ายต่อการ refresh รอบถัดไป
      cache.set(authPayloadKey, JSON.stringify(payload));
    } catch (error) {
      console.error(error);
    }
  }
  return token;
}

const defaultOptions = {
  baseURL: MOPH_IC_API,
  ...defaultClientOptions,
};

const instance = axios.create(defaultOptions);
applyNetworkRetry(instance);

// const controller = new AbortController();

// interceptor ฝั่ง request จะเติม Bearer token เข้าไปทุกครั้งก่อนส่ง
instance.interceptors.request.use(async (config) => {
  const token = await getToken({ app: 'mophic' });
  if (!token) {
    return Promise.reject({
      message:
        'Cannot create token, please check the username and password configuration.',
    });
  }

  // console.log('interceptors.request', `Bearer ${token}`);
  config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// interceptor ฝั่ง response จะลอง refresh token และเรียกซ้ำเมื่อได้ 401
instance.interceptors.response.use(null, async (error) => {
  if (error.config && error.response && error.response.status === 401) {
    const token = await getToken({ force: true, app: 'mophic' });
    if (!token) {
      console.log('Cancal Retry from interceptors.response', error);
      return Promise.reject(error);
    }

    // console.log('interceptors.response', `Bearer ${token}`);
    error.config.headers.Authorization = `Bearer ${token}`;
    console.log('Retry from interceptors.response');
    return axios.request(error.config);
  }

  return Promise.reject(error);
});

const epidemOptions = {
  baseURL: EPIDEM_API,
  ...defaultClientOptions,
};

const instanceEpidem = axios.create(epidemOptions);
applyNetworkRetry(instanceEpidem);

// ขั้นตอนสำหรับ EPIDEM จะเหมือน MOPH IC แต่ใช้ base URL ต่างกัน
instanceEpidem.interceptors.request.use(async (config) => {
  const token = await getToken({ app: 'mophic' });
  if (!token) {
    return Promise.reject({
      message:
        'Cannot create token, please check the username and password configuration.',
    });
  }
  // console.log('interceptors.request', `Bearer ${token}`);
  config.headers.Authorization = `Bearer ${token}`;
  return config;
});

instanceEpidem.interceptors.response.use(null, async (error) => {
  if (error.config && error.response && error.response.status === 401) {
    const token = await getToken({ force: true, app: 'mophic' });
    if (!token) {
      console.log('Cancal Retry from interceptors.response', error);
      return Promise.reject(error);
    }

    // console.log('interceptors.response', `Bearer ${token}`);
    error.config.headers.Authorization = `Bearer ${token}`;
    // console.log('Retry from interceptors.response');
    return axios.request(error.config);
  }

  return Promise.reject(error);
});

const phrOptions = {
  baseURL: MOPH_PHR_API,
  ...defaultClientOptions,
};

const instancePhr = axios.create(phrOptions);
applyNetworkRetry(instancePhr);

instancePhr.interceptors.request.use(async (config) => {
  // PHR ใช้โทเคนของ mophic เช่นเดียวกับ MOPH IC
  const token = await getToken({ app: 'mophic' });
  if (!token) {
    return Promise.reject({
      message:
        'Cannot create token, please check the username and password configuration.',
    });
  }
  config.headers.Authorization = `Bearer ${token}`;
  return config;
});

instancePhr.interceptors.response.use(null, async (error) => {
  if (
    error.config &&
    error.response &&
    (error.response.status === 401 || error.response.status === 501)
  ) {
    // หากฝั่ง PHR ตอบ 401/501 ให้ refresh token แล้วเรียกซ้ำ
    const token = await getToken({ force: true, app: 'mophic' });
    if (!token) {
      console.log('Cancal Retry from interceptors.response', error);
      return Promise.reject(error);
    }
    error.config.headers.Authorization = `Bearer ${token}`;
    return axios.request(error.config);
  }

  return Promise.reject(error);
});

const claimOptions = {
  baseURL: MOPH_CLAIM_API,
  ...defaultClientOptions,
};

const instanceClaim = axios.create(claimOptions);
applyNetworkRetry(instanceClaim);

instanceClaim.interceptors.request.use(async (config) => {
  // กลุ่ม Claim ต้องใช้ token จากระบบ FDH
  const token = await getToken({ app: 'fdh' });
  if (!token) {
    return Promise.reject({
      message:
        'Cannot create token, please check the username and password configuration.',
    });
  }
  config.headers.Authorization = `Bearer ${token}`;
  return config;
});

instanceClaim.interceptors.response.use(null, async (error) => {
  if (error.config && error.response && error.response.status === 401) {
    // ถ้าหมดอายุให้บังคับสร้าง token FDH ใหม่แล้วลองใหม่
    const token = await getToken({ force: true, app: 'fdh' });
    if (!token) {
      console.log('Cancal Retry from interceptors.response', error);
      return Promise.reject(error);
    }
    error.config.headers.Authorization = `Bearer ${token}`;
    return axios.request(error.config);
  }

  return Promise.reject(error);
});

const fdhOptions = {
  baseURL: FDH_API,
  ...defaultClientOptions,
};

const instanceFDH = axios.create(fdhOptions);
applyNetworkRetry(instanceFDH);

instanceFDH.interceptors.request.use(async (config) => {
  // เรียกข้อมูล FDH โดยใช้ token ของตัวเอง
  const token = await getToken({ app: 'fdh' });
  if (!token) {
    return Promise.reject({
      message:
        'Cannot create token, please check the username and password configuration.',
    });
  }
  // console.log('interceptors.request', `Bearer ${token}`);
  config.headers.Authorization = `Bearer ${token}`;
  return config;
});

instanceFDH.interceptors.response.use(null, async (error) => {
  if (error.config && error.response && error.response.status === 401) {
    // ถ้า token หมดอายุให้ refresh แล้วเรียกซ้ำโดยอัตโนมัติ
    const token = await getToken({ force: true, app: 'fdh' });
    if (!token) {
      console.log('Cancal Retry from interceptors.response', error);
      return Promise.reject(error);
    }

    // console.log('interceptors.response', `Bearer ${token}`);
    error.config.headers.Authorization = `Bearer ${token}`;
    // console.log('Retry from interceptors.response');
    return axios.request(error.config);
  }

  return Promise.reject(error);
});

function getClient(endpoint = 'mophic') {
  // คืน instance ให้ proxy ใช้ตามค่า endpoint ที่ผู้ใช้ระบุ
  switch (endpoint) {
    case 'epidem':
      return instanceEpidem;
    case 'phr':
      return instancePhr;
    case 'claim':
      return instanceClaim;
    case 'fdh':
      return instanceFDH;
    default:
      return instance;
  }
}

module.exports = {
  client: instance,
  clientEpidem: instanceEpidem,
  clientPhr: instancePhr,
  clientClaim: instanceClaim,
  clientFDH: instanceFDH,
  getToken,
  getClient,
};
