import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// --- Конфигурация теста ---
export const options = {
  thresholds: {
    // Те же SLI/SLO
    'http_req_duration': ['p(95)<300'],
    'http_req_failed': ['rate<0.001'],
  },
  // Ступенчатая нагрузка
  stages: [
    { duration: '30s', target: 50 }, // Плавный подъем до 50 ВУ за 30 сек
    { duration: '1m', target: 50 }, // Держим 50 ВУ в течение 1 минуты
    { duration: '10s', target: 0 },  // Плавное снижение
  ],
};

// --- Данные для теста ---
const BASE_URL = 'http://api:8080';
const TOTAL_TEAMS = 20;
const USERS_PER_TEAM = 10;

// --- 1. SETUP: Выполняется один раз ---
// Предварительно заполняем БД 20 командами и 200 пользователями
export function setup() {
  console.log('--- Начинаем фазу SETUP: Создание 20 команд и 200 пользователей ---');
  const params = { headers: { 'Content-Type': 'application/json' } };
  let activeUserIDs = []; // Сохраняем ID активных юзеров, они будут авторами PR

  for (let t = 0; t < TOTAL_TEAMS; t++) {
    const teamName = `team-${t}-${uuidv4()}`;
    let members = [];
    for (let u = 0; u < USERS_PER_TEAM; u++) {
      const userID = `user-${t}-${u}-${uuidv4()}`;
      // 9 из 10 пользователей будут активны
      const isActive = (u < (USERS_PER_TEAM - 1));
      
      members.push({
        user_id: userID,
        username: `User ${t}-${u}`,
        is_active: isActive,
      });

      if (isActive) {
        activeUserIDs.push(userID);
      }
    }

    const teamPayload = JSON.stringify({
      team_name: teamName,
      members: members,
    });

    // Создаем команду (и всех ее пользователей)
    const res = http.post(`${BASE_URL}/team/add`, teamPayload, params);
    check(res, { 'Setup: Team created': (r) => r.status === 201 });
  }
  
  console.log(`--- SETUP завершен: Создано ${activeUserIDs.length} активных пользователей ---`);
  return { authors: activeUserIDs }; // Передаем ID авторов в основной тест
}

// --- 2. VU SCRIPT: Выполняется в цикле каждым ВУ ---
export default function (data) {
  const params = { headers: { 'Content-Type': 'application/json' } };
  
  // Выбираем случайного активного пользователя
  const authorID = randomItem(data.authors);
  
  // --- Имитация смешанной нагрузки ---
  const r = Math.random();

  if (r < 0.6) {
    // --- Сценарий 1 (60%): Create -> Merge ---
    const prID = `pr-${uuidv4()}`;
    const createPayload = JSON.stringify({
      pull_request_id: prID,
      pull_request_name: 'Heavy Load PR (Create/Merge)',
      author_id: authorID,
    });

    const createRes = http.post(`${BASE_URL}/pullRequest/create`, createPayload, params);
    check(createRes, { 'Create PR: status 201': (r) => r.status === 201 });
    sleep(1);

    const mergePayload = JSON.stringify({ pull_request_id: prID });
    const mergeRes = http.post(`${BASE_URL}/pullRequest/merge`, mergePayload, params);
    check(mergeRes, { 'Merge PR: status 200': (r) => r.status === 200 });
  
  } else if (r < 0.9) {
    // --- Сценарий 2 (30%): Create -> Reassign -> Merge ---
    const prID = `pr-${uuidv4()}`;
    const createPayload = JSON.stringify({
      pull_request_id: prID,
      pull_request_name: 'Heavy Load PR (Reassign)',
      author_id: authorID,
    });

    const createRes = http.post(`${BASE_URL}/pullRequest/create`, createPayload, params);
    check(createRes, { 'Create PR: status 201': (r) => r.status === 201 });
    
    let oldReviewer = null;
    try {
      oldReviewer = createRes.json('pr.assigned_reviewers.0');
    } catch (e) {} // Игнорируем, если ревьюер не назначен

    sleep(0.5);

    if(oldReviewer) {
      const reassignPayload = JSON.stringify({
        pull_request_id: prID,
        old_user_id: oldReviewer,
      });
      const reassignRes = http.post(`${BASE_URL}/pullRequest/reassign`, reassignPayload, params);
      // Мы ожидаем 200 (успех) или 409 (нет кандидата), оба являются "успешными" с т.з. нагрузки
      check(reassignRes, { 'Reassign PR: status 200 or 409': (r) => r.status === 200 || r.status === 409 });
      sleep(0.5);
    }
    
    const mergePayload = JSON.stringify({ pull_request_id: prID });
    const mergeRes = http.post(`${BASE_URL}/pullRequest/merge`, mergePayload, params);
    check(mergeRes, { 'Merge PR (after reassign): status 200': (r) => r.status === 200 });

  } else {
    // --- Сценарий 3 (10%): Get Reviews (Чтение) ---
    const randomUser = randomItem(data.authors);
    const getRes = http.get(`${BASE_URL}/users/getReview?user_id=${randomUser}`);
    check(getRes, { 'Get Reviews: status 200': (r) => r.status === 200 });
  }

  sleep(1);
}