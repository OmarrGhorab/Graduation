@echo off
REM Script to apply payment service migrations to Docker PostgreSQL
REM Usage: apply-payment-migrations.bat

echo.
echo 🚀 Applying Payment Service Migrations...
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

REM Copy migration files to container
echo 📋 Copying migration files to container...
docker cp payment-service/migrations/002_add_cart_and_subscriptions.sql %POSTGRES_CONTAINER%:/tmp/
docker cp payment-service/migrations/003_add_payment_methods.sql %POSTGRES_CONTAINER%:/tmp/
echo ✓ Migration files copied
echo.

REM Apply migrations
echo 🔧 Applying migration 002: Cart and Subscriptions...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -f /tmp/002_add_cart_and_subscriptions.sql
echo ✓ Migration 002 applied successfully
echo.

echo 🔧 Applying migration 003: Payment Methods...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -f /tmp/003_add_payment_methods.sql
echo ✓ Migration 003 applied successfully
echo.

REM Verify tables were created
echo 🔍 Verifying new tables...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -c "\dt"
echo.

echo ✅ All migrations applied successfully!
echo.
echo New tables created:
echo   - carts
echo   - cart_items
echo   - subscriptions
echo   - payment_methods
echo   - payment_order_items
echo.
echo You can now use the cart and subscription features! 🎉
echo.
pause
