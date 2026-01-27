#!/bin/bash

# Примеры команд hey для нагрузочного тестирования сервиса сокращения URL

echo "=== Тестирование эндпоинта /ping ==="
echo "hey -n 1000 -c 50 http://localhost:8099/ping"
hey -n 1000 -c 50 http://localhost:8099/ping

echo -e "\n=== Тестирование POST /api/shorten с высокой нагрузкой ==="
echo "hey -n 500 -c 25 -m POST -d '{\"url\":\"https://www.example.com\"}' -H \"Content-Type: application/json\" http://localhost:8099/api/shorten"
hey -n 500 -c 25 -m POST -d '{"url":"https://www.example.com"}' -H "Content-Type: application/json" http://localhost:8099/api/shorten

echo -e "\n=== Тестирование с ограничением по времени (30 секунд) ==="
echo "hey -z 30s -c 10 http://localhost:8099/ping"
hey -z 30s -c 10 http://localhost:8099/ping

echo -e "\n=== Тестирование GET /{id} для существующего короткого URL (предполагая, что один был создан ранее) ==="
echo "hey -n 100 -c 20 http://localhost:8099/{some_existing_short_id}"
# ЗАМЕЧАНИЕ: Замените {some_existing_short_id} на реальный существующий короткий ID, если он есть

echo -e "\n=== Тестирование с пользовательским User-Agent ==="
echo "hey -n 50 -c 10 -H \"User-Agent: LoadTester/1.0\" http://localhost:8099/ping"
hey -n 50 -c 10 -H "User-Agent: LoadTester/1.0" http://localhost:8099/ping

echo -e "\n=== Тестирование с таймаутом на каждый запрос ==="
echo "hey -n 20 -c 5 -t 10 http://localhost:8099/ping"
hey -n 20 -c 5 -t 10 http://localhost:8099/ping