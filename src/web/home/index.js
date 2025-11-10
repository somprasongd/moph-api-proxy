// หน้าโฮมเพจแสดงรายการ API และรายละเอียดปลายทางแต่ละระบบ
const express = require('express');
const config = require('../../config');

function init(appName) {
  const router = express.Router();

  router.get('/', (req, res) => {
    const apis = [];
    // ดึงรายการ API ที่เปิดใช้งานตาม config เพื่อแสดงบนหน้าแรก
    if (config.MOPH_IC_API) {
      apis.push({
        tag: 'MOPH IC',
        name: 'MOPH Immunization Center',
        description: 'API MOPH Immunization Center',
        proxy_host: config.MOPH_IC_API,
        url: 'api/ImmunizationTarget?cid=xxxxxxxxxxxxx',
        doc: 'https://docs.google.com/document/d/1Inyhfrte0pECsD8YoForTL2W8B2hOxezf0GpTGEjJr8/edit',
      });
    }
    if (config.EPIDEM_API) {
      apis.push({
        tag: 'MOPH IC',
        name: 'EPIDEM Center',
        description: 'API EPIDEM Center',
        proxy_host: config.EPIDEM_API,
        url: 'api/SendEPIDEM?endpoint=epidem',
        doc: 'https://ddc.moph.go.th/viralpneumonia/file/g_surveillance/g_api_epidem_0165.pdf',
      });
    }
    if (config.MOPH_PHR_API) {
      apis.push({
        tag: 'MOPH IC',
        name: 'MOPH-PHR',
        description: 'ส่งข้อมูลเข้าระบบ MOPH-PHR',
        proxy_host: config.MOPH_PHR_API,
        url: 'api/RequestTokenv1?endpoint=phr',
        doc: 'https://docs.google.com/document/d/1ZWCBJnxVCtjqmBGNjj1sLnYv11dVzUvnweui-26NDJ0/edit',
      });
    }
    if (config.MOPH_CLAIM_API) {
      apis.push({
        tag: 'MOPH IC',
        name: 'MOPH Claim-NHSO',
        description: '(DMHT/EPI/dT Services)',
        proxy_host: config.MOPH_CLAIM_API,
        url: 'api/v1/opd/service-admissions/dmht?endpoint=claim',
        doc: 'https://docs.google.com/document/d/1iiybB2y7NJkEhXTdS4DbYe3-Fs7aka7MlEns81lzODQ/edit',
      });
    }
    if (config.FDH_API) {
      apis.push({
        tag: 'FDH',
        name: 'Financial Data Hub (FDH)',
        description:
          'ศูนย์กลางข้อมูลด้านการเงิน Financial Data Hub (FDH) กระทรวงสาธารณสุข',
        proxy_host: config.FDH_API,
        url: 'api/v1/data_hub/16_files?endpoint=fdh',
        doc: 'https://drive.google.com/file/d/17XqRmSEOnXoJVwzmCwteuVdy-Gp_SUyW',
      });
    }
    if (config.FDH_API) {
      apis.push({
        tag: 'FDH',
        name: 'Minimal Data Set',
        description: 'สำหรับเชื่อมต่อการจองเคลมผ่าน Financial Data Hub',
        proxy_host: config.FDH_API,
        url: 'api/v1/reservation?endpoint=fdh',
        doc: 'https://docs.google.com/document/d/1yDwflxOG_EG9HkEWbewfk446kmkGQ_2KiJhyqg2iY2E',
      });
    }
    res.render('index', {
      title: appName,
      useApiKey: config.USE_API_KEY,
      host: req.protocol + '://' + req.get('host'),
      apis,
    });
  });

  return router;
}

module.exports = { init };
