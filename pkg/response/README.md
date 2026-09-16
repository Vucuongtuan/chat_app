# Response Package

The `response` package provides utilities for formatting and sending consistent HTTP responses throughout the application. It ensures all API responses follow a uniform structure.

## Files

### `response.go`
Implements response formatting and helper functions.

**Functions:**
- `Success(c *gin.Context, statusCode int, data interface{}, message string)` - Send success response
- `Error(c *gin.Context, statusCode int, errorCode string, message string)` - Send error response
- `Paginated(c *gin.Context, statusCode int, data interface{}, pagination *Pagination, message string)` - Send paginated response
- `Message(c *gin.Context, statusCode int, message string)` - Send message-only response

**Responsibilities:**
- Format consistent JSON responses
- Include appropriate HTTP status codes
- Handle error responses
- Support pagination
- Add response metadata

## Response Structure

### Success Response

```json
{
  "success": true,
  "status_code": 200,
  "message": "Operation successful",
  "data": { ... },
  "timestamp": "2024-09-14T15:30:00Z"
}
```

### Error Response

```json
{
  "success": false,
  "status_code": 400,
  "error": "validation_error",
  "message": "Invalid email format",
  "timestamp": "2024-09-14T15:30:00Z"
}
```

### Paginated Response

```json
{
  "success": true,
  "status_code": 200,
  "message": "Users retrieved",
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 150,
    "total_pages": 8
  },
  "timestamp": "2024-09-14T15:30:00Z"
}
```

## Response Structures

### ResponseBody
```go
type ResponseBody struct {
    Success    bool        `json:"success"`
    StatusCode int         `json:"status_code"`
    Message    string      `json:"message"`
    Data       interface{} `json:"data,omitempty"`
    Error      string      `json:"error,omitempty"`
    Timestamp  time.Time   `json:"timestamp"`
}
```

### Pagination
```go
type Pagination struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}
```

## Usage Examples

### Success Response
```go
func GetUser(c *gin.Context) {
    user := &model.User{...}
    response.Success(c, http.StatusOK, user, "User retrieved successfully")
}
```

### Error Response
```go
func Login(c *gin.Context) {
    // Invalid credentials
    response.Error(c, http.StatusUnauthorized, "invalid_credentials", 
        "Email or password is incorrect")
}
```

### Paginated Response
```go
func ListUsers(c *gin.Context) {
    users := []*model.User{...}
    pagination := &response.Pagination{
        Page:       1,
        PerPage:    20,
        Total:      150,
        TotalPages: 8,
    }
    response.Paginated(c, http.StatusOK, users, pagination, "Users retrieved")
}
```

### Message-only Response
```go
func DeleteUser(c *gin.Context) {
    // No data to return
    response.Message(c, http.StatusNoContent, "User deleted successfully")
}
```

## HTTP Status Codes

| Code | Meaning | Use Case |
|------|---------|----------|
| 200 | OK | Successful GET/PUT/PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Invalid input/validation error |
| 401 | Unauthorized | Missing/invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 500 | Server Error | Unexpected error |

## Error Codes

Standardized error codes for client handling:

- `validation_error` - Input validation failed
- `invalid_credentials` - Login failed
- `unauthorized` - Not authenticated
- `forbidden` - Insufficient permissions
- `not_found` - Resource not found
- `duplicate_resource` - Resource already exists
- `server_error` - Unexpected server error
- `database_error` - Database operation failed
- `external_service_error` - Third-party service error

## Response Helpers

### CreateResponse
```go
resp := response.CreateResponse(true, 200, "Success", data, "", time.Now())
c.JSON(200, resp)
```

### CreateErrorResponse
```go
resp := response.CreateErrorResponse(false, 400, "validation_error", 
    "Invalid email format", time.Now())
c.JSON(400, resp)
```

### CalculatePagination
```go
page := 1
perPage := 20
total := 150
pagination := response.CalculatePagination(page, perPage, total)
// pagination.TotalPages = 8
```

## Consistent Formatting

All responses include:
- **Timestamp** - When response was generated
- **Status Code** - HTTP status in JSON (for client convenience)
- **Success Flag** - Boolean indicating success/failure
- **Message** - Human-readable message
- **Data** - Actual response data (if applicable)
- **Error** - Error code (for error responses)

## Custom Response Types

For special cases, extend the response package:

```go
type LoginResponse struct {
    ResponseBody
    AccessToken  string `json:"access_token,omitempty"`
    RefreshToken string `json:"refresh_token,omitempty"`
    ExpiresIn    int    `json:"expires_in,omitempty"`
}
```

## Best Practices

1. Always include consistent structure
2. Provide clear error messages
3. Use appropriate HTTP status codes
4. Include pagination metadata for list endpoints
5. Timestamp all responses
6. Use standardized error codes
