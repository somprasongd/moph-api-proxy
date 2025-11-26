// กำหนดค่า environment และตรวจสอบตัวแปรสำคัญก่อนเริ่มระบบ
// const { createHmac } = require('crypto');

const env = {
  APP_PORT: process.env.APP_PORT || 3000,
  REDIS_HOST: process.env.REDIS_HOST,
  REDIS_PORT: process.env.REDIS_PORT,
  REDIS_PASSWORD: process.env.REDIS_PASSWORD || '',
  MOPH_CLAIM_API: process.env.MOPH_CLAIM_API || 'https://claim-nhso.moph.go.th',
  MOPH_PHR_API: process.env.MOPH_PHR_API || 'https://phr1.moph.go.th',
  EPIDEM_API:
    process.env.EPIDEM_API || 'https://epidemcenter.moph.go.th/epidem',
  FDH_API: process.env.FDH_API || 'https://fdh.moph.go.th',
  FDH_AUTH: process.env.FDH_AUTH || 'https://fdh.moph.go.th',
  FDH_AUTH_SECRET: process.env.FDH_AUTH_SECRET || '$jwt@moph#',
  MOPH_IC_API: process.env.MOPH_IC_API || 'https://cvp1.moph.go.th',
  MOPH_IC_AUTH: process.env.MOPH_IC_AUTH || 'https://cvp1.moph.go.th',
  MOPH_IC_AUTH_SECRET: process.env.MOPH_IC_AUTH_SECRET || '$jwt@moph#',
  MOPH_HCODE: process.env.MOPH_HCODE,
  USE_API_KEY: process.env.USE_API_KEY
    ? process.env.USE_API_KEY === 'true'
    : true,
  HTTP_TIMEOUT_MS: process.env.HTTP_TIMEOUT_MS || 30000,
  TOKEN_KEY: '-auth-token',
  AUTH_PAYLOAD_KEY: '-auth-payload',
};

const requireds = ['MOPH_HCODE'];

const errors = [];

requireds.forEach((key) => {
  if (!env[key]) {
    errors.push(key);
  }
});

if (errors.length > 0) {
  throw new Error(
    `Fatal error: Environment is required [${errors.join(', ')}]`
  );
}

// // hash password
// const hash = createHmac('sha256', MOPH_IC_AUTH_SECRET)
//   .update(MOPH_PASSWD)
//   .digest('hex');

module.exports = env;
