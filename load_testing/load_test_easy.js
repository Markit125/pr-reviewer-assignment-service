import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// --- Конфигурация теста ---
export const options = {
  // Имитируем 10 "виртуальных пользователей" (ВУ)
  // в течение 30 секунд.
  vus: 10,
  duration: '30s',
  // Устанавливаем пороги (SLOs)
  // - 95% запросов должны быть быстрее 300мс (как в задании)
  // - 99.9% запросов должны быть успешными (как в задании)
  thresholds: {
    'http_req_duration': ['p(95)<300'],
    'http_req_failed': ['rate<0.001'],
  },
};

// --- Данные для теста ---
const BASE_URL = 'http://api:8080'; // 'api' - это имя сервиса в docker-compose
const TEAM_NAME = `load_test_team_${uuidv4()}`;
const AUTHOR_ID = `author_${uuidv4()}`;

// --- 1. SETUP: Выполняется один раз перед тестом ---
// Создаем команду и одного автора для всех ВУ
export function setup() {
  const teamPayload = JSON.stringify({
    team_name: TEAM_NAME,
    members: [
      { user_id: AUTHOR_ID, username: 'Load Test Author', is_active: true },
      { user_id: 'u1', username: 'Load Test Reviewer 1', is_active: true },
      { user_id: 'u2', username: 'Load Test Reviewer 2', is_active: true },
    ],
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/team/add`, teamPayload, params);
  check(res, { 'Setup: Team created successfully': (r) => r.status === 201 });

  // Передаем ID автора в основной тест
  return { authorID: AUTHOR_ID };
}

// --- 2. VU SCRIPT: Выполняется в цикле каждым ВУ ---
export default function (data) {
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  // --- Этап 1: Создание PR ---
  const prID = `pr-${uuidv4()}`;
  const createPayload = JSON.stringify({
    pull_request_id: prID,
    pull_request_name: 'Load Test PR',
    author_id: data.authorID,
  });

  const createRes = http.post(`${BASE_URL}/pullRequest/create`, createPayload, params);
  check(createRes, {
    'Create PR: status 201': (r) => r.status === 201,
  });

  // Пауза, имитирующая "работу"
  sleep(1); // 1 секунда

  // --- Этап 2: Мерж PR ---
  const mergePayload = JSON.stringify({
    pull_request_id: prID,
  });

  const mergeRes = http.post(`${BASE_URL}/pullRequest/merge`, mergePayload, params);
  check(mergeRes, {
    'Merge PR: status 200': (r) => r.status === 200,
  });

  sleep(1);
}