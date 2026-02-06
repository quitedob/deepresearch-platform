# Repository Layer Tests

This directory contains the repository layer implementation and tests for the AI Research Platform.

## Running Tests

The repository tests require a PostgreSQL database. You can run the tests in two ways:

### Option 1: Using Docker Compose (Recommended)

1. Start the PostgreSQL container:
```bash
docker-compose up -d postgres
```

2. Create the test database:
```bash
docker exec -it ai-research-postgres psql -U postgres -c "CREATE DATABASE ai_research_test;"
```

3. Run the tests:
```bash
go test -v ./internal/repository/...
```

### Option 2: Using Local PostgreSQL

1. Ensure PostgreSQL is running locally on port 5432

2. Create the test database:
```bash
psql -U postgres -c "CREATE DATABASE ai_research_test;"
```

3. Set environment variables (optional):
```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=postgres
export TEST_DB_NAME=ai_research_test
```

4. Run the tests:
```bash
go test -v ./internal/repository/...
```

## Test Configuration

The tests use the following default configuration:
- Host: `localhost`
- Port: `5432`
- User: `postgres`
- Password: `postgres`
- Database: `ai_research_test`

You can override these defaults using environment variables:
- `TEST_DB_HOST`
- `TEST_DB_PORT`
- `TEST_DB_USER`
- `TEST_DB_PASSWORD`
- `TEST_DB_NAME`

## Test Coverage

The repository layer includes comprehensive tests for:

### ChatRepository
- Session CRUD operations
- Message persistence and retrieval
- Pagination support
- Transaction handling
- Soft delete functionality

### ResearchRepository
- Research session management
- Task tracking and status updates
- Result persistence
- Pagination support
- Transaction handling

## Implementation Details

Both repositories implement:
- Context-aware operations for cancellation and timeouts
- Transaction support for atomic operations
- Proper error handling with descriptive messages
- Efficient queries with indexes
- Pagination for large result sets
