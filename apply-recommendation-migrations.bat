@echo off
REM Script to apply recommendation service migrations to Docker PostgreSQL
REM Usage: apply-recommendation-migrations.bat

echo.
echo 🚀 Applying Recommendation Service Migrations (Chat Media Columns)...
echo.

REM Get postgres container ID
for /f "tokens=*" %%i in ('docker ps -q -f name^=postgres') do set POSTGRES_CONTAINER=%%i

if "%POSTGRES_CONTAINER%"=="" (
    echo ❌ Error: PostgreSQL container not found!
    echo Please start Docker services first: docker-compose up -d
    exit /b 1
)

echo ✓ Found PostgreSQL container: %POSTGRES_CONTAINER%
echo.

REM Apply migrations directly via psql
echo 🔧 Adding media_url and media_type columns to chat_messages table...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -c "ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media_url VARCHAR(500); ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS media_type VARCHAR(50);"

if %ERRORLEVEL% equ 0 (
    echo ✓ Columns added successfully or already exist.
) else (
    echo ❌ Failed to add columns.
)

echo.
echo 🔍 Verifying table structure...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -c "\d chat_messages"
echo.

echo ✅ Recommendation service database fix applied!
echo.
pause
