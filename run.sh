#!/bin/bash
# run.sh

# 1. Clean up
echo "1. Cleaning up..."
docker stop gophermart-db 2>/dev/null || true
docker rm gophermart-db 2>/dev/null || true

# 2. Start PostgreSQL
echo "2. Starting PostgreSQL..."
docker run --name gophermart-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=gophermart \
  -p 5433:5432 \
  -d postgres:latest

# 3. Wait for PostgreSQL to be ready
echo "3. Waiting for PostgreSQL..."
until docker exec gophermart-db pg_isready -U postgres; do
  echo "Waiting for PostgreSQL to be ready..."
  sleep 2
done

# 4. Verify
echo "4. Verifying setup..."
docker exec gophermart-db psql -U postgres -d gophermart -c "SELECT 'Setup complete' as status;"

echo ""
echo "✅ PostgreSQL is ready!"
echo "📊 Connection: postgres://postgres:postgres@localhost:5433/gophermart?sslmode=disable"
echo ""

# 5. Run application
echo "🚀 Starting Gophermart..."
go run cmd/gophermart/main.go
