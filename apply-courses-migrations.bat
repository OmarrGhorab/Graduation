@echo off
REM Script to apply courses attendance service migrations to Docker PostgreSQL
REM Usage: apply-courses-migrations.bat

echo.
echo 🚀 Applying Courses Attendance Service Migrations...
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
echo 📋 Copying migration file 000033 to container...
docker cp courses-attendance-service/migrations/000033_add_group_image_to_courses.up.sql %POSTGRES_CONTAINER%:/tmp/
echo ✓ Migration file copied
echo.

REM Apply migrations
echo 🔧 Applying migration 000033: Add Group Image...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -f /tmp/000033_add_group_image_to_courses.up.sql
echo ✓ Migration applied successfully
echo.

REM Verify columns
echo 🔍 Verifying courses table columns...
docker exec -i %POSTGRES_CONTAINER% psql -U graduation -d graduation -c "\d courses"
echo.

echo ✅ All migrations applied successfully!
echo.
pause
