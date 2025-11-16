## Сервис Назначения Ревьюеров PR

### Обзор проекта

Микросервис на Go (Golang) для автоматического назначения ревьюверов на Pull Request'ы (PR) в рамках команды, согласно заданной бизнес-логике. Использует PostgreSQL для хранения данных. Весь стек запускается и управляется с помощью Docker Compose.

## Требования

Для запуска и тестирования необходимы Docker Desktop (или Docker Engine + Compose V2)

## Запуск

### 1. Настройка окружения

Скопируйте пример файла конфигурации и задайте свои данные в .env:

```bash
cp .env_example .env
```

### 2. Сборка и старт сервиса

Команда собирает Go-приложение, запускает контейнер БД и автоматически применяет все необходимые миграции базы данных.

```bash
docker compose up -d --build
```

### Тестирование API

Для проверки всех функций API (создание команд, назначение ревьюверов, переназначение) есть коллекция запросов в `test_requests/`.
Для их запуска можно использовать расширение VS Code [REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client).

### Управление и отчистка

Просмотр логов:
```bash
docker compose logs -f api
```

Остановка сервиса:
```bash
docker compose down
```

Остановка и отчистка бд:
```bash
docker compose down -v
```

## Нагрузочное тестирование

### Базовый тест (10 ВУ, 30 сек, 1 команда)

#### Конфигурация

- Сценарий: load-test_easy.js.

- Конфигурация: 10 ВУ в течение 30 секунд.

- Цели (SLI/SLO): p(95) < 300ms, rate < 0.1%

```bash
  █ THRESHOLDS 

    http_req_duration
    ✓ 'p(95)<300' p(95)=15.43ms

    http_req_failed
    ✓ 'rate<0.001' rate=0.00%


  █ TOTAL RESULTS 

    checks_total.......: 301     9.892876/s
    checks_succeeded...: 100.00% 301 out of 301
    checks_failed......: 0.00%   0 out of 301

    ✓ Setup: Team created successfully
    ✓ Create PR: status 201
    ✓ Merge PR: status 200

    HTTP
    http_req_duration..............: avg=12.07ms min=2.8ms med=12.34ms max=21.52ms p(90)=14.66ms p(95)=15.43ms
      { expected_response:true }...: avg=12.07ms min=2.8ms med=12.34ms max=21.52ms p(90)=14.66ms p(95)=15.43ms
    http_req_failed................: 0.00%  0 out of 301
    http_reqs......................: 301    9.892876/s

    EXECUTION
    iteration_duration.............: avg=2.02s   min=2.01s med=2.02s   max=2.03s   p(90)=2.03s   p(95)=2.03s  
    iterations.....................: 150    4.930004/s
    vus............................: 10     min=10       max=10
    vus_max........................: 10     min=10       max=10

    NETWORK
    data_received..................: 113 kB 3.7 kB/s
    data_sent......................: 73 kB  2.4 kB/s




running (0m30.4s), 00/10 VUs, 150 complete and 0 interrupted iterations
default ✓ [ 100% ] 10 VUs  30s
```

### Выводы

- SLI Успешности (99.9%): ВЫПОЛНЕНО (http_req_failed: 0.00%).

- SLI Времени Ответа (p95 < 300ms): ВЫПОЛНЕНО (p(95): 15.43ms).

- Итог: Сервис полностью удовлетворяет SLI при базовой нагрузке.


### Усиленный тест (50 ВУ, 1.5 мин, 200 пользователей)

#### Конфигурация

- Сценарий: load-test-heavy.js (или load_test.js с тяжелым сценарием)

- Setup: 20 команд и 200 пользователей (180 активных) созданы до старта.

- Конфигурация: Ступенчатая нагрузка (рамп-ап до 50 ВУ, удержание 1 мин).

- Цели (SLI/SLO): p(95) < 300ms, rate < 0.1%

### Результаты
```bash
  █ THRESHOLDS 

    http_req_duration
    ✓ 'p(95)<300' p(95)=6.46ms

    http_req_failed
    ✓ 'rate<0.001' rate=0.00%


  █ TOTAL RESULTS 

    checks_total.......: 4665    45.843895/s
    checks_succeeded...: 100.00% 4665 out of 4665
    checks_failed......: 0.00%   0 out of 4665

    ✓ Setup: Team created
    ✓ Create PR: status 201
    ✓ Merge PR: status 200
    ✓ Reassign PR: status 200 or 409
    ✓ Merge PR (after reassign): status 200
    ✓ Get Reviews: status 200

    HTTP
    http_req_duration..............: avg=4.41ms min=643.58µs med=4.45ms max=23.07ms p(90)=5.89ms p(95)=6.46ms
      { expected_response:true }...: avg=4.41ms min=643.58µs med=4.45ms max=23.07ms p(90)=5.89ms p(95)=6.46ms
    http_req_failed................: 0.00%  0 out of 4665
    http_reqs......................: 4665   45.843895/s

    EXECUTION
    iteration_duration.............: avg=1.9s   min=1s       med=2.01s  max=2.03s   p(90)=2.01s  p(95)=2.01s 
    iterations.....................: 2128   20.912285/s
    vus............................: 3      min=2         max=50
    vus_max........................: 50     min=50        max=50

    NETWORK
    data_received..................: 2.7 MB 27 kB/s
    data_sent......................: 1.2 MB 12 kB/s




running (1m41.8s), 00/50 VUs, 2128 complete and 0 interrupted iterations
default ✓ [ 100% ] 00/50 VUs  1m40s
```

### Выводы
- SLI Успешности (99.9%): Да

    - http_req_failed: 0.00%. 0 ошибок на 4665 запросов.

- SLI Времени Ответа (p95 < 300ms): Да

    - http_req_duration (p(95)): 6.46ms. 

- Итог: Сервис показал себя хорошо. Под нагрузкой в 50 одновременных юзеров, с заполненной базой и разными сценариями (Create, Merge, Reassign), 95% запросов выполнялись быстрее 6.5мс.