// ส่วนติดต่อ Redis สำหรับจัดการ cache โทเคนและข้อมูลอื่น ๆ
// หาก Redis ใช้งานไม่ได้จะ fallback เป็น in-memory Map อัตโนมัติ
const redis = require('redis');

const { REDIS_HOST, REDIS_PORT, REDIS_PASSWORD } = require('../config');

let redisClient;
let backend = 'redis'; // redis | memory

const memoryStore = new Map();
const memoryTimers = new Map();

const clearTimer = (key) => {
  const timer = memoryTimers.get(key);
  if (timer) {
    clearTimeout(timer);
    memoryTimers.delete(key);
  }
};

const memoryGet = (key) =>
  new Promise((resolve) => {
    const record = memoryStore.get(key);
    if (!record) {
      return resolve(null);
    }
    const { value, expiresAt } = record;
    if (expiresAt && expiresAt <= Date.now()) {
      clearTimer(key);
      memoryStore.delete(key);
      return resolve(null);
    }
    return resolve(value);
  });

const memorySet = (key, value, expiresAtMs = null) =>
  new Promise((resolve) => {
    clearTimer(key);
    const record = { value, expiresAt: expiresAtMs };
    memoryStore.set(key, record);
    if (expiresAtMs) {
      const delay = Math.max(expiresAtMs - Date.now(), 0);
      const timer = setTimeout(() => {
        memoryStore.delete(key);
        memoryTimers.delete(key);
      }, delay);
      memoryTimers.set(key, timer);
    }
    resolve('OK');
  });

const memorySetex = (key, value, unixTimestamp) => {
  const expiresAtMs = Number(unixTimestamp) * 1000;
  if (Number.isNaN(expiresAtMs)) {
    return memorySet(key, value);
  }
  if (expiresAtMs <= Date.now()) {
    clearTimer(key);
    memoryStore.delete(key);
    return Promise.resolve('OK');
  }
  return memorySet(key, value, expiresAtMs);
};

const memoryDel = (key) =>
  new Promise((resolve) => {
    clearTimer(key);
    const deleted = memoryStore.delete(key);
    resolve(deleted ? 1 : 0);
  });

const useMemoryFallback = (reason) => {
  if (backend === 'memory') {
    return;
  }
  console.warn(`Cache fallback: ${reason} -> use in-memory store instead.`);
  backend = 'memory';
  if (redisClient) {
    try {
      redisClient.quit();
    } catch (error) {
      console.error('Error closing redis client:', error.message || error);
    } finally {
      redisClient = null;
    }
  }
};

const createClient = () =>
  new Promise((resolve) => {
    if (backend === 'memory') {
      return resolve();
    }

    if (!REDIS_HOST) {
      useMemoryFallback('Redis host is not configured');
      return resolve();
    }

    const option = {
      host: REDIS_HOST,
      port: REDIS_PORT,
    };

    if (REDIS_PASSWORD) {
      // หากมีการตั้งรหัสผ่าน Redis ให้แนบไปด้วย
      option.password = REDIS_PASSWORD;
    }

    try {
      redisClient = redis.createClient(option);
    } catch (error) {
      useMemoryFallback('Cannot create redis client');
      return resolve();
    }

    let settled = false;
    const finish = () => {
      if (settled) {
        return;
      }
      settled = true;
      resolve();
    };

    redisClient.once('ready', () => {
      console.log('Redis: connected.');
      // ล้างคีย์ token เดิมเพื่อกันข้อมูลค้างจากรอบก่อน
      redisClient.del('token');
      finish();
    });

    redisClient.on('error', (error) => {
      console.error('Redis error:', error.message || error);
      useMemoryFallback('Redis unavailable');
      finish();
    });

    redisClient.on('end', () => {
      console.warn('Redis connection ended.');
      useMemoryFallback('Redis connection closed');
    });
  });

const ensureBackend = () => backend === 'redis' && redisClient;

const get = (key) => {
  if (!ensureBackend()) {
    return memoryGet(key);
  }
  return new Promise((resolve, reject) => {
    redisClient.get(key, (err, reply) => {
      if (err) {
        console.log(err);
        useMemoryFallback('Redis get failed');
        return memoryGet(key).then(resolve).catch(reject);
      }
      if (reply === null) {
        return resolve(null);
      }
      resolve(reply);
    });
  });
};

const set = (key, value) => {
  if (!ensureBackend()) {
    return memorySet(key, value);
  }
  return new Promise((resolve, reject) => {
    redisClient.set(key, value, (err, reply) => {
      if (err) {
        console.log(err);
        useMemoryFallback('Redis set failed');
        return memorySet(key, value).then(resolve).catch(reject);
      }
      resolve(reply);
    });
  });
};

const setex = (key, value, unixTimestamp) => {
  if (!ensureBackend()) {
    return memorySetex(key, value, unixTimestamp);
  }
  return new Promise((resolve, reject) => {
    redisClient.set(key, value, (err, reply) => {
      if (err) {
        console.log(err);
        useMemoryFallback('Redis setex failed');
        return memorySetex(key, value, unixTimestamp)
          .then(resolve)
          .catch(reject);
      }

      redisClient.expireat(key, unixTimestamp, (expireErr) => {
        if (expireErr) {
          console.log(expireErr);
          useMemoryFallback('Redis expire failed');
          return memorySetex(key, value, unixTimestamp)
            .then(resolve)
            .catch(reject);
        }
        resolve(reply);
      });
    });
  });
};

const del = (key) => {
  if (!ensureBackend()) {
    return memoryDel(key);
  }
  return new Promise((resolve, reject) => {
    redisClient.del(key, (err, result) => {
      if (err) {
        useMemoryFallback('Redis delete failed');
        return memoryDel(key).then(resolve).catch(reject);
      }
      return resolve(result);
    });
  });
};

module.exports = { createClient, get, set, setex, del };
