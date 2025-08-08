# Exam System Backend

A high-performance, scalable Multiple Choice Exam Mixing System built with Go, Gin, PostgreSQL, and Redis.

## Features

### 🔐 Authentication & Authorization
- JWT-based authentication with access and refresh tokens
- Role-based access control (Admin/User)
- Secure password hashing with bcrypt
- Token refresh mechanism
- Rate limiting for login and exam submission

### 📝 Question Management
- CRUD operations for questions
- Multiple choice and true/false question types
- Question tagging system for categorization
- Difficulty levels (Easy, Medium, Hard)
- Question search and filtering
- Bulk question import/export

### 📋 Exam Management
- Create exams with random question selection
- Time-limited exams with auto-submission
- Exam scheduling with start/end times
- Question mixing and randomization
- Exam assignment to specific users
- Draft and active exam states

### 📊 Results & Analytics
- Automatic scoring and grading
- Detailed result analysis
- Performance statistics
- Pass/fail determination
- Answer review with explanations
- Comprehensive reporting

### 🛡️ Security & Performance
- Rate limiting with Redis token bucket
- Request ID tracking for debugging
- Structured logging with context propagation
- Input validation and sanitization
- SQL injection prevention with GORM
- CORS support for frontend integration

## Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **Authentication**: JWT tokens
- **ORM**: GORM
- **Logging**: Logrus
- **Testing**: Testify
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- Redis 7+
- Make (optional, for convenience commands)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd exam-system-backend
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run database migrations**
   ```bash
   make migrate-up
   # or manually with migrate CLI
   ```

5. **Start the application**
   ```bash
   make run
   # or
   go run main.go
   ```

### Using Docker Compose (Recommended)

1. **Start all services**
   ```bash
   docker-compose up -d
   ```

2. **View logs**
   ```bash
   docker-compose logs -f backend
   ```

3. **Stop services**
   ```bash
   docker-compose down
   ```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Server port | `8080` |
| `GIN_MODE` | Gin mode (debug/release) | `debug` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL username | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `password` |
| `DB_NAME` | PostgreSQL database name | `exam_system` |
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `JWT_SECRET` | JWT signing secret | `your-secret-key` |
| `JWT_ACCESS_EXPIRY` | Access token expiry | `15m` |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry | `168h` |
| `RATE_LIMIT_LOGIN` | Login rate limit | `5` |
| `RATE_LIMIT_SUBMIT` | Submit rate limit | `10` |
| `RATE_LIMIT_WINDOW` | Rate limit window | `1m` |
| `LOG_LEVEL` | Log level | `info` |
| `LOG_FORMAT` | Log format (text/json) | `text` |

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication Endpoints

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "username",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

#### Refresh Token
```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "your-refresh-token"
}
```

#### Logout
```http
POST /auth/logout
Authorization: Bearer <access-token>
```

### Question Endpoints

#### Get Questions
```http
GET /questions?page=1&page_size=10&tags=programming,go&difficulty=easy
Authorization: Bearer <access-token>
```

#### Create Question (Admin only)
```http
POST /questions
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "title": "What is Go?",
  "content": "Go is a programming language developed by Google.",
  "type": "multiple_choice",
  "difficulty": "easy",
  "options": [
    {"id": "a", "text": "Interpreted language", "is_correct": false},
    {"id": "b", "text": "Compiled language", "is_correct": true},
    {"id": "c", "text": "Scripting language", "is_correct": false}
  ],
  "tags": ["programming", "go"],
  "points": 1,
  "time_limit": 60,
  "explanation": "Go is a compiled programming language."
}
```

### Exam Endpoints

#### Get Exams
```http
GET /exams?page=1&page_size=10
Authorization: Bearer <access-token>
```

#### Create Exam (Admin only)
```http
POST /exams
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "title": "Go Programming Quiz",
  "description": "Basic Go programming concepts",
  "duration": 30,
  "pass_score": 70,
  "question_ids": [1, 2, 3, 4, 5],
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T18:00:00Z"
}
```

#### Start Exam
```http
POST /exams/{id}/start
Authorization: Bearer <access-token>
```

#### Submit Exam
```http
POST /exams/{id}/submit
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "answers": [
    {"question_id": 1, "selected_option": "b"},
    {"question_id": 2, "selected_option": "a"}
  ]
}
```

### Result Endpoints

#### Get Results
```http
GET /results?page=1&page_size=10&exam_id=1
Authorization: Bearer <access-token>
```

#### Get User Statistics
```http
GET /results/my-statistics
Authorization: Bearer <access-token>
```

## Development

### Available Make Commands

```bash
make help                 # Show all available commands
make run                  # Run the application
make dev                  # Run with hot reload
make test                 # Run tests
make test-coverage        # Run tests with coverage
make build                # Build the application
make clean                # Clean build artifacts
make fmt                  # Format code
make lint                 # Lint code
make security             # Run security checks
make check                # Run all checks
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests in watch mode
make test-watch
```

### Database Migrations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Project Structure

```
backend/
├── config/           # Configuration management
├── handlers/         # HTTP request handlers
├── middleware/       # Custom middleware
├── migrations/       # Database migrations
├── models/          # Data models
├── services/        # Business logic
├── tests/           # Unit tests
├── utils/           # Utility functions
├── main.go          # Application entry point
├── Dockerfile       # Docker configuration
├── docker-compose.yml # Docker Compose configuration
├── Makefile         # Development commands
└── README.md        # This file
```

## Testing

The application includes comprehensive unit tests for:

- Authentication service and handlers
- Question service and handlers
- Exam service and handlers
- Result service and handlers
- Middleware functions
- Utility functions

### Test Coverage

Run tests with coverage to ensure code quality:

```bash
make test-coverage
```

This generates an HTML coverage report at `coverage.html`.

## Security

### Authentication
- JWT tokens with configurable expiry
- Secure password hashing with bcrypt
- Refresh token rotation
- Token blacklisting on logout

### Rate Limiting
- Login endpoint: 5 requests per minute
- Exam submission: 10 requests per minute
- Configurable rate limits per endpoint

### Input Validation
- Request payload validation
- SQL injection prevention
- XSS protection
- CORS configuration

## Performance

### Caching
- Redis for session management
- Rate limiting with token bucket
- Query result caching (configurable)

### Database
- Optimized queries with GORM
- Database connection pooling
- Proper indexing on frequently queried fields

### Monitoring
- Structured logging with request IDs
- Performance metrics
- Health check endpoints

## Deployment

### Docker Deployment

1. **Build and run with Docker Compose**
   ```bash
   docker-compose up -d
   ```

2. **Scale the application**
   ```bash
   docker-compose up -d --scale backend=3
   ```

### Production Considerations

1. **Environment Variables**
   - Use strong JWT secrets
   - Configure proper database credentials
   - Set appropriate rate limits

2. **Database**
   - Enable SSL connections
   - Configure connection pooling
   - Set up database backups

3. **Redis**
   - Configure persistence
   - Set up Redis clustering for high availability
   - Enable authentication

4. **Monitoring**
   - Set up log aggregation
   - Configure health checks
   - Monitor performance metrics

## API Response Format

All API responses follow a consistent format:

### Success Response
```json
{
  "message": "Operation successful",
  "data": { ... }
}
```

### Error Response
```json
{
  "error": true,
  "code": "ERROR_CODE",
  "message": "Human readable error message",
  "details": "Additional error details (optional)"
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the test files for usage examples

