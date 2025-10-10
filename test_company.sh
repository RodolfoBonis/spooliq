#!/bin/bash
TOKEN=$(curl -s -X POST http://localhost:8000/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"dev@rodolfodebonis.com.br","password":"U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"}' | jq -r '.accessToken')

echo "Token extracted successfully"
echo ""
echo "Testing GET /v1/company/"
curl -v -X GET http://localhost:8000/v1/company/ \
  -H "Authorization: Bearer $TOKEN" 2>&1 | grep -A 20 "< HTTP"
