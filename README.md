# Mix Multiple Choice Test - Backend API

## Tổng quan

Mix Multiple Choice Test là một hệ thống backend API được xây dựng bằng Go (Golang) để quản lý các bài kiểm tra trắc nghiệm. Hệ thống cung cấp đầy đủ các chức năng để tạo, quản lý và thực hiện các bài kiểm tra trực tuyến với kiến trúc RESTful API.

### Tính năng chính

- **Quản lý người dùng**: Đăng ký, đăng nhập, phân quyền (User/Admin)
- **Quản lý câu hỏi**: CRUD câu hỏi với nhiều loại (Multiple Choice, True/False)
- **Quản lý đề thi**: Tạo đề thi từ ngân hàng câu hỏi, phân công cho người dùng
- **Thực hiện bài thi**: Bắt đầu, làm bài và nộp bài thi với tính thời gian
- **Chấm điểm tự động**: Tự động chấm điểm và tạo báo cáo kết quả
- **Thống kê và báo cáo**: Thống kê chi tiết về kết quả thi và hiệu suất

### Công nghệ sử dụng

- **Backend Framework**: Gin (Go web framework)
- **Database**: PostgreSQL với GORM ORM
- **Authentication**: JWT (JSON Web Tokens)
- **Caching**: Redis
- **Testing**: Testify framework
- **Documentation**: Swagger/OpenAPI
- **Logging**: Logrus

## Quick Start

### Prerequisites

- Go 1.23 or higher
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


## API Documentation Chi Tiết

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication

Hệ thống sử dụng JWT Bearer Token authentication. Để truy cập các API được bảo vệ, bạn cần include header:

```
Authorization: Bearer <your-jwt-token>
```

### Response Format

Tất cả API responses đều tuân theo format chuẩn:

**Success Response:**
```json
{
  "data": {...},
  "message": "Success message"
}
```

**Error Response:**
```json
{
  "error": "Error type",
  "code": "ERROR_CODE",
  "message": "Human readable error message",
  "details": "Additional error details (optional)"
}
```

## API Endpoints Chi Tiết

### Authentication APIs

#### POST /auth/register
Đăng ký tài khoản người dùng mới.

**Request Body:**
```json
{
  "email": "user@example.com",
  "username": "username",
  "password": "password123",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response (201 Created):**
```json
{
  "message": "User registered successfully",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data
- `409 Conflict`: Email or username already exists
- `500 Internal Server Error`: Registration failed

#### POST /auth/login
Đăng nhập và nhận JWT tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200 OK):**
```json
{
  "message": "Login successful",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Invalid credentials
- `500 Internal Server Error`: Login failed

#### POST /auth/refresh
Làm mới access token bằng refresh token.

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "message": "Token refreshed successfully",
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "token_type": "Bearer"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Invalid or expired refresh token
- `500 Internal Server Error`: Token refresh failed

#### POST /auth/logout
Đăng xuất và vô hiệu hóa refresh token.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Response (200 OK):**
```json
{
  "message": "Logout successful"
}
```

**Error Responses:**
- `401 Unauthorized`: Authentication required
- `500 Internal Server Error`: Logout failed

### User Management APIs

#### GET /users/profile
Lấy thông tin profile chi tiết của user hiện tại.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Response (200 OK):**
```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "username",
    "first_name": "John",
    "last_name": "Doe",
    "role": "user",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### PUT /users/profile
Cập nhật thông tin profile của user hiện tại.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Request Body:**
```json
{
  "first_name": "John Updated",
  "last_name": "Doe Updated",
  "username": "new_username"
}
```

**Response (200 OK):**
```json
{
  "message": "Profile updated successfully",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "new_username",
    "first_name": "John Updated",
    "last_name": "Doe Updated",
    "role": "user",
    "is_active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `409 Conflict`: Username already taken
- `500 Internal Server Error`: Profile update failed

#### GET /users (Admin only)
Lấy danh sách tất cả người dùng (chỉ admin).

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Query Parameters:**
- `page` (int, optional): Số trang (default: 1)
- `page_size` (int, optional): Số items per page (default: 10, max: 100)
- `search` (string, optional): Tìm kiếm theo tên, email, username

**Response (200 OK):**
```json
{
  "users": [
    {
      "id": 1,
      "email": "user@example.com",
      "username": "username",
      "first_name": "John",
      "last_name": "Doe",
      "role": "user",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1
}
```

#### POST /users/change-password (Admin only)
Thay đổi mật khẩu cho bất kỳ user nào (chỉ admin).

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Request Body:**
```json
{
  "user_id": 1,
  "new_password": "newpassword123"
}
```

**Response (200 OK):**
```json
{
  "message": "Password changed successfully"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Admin access required
- `404 Not Found`: User not found
- `500 Internal Server Error`: Password change failed

### Question Management APIs

#### GET /questions
Lấy danh sách câu hỏi với phân trang và filter.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Query Parameters:**
- `page` (int, optional): Số trang (default: 1)
- `page_size` (int, optional): Số items per page (default: 10, max: 100)
- `tags` (string, optional): Comma-separated list of tags
- `difficulty` (string, optional): easy, medium, hard
- `type` (string, optional): multiple_choice, true_false
- `search` (string, optional): Tìm kiếm trong title và content
- `is_active` (bool, optional): Filter theo trạng thái active

**Response (200 OK):**
```json
{
  "questions": [
    {
      "id": 1,
      "title": "What is Go?",
      "content": "Go is a programming language developed by Google. What type of language is it?",
      "type": "multiple_choice",
      "difficulty": "easy",
      "options": [
        {
          "id": "a",
          "text": "Interpreted language"
        },
        {
          "id": "b",
          "text": "Compiled language"
        },
        {
          "id": "c",
          "text": "Scripting language"
        }
      ],
      "tags": ["programming", "go", "basics"],
      "points": 1,
      "time_limit": 60,
      "is_active": true,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1
}
```

#### POST /questions (Admin only)
Tạo câu hỏi mới.

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Request Body:**
```json
{
  "title": "New Question Title",
  "content": "Question content goes here",
  "type": "multiple_choice",
  "difficulty": "medium",
  "options": [
    {
      "id": "a",
      "text": "Option A",
      "is_correct": false
    },
    {
      "id": "b",
      "text": "Option B",
      "is_correct": true
    },
    {
      "id": "c",
      "text": "Option C",
      "is_correct": false
    }
  ],
  "tags": ["tag1", "tag2"],
  "points": 2,
  "time_limit": 90,
  "explanation": "Explanation for the correct answer"
}
```

**Response (201 Created):**
```json
{
  "message": "Question created successfully",
  "question": {
    "id": 2,
    "title": "New Question Title",
    "content": "Question content goes here",
    "type": "multiple_choice",
    "difficulty": "medium",
    "options": [
      {
        "id": "a",
        "text": "Option A"
      },
      {
        "id": "b",
        "text": "Option B"
      },
      {
        "id": "c",
        "text": "Option C"
      }
    ],
    "tags": ["tag1", "tag2"],
    "points": 2,
    "time_limit": 90,
    "explanation": "Explanation for the correct answer",
    "is_active": true,
    "created_by": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

### Exam Management APIs

#### GET /exams
Lấy danh sách đề thi.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Query Parameters:**
- `page` (int, optional): Số trang (default: 1)
- `page_size` (int, optional): Số items per page (default: 10, max: 100)

**Response (200 OK):**
```json
{
  "exams": [
    {
      "id": 1,
      "title": "Basic Programming Quiz",
      "description": "A basic quiz covering fundamental programming concepts",
      "duration": 30,
      "total_points": 10,
      "pass_score": 70,
      "status": "active",
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-02T10:00:00Z",
      "is_active": true,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "user_exam": {
        "id": 1,
        "status": "assigned",
        "started_at": null,
        "completed_at": null,
        "expires_at": "2024-01-03T00:00:00Z",
        "attempt_count": 0,
        "max_attempts": 1
      }
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1
}
```

#### POST /exams (Admin only)
Tạo đề thi mới.

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Request Body:**
```json
{
  "title": "New Programming Exam",
  "description": "Advanced programming concepts exam",
  "duration": 60,
  "pass_score": 75,
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-02T10:00:00Z",
  "questions": [
    {
      "question_id": 1,
      "points": 3,
      "order": 1
    },
    {
      "question_id": 2,
      "points": 2,
      "order": 2
    }
  ]
}
```

**Response (201 Created):**
```json
{
  "message": "Exam created successfully",
  "exam": {
    "id": 2,
    "title": "New Programming Exam",
    "description": "Advanced programming concepts exam",
    "duration": 60,
    "total_points": 5,
    "pass_score": 75,
    "status": "draft",
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-02T10:00:00Z",
    "is_active": true,
    "created_by": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### POST /exams/{id}/start
Bắt đầu làm bài thi.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Path Parameters:**
- `id` (int): Exam ID

**Response (200 OK):**
```json
{
  "message": "Exam started successfully",
  "user_exam": {
    "id": 1,
    "status": "started",
    "started_at": "2024-01-01T10:00:00Z",
    "completed_at": null,
    "expires_at": "2024-01-03T00:00:00Z",
    "attempt_count": 1,
    "max_attempts": 2,
    "time_left": 1800
  },
  "questions": [
    {
      "id": 1,
      "title": "What is Go?",
      "content": "Go is a programming language...",
      "type": "multiple_choice",
      "difficulty": "easy",
      "options": [
        {
          "id": "a",
          "text": "Interpreted language"
        },
        {
          "id": "b",
          "text": "Compiled language"
        }
      ],
      "tags": ["programming", "go"],
      "points": 2,
      "time_limit": 60,
      "is_active": true,
      "created_by": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "time_left": 1800
}
```

#### POST /exams/{id}/submit
Nộp bài thi.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Path Parameters:**
- `id` (int): Exam ID

**Request Body:**
```json
{
  "answers": [
    {
      "question_id": 1,
      "selected_options": ["b"],
      "time_spent": 45
    },
    {
      "question_id": 2,
      "selected_options": ["a", "c"],
      "time_spent": 60
    }
  ]
}
```

**Response (200 OK):**
```json
{
  "message": "Exam submitted successfully",
  "result": {
    "id": 1,
    "user_id": 1,
    "exam_id": 1,
    "user_exam_id": 1,
    "exam_title": "Basic Programming Quiz",
    "score": 85.5,
    "total_points": 8,
    "max_points": 10,
    "passed": true,
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T10:30:00Z",
    "duration": 1800,
    "created_at": "2024-01-01T10:30:00Z"
  }
}
```

### Result Management APIs

#### GET /results
Lấy danh sách kết quả thi.

**Headers:**
```
Authorization: Bearer <access-token>
```

**Query Parameters:**
- `page` (int, optional): Số trang (default: 1)
- `page_size` (int, optional): Số items per page (default: 10, max: 100)
- `exam_id` (int, optional): Filter theo exam ID

**Response (200 OK):**
```json
{
  "results": [
    {
      "id": 1,
      "user_id": 1,
      "exam_id": 1,
      "user_exam_id": 1,
      "exam_title": "Basic Programming Quiz",
      "score": 85.5,
      "total_points": 8,
      "max_points": 10,
      "passed": true,
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T10:30:00Z",
      "duration": 1800,
      "created_at": "2024-01-01T10:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10,
  "total_pages": 1
}
```

#### GET /results/statistics (Admin only)
Lấy thống kê tổng quan về kết quả thi.

**Headers:**
```
Authorization: Bearer <admin-access-token>
```

**Response (200 OK):**
```json
{
  "exam_statistics": [
    {
      "exam_id": 1,
      "exam_title": "Basic Programming Quiz",
      "total_attempts": 10,
      "passed_attempts": 7,
      "failed_attempts": 3,
      "pass_rate": 70.0,
      "average_score": 78.5,
      "highest_score": 95.0,
      "lowest_score": 45.0,
      "average_duration": 1650
    }
  ],
  "overall_stats": {
    "total_exams": 5,
    "total_users": 15,
    "total_attempts": 25,
    "average_score": 76.8,
    "pass_rate": 68.0,
    "total_time_spent": 45000,
    "average_duration": 1800
  }
}
```

## Error Handling

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | INVALID_REQUEST | Request data không hợp lệ |
| 401 | UNAUTHORIZED | Chưa xác thực hoặc token không hợp lệ |
| 403 | FORBIDDEN | Không đủ quyền truy cập |
| 404 | NOT_FOUND | Resource không tồn tại |
| 409 | CONFLICT | Xung đột dữ liệu |
| 429 | RATE_LIMIT_EXCEEDED | Vượt quá giới hạn request |
| 500 | INTERNAL_SERVER_ERROR | Lỗi server nội bộ |

### Rate Limiting

API có rate limiting để bảo vệ hệ thống:

- **Login API**: 5 requests/minute per IP
- **Submit Exam API**: 10 requests/minute per user
- **General APIs**: 100 requests/minute per user

## Security

### Authentication & Authorization

- **JWT Tokens**: Sử dụng JWT cho authentication
- **Access Token**: Thời hạn 15 phút
- **Refresh Token**: Thời hạn 7 ngày, lưu trong Redis
- **Role-based Access**: User và Admin roles
- **Password Hashing**: Sử dụng bcrypt

### Input Validation

- Tất cả input đều được validate
- SQL Injection protection với GORM
- XSS protection với input sanitization
- Request size limiting

## Testing

### Unit Tests

Chạy unit tests:
```bash
go test ./tests/
```

Tests bao gồm:
- **Service Layer Tests**: AuthService, QuestionService, ExamService, ResultService
- **Handler Tests**: HTTP endpoint tests
- **Model Tests**: Database model validation

### Test Coverage

```bash
go test -cover ./...
```

Target coverage: >80% cho business logic
