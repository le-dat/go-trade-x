# GoTradeX — Hướng Dẫn Toàn Diện Từ A Đến Z

> Giải thích toàn bộ source code từ setup đến kiến trúc hiện tại cho người mới bắt đầu viết API bằng Go.

---

## Mục Lục

1. [Tổng Quan Kiến Trúc](#1-tổng-quan-kiến-trúc)
2. [Cấu Trúc Thư Mục](#2-cấu-trúc-thư-mục)
3. [Bắt Đầu Từ Đâu — Flow Của Một Request](#3-bắt-đầu-từ-đâu--flow-của-một-request)
4. [API Gateway — Nơi Tiếp Nhận Request HTTP](#4-api-gateway--nơi-tiếp-nhận-request-http)
5. [Middleware — Bộ Lọc Trước Khi Vào Handler](#5-middleware--bộ-lọc-trước-khi-vào-handler)
6. [JWT — Xác Thực Người Dùng](#6-jwt--xác-thực-người-dùng)
7. [User Service — Giao Diện Gọi Server Khác (gRPC)](#7-user-service--giao-diện-gọi-server-khác-grpc)
8. [Layered Architecture — Handler → Service → Repository](#8-layered-architecture--handler--service--repository)
9. [Database — Migrations + PostgreSQL](#9-database--migrations--postgresql)
10. [Proto Files — Định Nghĩa Giao Diện gRPC](#10-proto-files--định-nghĩa-giao-diện-grpc)
11. [Docker — Đóng Gói Ứng Dụng](#11-docker--đóng-gói-ứng-dụng)
12. [CI/CD — Tự Động Build và Deploy](#12-cicd--tự-động-build-và-deploy)
13. [Config & Logger](#13-config--logger)
14. [Tổng Kết Flow Hoàn Chỉnh](#14-tổng-kết-flow-hoàn-chỉnh)

---

## 1. Tổng Quan Kiến Trúc

GoTradeX là một nền tảng giao dịch cryptocurrency real-time. Dưới đây là sơ đồ toàn bộ hệ thống:

```
Client (trình duyệt/app)
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (Gin)                         │  Port 8080
│              HTTP REST — nhận request từ client              │
└────────────────────────────┬────────────────────────────────┘
                             │ gRPC (khi cần gọi service khác)
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                  User Service (gRPC)                        │  Port 50051
│              Xử lý đăng ký, đăng nhập, số dư               │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                       PostgreSQL                             │  Port 5432
│                    Lưu trữ dữ liệu                          │
└─────────────────────────────────────────────────────────────┘

Các service khác (chưa implemented đầy đủ):
- Order Service (gRPC) — xử lý đặt lệnh
- Matching Engine — ghép lệnh (Kafka consumer)
- Market Service — dữ liệu thị trường real-time (WebSocket + Kafka)
```

### Các Service Đã Implement

| Service | Protocol | Port | Trạng thái |
|---------|----------|------|-----------|
| API Gateway | REST/Gin | 8080 | ✅ Hoàn chỉnh |
| User Service | gRPC | 50051 | ✅ Hoàn chỉnh |
| Order Service | gRPC + Kafka | 50052 | 🚧 Stub |
| Matching Engine | Kafka consumer | — | 🚧 Stub |
| Market Service | WebSocket + Kafka | 8081 | 🚧 Stub |

### Key Design Decisions

1. **Kafka cho order matching** — Decouples Order Service khỏi Matching Engine; cho phép replay để recovery state
2. **Per-symbol goroutines trong matching engine** — Tránh cross-symbol lock contention; scale tuyến tính với số symbols
3. **Heap-based order book** — O(log n) insert/remove; tự động có price-time priority (FIFO tại cùng price level)
4. **Transactional outbox** — Order + outbox message được viết trong cùng DB transaction; outbox relay publish lên Kafka. Ngăn "lost orders" khi service crash sau DB commit nhưng trước Kafka publish
5. **SELECT FOR UPDATE cho balance** — Row-level locking ngăn race conditions trên concurrent deductions
6. **segmentio/kafka-go** — Thư viện Go-native nhẹ hơn Sarama

---

## 2. Cấu Trúc Thư Mục

```
GoTradeX/
├── src/server/                              # Tất cả code Go nằm ở đây
│   ├── cmd/                                # Điểm bắt đầu (entry point) của mỗi service
│   │   ├── api-gateway/                     # Service HTTP #1
│   │   │   ├── main.go                      # ← Chạy cái này để start API Gateway
│   │   │   ├── handlers/                    # Xử lý request (auth, orders)
│   │   │   ├── middleware/                  # Bộ lọc trước khi vào handler
│   │   │   ├── clients/                     # Kết nối sang service khác (gRPC)
│   │   │   └── docs/                        # Swagger docs (tự động sinh)
│   │   ├── user-service/                    # Service gRPC #1
│   │   │   └── main.go                      # ← Chạy cái này để start User Service
│   │   ├── order-service/                   # 🚧 Stub (chưa implement)
│   │   ├── matching-engine/                 # 🚧 Stub
│   │   └── market-service/                  # 🚧 Stub
│   │
│   ├── internal/user/                       # Logic của User Service
│   │   ├── handler.go                       # Xử lý request gRPC (Register, Login...)
│   │   ├── service.go                       # Logic nghiệp vụ (mã hóa password, tạo token)
│   │   └── repository.go                    # Tương tác database (SELECT, INSERT...)
│   │
│   ├── pkg/                               # Code dùng chung cho nhiều service
│   │   ├── auth/jwt.go                      # Tạo và xác thực JWT token (HS256)
│   │   ├── config/config.go                 # Đọc biến môi trường (.env)
│   │   └── logger/logger.go                 # Zap logger singleton
│   │
│   ├── proto/                              # Định nghĩa gRPC interface
│   │   ├── user.proto                       # ← Sửa ở đây rồi chạy `make proto`
│   │   ├── user.pb.go                       # File tự động sinh (KHÔNG sửa tay)
│   │   └── user_grpc.pb.go                  # File tự động sinh
│   │
│   ├── migrations/                          # SQL migration (version control cho DB)
│   │   ├── 001_create_users.up.sql           # Tạo bảng users, balances
│   │   ├── 001_create_users.down.sql        # Rollback
│   │   └── 002_create_orders.up.sql         # Tạo bảng orders
│   │
│   ├── docker-compose.yml                   # Infrastructure (postgres + kafka)
│   ├── docker-compose.override.yml          # Dev: build local + expose ports
│   ├── docker-compose.prod.yml              # Prod: pull từ GHCR
│   ├── docker/Dockerfile                    # Multi-stage build
│   ├── Makefile                            # build, migrate, dev-up...
│   ├── go.mod                               # Module declaration + dependencies
│   └── go.work                             # Go workspace (monorepo)
│
├── .github/workflows/                       # CI/CD
│   ├── pr-check.yml                         # Chạy khi tạo PR
│   └── deploy.yml                          # Deploy khi push vào main
│
└── docs/                                   # Tài liệu kiến trúc
    ├── 01-infrastructure/                   # System overview, infrastructure docs
    ├── 02-api-gateway/                      # API Gateway chi tiết
    ├── 03-services/                         # User, Order, Matching, Market services
    ├── 04-devops/                           # Docker, CI/CD, Deploy, Nginx
    ├── 05-architecture/                     # Architecture decisions
    └── 06-security/                         # Security considerations
```

---

## 3. Bắt Đầu Từ Đâu — Flow Của Một Request

### Flow: POST /api/v1/auth/register

```
1. Client gửi HTTP POST /api/v1/auth/register {email, password}
          │
          ▼
2. Docker/Gin router nhận request
          │
          ▼
3. Middleware stack chạy lần lượt:
   ├── Recovery middleware  ──── Catch panic, trả 500 nếu crash
   ├── Logger middleware    ──── Log: method, path, status, latency
   └── RateLimiter middleware ── Block nếu > 100 req/s/IP
          │
          ▼
4. Router điều hướng đến authHandler.Register()
          │
          ▼
5. Handler gọi UserServiceClient.Register()  (hiện tại là mock)
          │
          ▼
6. Trả JSON response cho client
```

---

## 4. API Gateway — Nơi Tiếp Nhận Request HTTP

### 4.1 Entry Point — `cmd/api-gateway/main.go`

`main.go` là điểm khởi đầu của API Gateway. Nó thiết lập Gin router, đăng ký middleware, tạo gRPC client factory, và chạy HTTP server.

```go
package main

func main() {
    // 1. Load config từ .env
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // 2. Khởi tạo JWT manager (dùng để xác thực token)
    jwtMgr := auth.NewJWTManager(cfg.JWTSecret)

    // 3. Tạo gRPC client factory (mock hoặc thật)
    factory := clients.NewGRPCClientFactoryWithTimeout(5 * time.Second)

    // 4. Khởi tạo Gin router
    r := gin.New()

    // 5. Đăng ký middleware — MỌI request đều phải qua 3 cái này
    r.Use(middleware.Recovery(logger.Get()))      // 5a. Panic → 500
    r.Use(middleware.Logger(logger.Get()))       // 5b. Log request
    r.Use(middleware.RateLimiter())              // 5c. Giới hạn 100 req/s/IP

    // 6. Đăng ký routes
    setupRoutes(r, jwtMgr, factory)

    // 7. Chạy server trong goroutine riêng
    go func() {
        srv := &http.Server{
            Addr:    ":" + cfg.AppPort,  // port 8080
            Handler: r,
        }
        logger.Get().Info("starting api-gateway on " + cfg.AppPort)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()

    // 8. Đợi tín hiệu dừng (Ctrl+C) để shutdown graceful
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
}
```

### 4.2 Routes — `setupRoutes`

```go
func setupRoutes(r *gin.Engine, jwtMgr *auth.JWTManager, factory *clients.GRPCClientFactory) {
    // Tạo handlers — truyền jwtMgr và factory vào để dùng
    authHandler := handlers.NewAuthHandler(jwtMgr, factory)
    orderHandler := handlers.NewOrderHandler(jwtMgr, factory)

    // Health check — công khai, không cần auth
    r.GET("/healthz", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "ok"})
    })

    // Swagger UI — công khai
    r.GET("/swagger/*any", ...)

    api := r.Group("/api/v1")

    // AUTH routes — công khai (không qua middleware auth)
    authGroup := api.Group("/auth")
    {
        authGroup.POST("/register", authHandler.Register)
        authGroup.POST("/login",    authHandler.Login)
    }

    // ORDERS routes — BẮT BUỘC phải có token
    ordersGroup := api.Group("/orders")
    ordersGroup.Use(middleware.RequireAuth(jwtMgr))  // ← Áp dụng middleware auth
    {
        ordersGroup.POST("",    orderHandler.PlaceOrder)
        ordersGroup.GET("/:id", orderHandler.GetOrder)
    }
}
```

**Điểm quan trọng**: Route có `.Use(middleware.RequireAuth(...))` nghĩa là trước khi vào handler, request phải qua middleware đó. Route không có `.Use()` thì bypass middleware auth.

### 4.3 Route Structure Tổng Hợp

```
GET  /healthz                              → Health check (public)
GET  /swagger/*any                        → Swagger UI (public)

POST /api/v1/auth/register                → Tạo tài khoản mới (public)
POST /api/v1/auth/login                  → Đăng nhập, nhận JWT (public)

POST /api/v1/orders                       → Đặt lệnh (auth required)
GET  /api/v1/orders/:id                   → Xem lệnh theo ID (auth required)
```

---

## 5. Middleware — Bộ Lọc Trước Khi Vào Handler

Middleware là các hàm chạy TRƯỚC handler chính. Mỗi middleware nhận request, xử lý, rồi gọi `c.Next()` để chuyển sang middleware/handler tiếp theo.

### 5.1 Recovery Middleware — `middleware/recovery.go`

```go
func Recovery(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // defer: đảm bảo luôn chạy dù có panic hay không
        defer func() {
            if err := recover(); err != nil {
                // Log lỗi
                logger.Error("panic recovered",
                    zap.Any("error", err),
                    zap.String("path", c.Request.URL.Path),
                )
                // Trả 500 cho client, KHÔNG crash server
                c.AbortWithStatusJSON(http.StatusInternalServerError,
                    gin.H{"error": "internal server error"})
            }
        }()
        c.Next()  // ← Tiếp tục xử lý request
    }
}
```

**Ý nghĩa**: Nếu handler bị panic (ví dụ: null pointer), server không crash mà trả 500 và tiếp tục phục vụ request khác.

### 5.2 Logger Middleware — `middleware/logger.go`

```go
func Logger(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()  // ← Chạy handler trước
        // Sau khi handler xong, log thông tin request
        logger.Info("request",
            zap.String("method", c.Request.Method),
            zap.String("path", c.Request.URL.Path),
            zap.Int("status", c.Writer.Status()),
            zap.Duration("latency", time.Since(start)),
            zap.String("client_ip", c.ClientIP()),
        )
    }
}
```

**Ý nghĩa**: Log mọi request để debug và monitor. Dùng `c.Next()` để đo thời gian **sau khi** handler hoàn thành.

### 5.3 Rate Limiter Middleware — `middleware/ratelimiter.go`

```go
type limiter struct {
    mu        sync.Mutex
    limit     *rate.Limiter  // rate.Limiter từ thư viện golang.org/x/time/rate
    lastClean time.Time
}

var (
    limiters       = make(map[string]*limiter)
    globalLimiter  = rate.NewLimiter(100, 50)  // 100 req/s, burst 50
    cleanInterval  = 5 * time.Minute
)

func RateLimiter() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()  // Lấy IP của client
        l := getLimiter(ip) // Lấy hoặc tạo limiter cho IP đó

        if !l.Allow() {  // Nếu vượt quá rate limit
            c.AbortWithStatusJSON(http.StatusTooManyRequests,
                gin.H{"error": "rate limit exceeded"})
            return
        }
        c.Next()
    }
}
```

**Ý nghĩa**: Giới hạn 100 request/giây/IP. Burst 50 cho phép tạm thời vượt rate limit (ví dụ: burst request lúc start).

### 5.4 Auth Middleware — `middleware/auth.go`

```go
func RequireAuth(jwtMgr *auth.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Đọc header "Authorization: Bearer <token>"
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "missing authorization header"})
            return
        }

        // 2. Tách "Bearer <token>" → "<token>"
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "invalid authorization header format"})
            return
        }

        // 3. Validate token
        claims, err := jwtMgr.ValidateToken(parts[1])
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "invalid or expired token"})
            return
        }

        // 4. Lưu user info vào context để handler đọc được
        c.Set("userID", claims.UserID)
        c.Set("email", claims.Email)

        c.Next()  // ← Token hợp lệ, cho đi tiếp
    }
}
```

**Ý nghĩa**: Kiểm tra token trước khi cho phép truy cập route `/orders`. Handler đọc `userID` từ context (đã được set ở đây).

---

## 6. JWT — Xác Thực Người Dùng

### `pkg/auth/jwt.go`

```go
type JWTManager struct {
    secret      string          // Khóa bí mật, đọc từ JWT_SECRET env var
    expiryTime  time.Duration   // Thời gian hết hạn token
}

type Claims struct {
    UserID string  // UUID của user
    Email  string  // Email của user
    jwt.RegisteredClaims
}

// Tạo token mới — dùng khi user đăng nhập
func (j *JWTManager) GenerateToken(userID, email string) (string, error) {
    claims := Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiryTime)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    // Tạo token với thuật toán HS256 (HMAC SHA-256)
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(j.secret))
}

// Xác thực token — dùng khi middleware nhận request
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{},
        func(t *jwt.Token) (interface{}, error) {
            // Kiểm tra thuật toán — phòng chống algorithm substitution attack
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return []byte(j.secret), nil
        })
    // ... error handling
}
```

**Security điểm quan trọng**: Kiểm tra `t.Method.(*jwt.SigningMethodHMAC)` trong callback để đảm bảo kẻ tấn công không thể đổi thuật toán từ HS256 sang HS384/HS512 với key khác.

---

## 7. User Service — Giao Diện Gọi Server Khác (gRPC)

### 7.1 Client Interfaces — `clients/grpc.go`

```go
type UserServiceClient interface {
    Register(ctx context.Context, email, password string) (*RegisterResponse, error)
    Login(ctx context.Context, email, password string) (*LoginResponse, error)
    GetBalance(ctx context.Context, userID string) (*GetBalanceResponse, error)
    DeductBalance(ctx context.Context, userID, asset string, amount float64) (*DeductBalanceResponse, error)
    CreditBalance(ctx context.Context, userID, asset string, amount float64) (*CreditBalanceResponse, error)
}

type OrderServiceClient interface {
    PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*PlaceOrderResponse, error)
    GetOrder(ctx context.Context, orderID string) (*GetOrderResponse, error)
    CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)
}

type GRPCClientFactory struct {
    userClient   UserServiceClient
    orderClient  OrderServiceClient
}
```

**Interface này định nghĩa**: "API Gateway cần gọi User Service bằng cách nào". Nhờ interface, sau này đổi từ mock sang gRPC thật chỉ cần viết implementation mới, không cần sửa handler.

### 7.2 Mock Implementation — `clients/mock.go`

```go
type MockUserClient struct{}

func (m *MockUserClient) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
    // Mock: luôn thành công, trả về token giả
    token := "mock-jwt-token-" + email
    userID := uuid.New().String()
    return &LoginResponse{
        Token: token,
        UserId: userID,
        Email: email,
    }, nil
}
```

**Hiện tại**: API Gateway dùng mock client — không có kết nối gRPC thật. Khi cần kết nối thật, tạo file `grpc_client.go` implement interface và khởi tạo trong `NewGRPCClientFactory()`.

---

## 8. Layered Architecture — Handler → Service → Repository

Đây là pattern phổ biến nhất trong Go backend, giúp tách biệt trách nhiệm rõ ràng.

```
┌─────────────────────────────────────┐
│           Handler (gRPC)            │ ← Nhận request, validate input, map response
└─────────────────┬───────────────────┘
                  │ gọi
                  ▼
┌─────────────────────────────────────┐
│           Service                    │ ← Logic nghiệp vụ, bcrypt, JWT, business rules
└─────────────────┬───────────────────┘
                  │ gọi
                  ▼
┌─────────────────────────────────────┐
│          Repository                  │ ← Tương tác trực tiếp với PostgreSQL
└─────────────────────────────────────┘
```

### 8.1 Handler Layer — `internal/user/handler.go`

Handler nhận request từ gRPC client và gọi xuống service layer:

```go
type UserServiceServer struct {
    proto.UnimplementedUserServiceServer  // Embed để satisfy interface
    service Service                       // Phụ thuộc vào interface, không phải implementation
}

// Ví dụ: xử lý Register request
func (s *UserServiceServer) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
    // 1. Validate input
    if req.Email == "" || req.Password == "" {
        return nil, status.Errorf(codes.InvalidArgument, "email and password are required")
    }
    if len(req.Password) < 6 {
        return nil, status.Errorf(codes.InvalidArgument, "password must be at least 6 characters")
    }

    // 2. Gọi xuống service layer
    userID, err := s.service.Register(req.Email, req.Password)
    if err != nil {
        if err == ErrUserExists {
            return nil, status.Error(codes.AlreadyExists, "user already exists")
        }
        return nil, status.Error(codes.Internal, "failed to register user")
    }

    // 3. Trả response
    return &proto.RegisterResponse{
        UserId: userID,
        Email:  req.Email,
    }, nil
}
```

**Handler CHỈ làm**:
- Validate input (email có rỗng không, password đủ dài không)
- Gọi service layer
- Map kết quả sang proto response
- Trả đúng gRPC status code (`InvalidArgument`, `AlreadyExists`, `Internal`...)

### 8.2 Service Layer — `internal/user/service.go`

Chứa logic nghiệp vụ — nơi xảy ra "business rules":

```go
type Service interface {
    Register(email, password string) (string, error)     // string = userID
    Login(email, password string) (string, string, error)  // token, userID, error
    GetBalance(userID string) ([]Balance, error)
    DeductBalance(userID, asset string, amount float64) (float64, error)
    CreditBalance(userID, asset string, amount float64) (float64, error)
}

type userService struct {
    repo   Repository    // Phụ thuộc interface Repository
    jwtMgr *auth.JWTManager
    pepper []byte        // Secret pepper thêm vào password trước khi hash
}

func (s *userService) Register(email, password string) (string, error) {
    // 1. Hash password với bcrypt + pepper
    hashedPassword, err := bcrypt.GenerateFromPassword(
        append([]byte(password), s.pepper...), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }

    // 2. Lưu vào database qua repository
    userID, err := s.repo.CreateUser(email, string(hashedPassword))
    if err != nil {
        if s.repo.IsUniqueViolation(err) {  // Email đã tồn tại
            return "", ErrUserExists
        }
        return "", err
    }

    // 3. Tạo balance mặc định cho user mới (free $10k để test)
    defaultBalances := []Balance{
        {Asset: "USD", Available: 10000.0, Locked: 0},
        {Asset: "BTC", Available: 0.0, Locked: 0},
        {Asset: "ETH", Available: 0.0, Locked: 0},
    }
    for _, b := range defaultBalances {
        s.repo.CreateBalance(userID, b.Asset, b.Available, b.Locked)
    }

    return userID, nil
}

func (s *userService) Login(email, password string) (token, userID string, err error) {
    // 1. Tìm user theo email
    user, err := s.repo.GetUserByEmail(email)
    if err != nil {
        if err == ErrUserNotFound {
            return "", "", ErrInvalidCredentials
        }
        return "", "", err
    }

    // 2. So sánh password với hash trong DB
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash),
        append([]byte(password), s.pepper...)); err != nil {
        return "", "", ErrInvalidCredentials
    }

    // 3. Tạo JWT token
    token, err = s.jwtMgr.GenerateToken(user.ID, user.Email)
    return token, user.ID, nil
}
```

**Service CHỈ làm**:
- Hash password với bcrypt + pepper
- Validate business rules (email đã tồn tại chưa, password đúng không)
- Gọi repository để lưu/lấy data
- Tạo JWT token
- KHÔNG viết SQL trực tiếp

### 8.3 Repository Layer — `internal/user/repository.go`

Trực tiếp tương tác với PostgreSQL:

```go
type Repository interface {
    CreateUser(email, passwordHash string) (string, error)
    GetUserByEmail(email string) (*User, error)
    GetUserByID(id string) (*User, error)
    GetBalances(userID string) ([]Balance, error)
    DeductBalance(ctx context.Context, userID, asset string, amount float64) (float64, error)
    CreditBalance(ctx context.Context, userID, asset string, amount float64) (float64, error)
}

type postgresRepository struct {
    db *pgxpool.Pool  // Connection pool từ pgx
}

func (r *postgresRepository) DeductBalance(ctx context.Context, userID, asset string, amount float64) (float64, error) {
    tx, err := r.db.Begin(ctx)  // Bắt đầu transaction
    if err != nil {
        return 0, err
    }
    defer tx.Rollback(ctx)

    // 1. Lock dòng balance của user bằng SELECT FOR UPDATE
    //    Đây là critical — ngăn race condition khi 2 request cùng deduct 1 lúc
    var balance float64
    err = tx.QueryRow(ctx,
        `SELECT available FROM balances
         WHERE user_id = $1 AND asset = $2
         FOR UPDATE`,  // ← ROW LOCK — các transaction khác phải đợi
        userID, asset).Scan(&balance)
    if err != nil {
        return 0, err
    }

    // 2. Kiểm tra đủ tiền không
    if balance < amount {
        return 0, ErrInsufficientBalance
    }

    // 3. Trừ balance
    newBalance := balance - amount
    _, err = tx.Exec(ctx,
        `UPDATE balances SET available = $1 WHERE user_id = $2 AND asset = $3`,
        newBalance, userID, asset)
    if err != nil {
        return 0, err
    }

    // 4. Commit transaction
    if err := tx.Commit(ctx); err != nil {
        return 0, err
    }

    return newBalance, nil
}
```

**Race condition prevention**: `SELECT ... FOR UPDATE` khóa dòng balance trong database. Nếu request A đang deduct $500, request B phải đợi A commit xong mới được đọc balance — tránh trừ 2 lần.

---

## 9. Database — Migrations + PostgreSQL

### 9.1 Migration System

Migrations đánh số thứ tự, có UP (apply) và DOWN (rollback):

**`migrations/001_create_users.up.sql`**:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,           -- Email là duy nhất
    password_hash TEXT NOT NULL,          -- bcrypt hash
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE balances (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,  -- Xóa user → xóa balance
    asset TEXT NOT NULL,                   -- "USD", "BTC", "ETH"
    available NUMERIC(20, 8) NOT NULL DEFAULT 0,  -- Số dư khả dụng
    locked NUMERIC(20, 8) NOT NULL DEFAULT 0,     -- Số dư bị khóa (đang có lệnh đặt)
    PRIMARY KEY (user_id, asset)           -- Mỗi user có nhiều asset, mỗi asset 1 dòng
);

-- Index để query nhanh
CREATE INDEX idx_balances_user_id ON balances(user_id);
CREATE INDEX idx_users_email ON users(email);
```

**`migrations/001_create_users.down.sql`**:
```sql
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS users;
```

**`migrations/002_create_orders.up.sql`**:
```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key TEXT UNIQUE,          -- Đảm bảo đặt lệnh idempotent
    user_id UUID NOT NULL,
    symbol TEXT NOT NULL,                  -- "BTC-USD", "ETH-USD"
    side TEXT NOT NULL CHECK (side IN ('BUY', 'SELL')),
    type TEXT NOT NULL CHECK (type IN ('MARKET', 'LIMIT')),
    price NUMERIC(20, 8),                  -- NULL nếu là MARKET order
    quantity NUMERIC(20, 8) NOT NULL,      -- Số lượng mua/bán
    filled_qty NUMERIC(20, 8) NOT NULL DEFAULT 0,  -- Đã khớp bao nhiêu
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PARTIAL', 'FILLED', 'CANCELLED', 'REJECTED')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index để query order theo user, symbol, status
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_symbol ON orders(symbol);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_idempotency_key ON orders(idempotency_key);
```

**Chạy migrations**:
```bash
make migrate       # Chạy tất cả UP migrations
make migrate-down  # Rollback migration cuối cùng
```

### 9.2 PostgreSQL Connection — `cmd/user-service/main.go`

```go
func main() {
    cfg, _ := config.Load()

    // Tạo connection pool đến PostgreSQL
    // pgxpool tự quản lý connection reuse, không cần tạo connection mới mỗi request
    pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
    if err != nil {
        log.Fatal("unable to create connection pool", err)
    }
    defer pool.Close()

    // Health check: kiểm tra kết nối có OK không
    if err := pool.Ping(context.Background()); err != nil {
        log.Fatal("unable to ping database", err)
    }

    // Khởi tạo layers
    repo := userRepo.NewRepository(pool)
    svc := userSvc.NewService(repo, jwtMgr, pepper)
    handler := userHandler.NewUserServiceServer(svc)

    // Tạo gRPC server
    grpcServer := grpc.NewServer()
    proto.RegisterUserServiceServer(grpcServer, handler)

    // Listen on port 50051
    lis, _ := net.Listen("tcp", ":50051")
    grpcServer.Serve(lis)  // Blocking forever
}
```

---

## 10. Proto Files — Định Nghĩa Giao Diện gRPC

gRPC dùng Protocol Buffers (protobuf) để định nghĩa interface giữa client và server.

### `proto/user.proto`

```protobuf
syntax = "proto3";

package proto;

option go_package = "github.com/verno/gotradex/proto";

// Định nghĩa service — giống như interface trong Go
service UserService {
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Login(LoginRequest) returns (LoginResponse);
    rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
    rpc DeductBalance(DeductBalanceRequest) returns (DeductBalanceResponse);
    rpc CreditBalance(CreditBalanceRequest) returns (CreditBalanceResponse);
}

// Request/Response messages — kiểu dữ liệu được truyền qua mạng
message RegisterRequest {
    string email = 1;
    string password = 2;
}

message RegisterResponse {
    string user_id = 1;
    string email = 2;
}

message LoginRequest {
    string email = 1;
    string password = 2;
}

message LoginResponse {
    string token = 1;
    string user_id = 2;
    string email = 3;
}

message Balance {
    string asset = 1;
    double available = 2;
    double locked = 3;
}

message GetBalanceRequest {
    string user_id = 1;
}

message GetBalanceResponse {
    repeated Balance balances = 1;  // Mảng balance (USD, BTC, ETH)
}
```

### Sinh Code Từ Proto

```bash
make proto
```

Lệnh này chạy:
```bash
protoc --go_out=. --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    --go_opt=paths=source_relative proto/*.proto
```

Sau khi chạy, sinh ra:
- `user.pb.go` — Message types (RegisterRequest, Balance, etc.)
- `user_grpc.pb.go` — Server stub và Client stub

### Cách dùng Server Stub (trong `handler.go`)

```go
// Handler embed proto.UnimplementedUserServiceServer để satisfy interface
type UserServiceServer struct {
    proto.UnimplementedUserServiceServer
    service Service
}

// gRPC server tự gọi hàm này khi client gọi Register
func (s *UserServiceServer) Register(ctx context.Context, req *proto.RegisterRequest)
    (*proto.RegisterResponse, error) {
    // ... implement logic
}
```

---

## 11. Docker — Đóng Gói Ứng Dụng

### `docker/Dockerfile` (Multi-stage Build)

```dockerfile
# Stage 1: Build — biên dịch Go code
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git gcc musl-dev   # Cần git (private deps) và gcc (CGO)
WORKDIR /app

# Copy go.mod/go.sum trước — tận dụng Docker cache
COPY go.mod go.sum* ./
RUN go mod download                    # Download deps trước

COPY . .                               # Copy source
RUN go mod tidy

# Build service được chỉ định qua SERVICE_NAME arg
# Ví dụ: docker build --build-arg SERVICE_NAME=api-gateway
ARG SERVICE_NAME
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE_NAME}
# CGO_ENABLED=0: không dùng CGO → binary nhỏ hơn, chạy trên image minimal

# Stage 2: Runtime — image nhỏ gọn, chỉ chạy binary
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
```

### `docker-compose.yml` (Infrastructure)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: gotradex
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5436:5432"    # Local port 5436 → container port 5432
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  kafka:
    image: bitnamilegacy/kafka:3.7
    ports:
      - "9092:9092"
    environment:
      ALLOW_PLAINTEXT_LISTENER: yes
      KAFKA_CFG_NODE_ID: 0
      KAFKA_CFG_PROCESS_ROLES: controller,broker
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
```

---

## 12. CI/CD — Tự Động Build và Deploy

### `pr-check.yml` (chạy khi tạo PR)

```yaml
jobs:
  backend-check:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: src/server
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go mod download
      - run: go test -v ./...

  docker-dry-run:
    needs: backend-check
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - name: Build API Gateway image
        uses: docker/build-push-action@v5
        with:
          push: false               # Dry-run, không push
          tags: backend-test:latest
          build-args: SERVICE_NAME=api-gateway
```

### `deploy.yml` (chạy khi push vào main)

```yaml
jobs:
  build-and-push:
    strategy:
      matrix:
        service: [api-gateway, user-service, order-service, matching-engine, market-service]
    steps:
      - uses: actions/checkout@v4
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.repository_owner }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/${{ github.repository_owner }}/gotradex-${{ matrix.service }}:latest
          build-args: SERVICE_NAME=${{ matrix.service }}

  deploy:
    needs: build-and-push
    steps:
      - uses: appleboy/ssh-action@v1.0.3
        with:
          host: ${{ secrets.VPS_HOST }}
          username: ${{ secrets.VPS_USER }}
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd /opt/gotradex
            git pull origin main
            docker login ghcr.io -u ${{ github.repository_owner }} -p ${{ secrets.GITHUB_TOKEN }}
            make prod-pull && make prod-restart
```

---

## 13. Config & Logger

### `pkg/config/config.go`

```go
type Config struct {
    AppPort      string  // Port HTTP (8080)
    DatabaseURL  string  // PostgreSQL connection string
    KafkaBrokers string  // Kafka brokers
    RedisURL     string  // Redis (optional)
    JWTSecret    string  // BẮT BUỘC phải set
}

func Load() (*Config, error) {
    // Load .env file vào environment
    godotenv.Load(".env")

    cfg := &Config{
        AppPort:      getEnv("APP_PORT", "8080"),
        DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5436/gotradex?sslmode=disable"),
        KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
        RedisURL:     getEnv("REDIS_URL", ""),
        JWTSecret:    getEnv("JWT_SECRET", ""),
    }

    // BẮT BUỘC phải có JWT_SECRET
    if cfg.JWTSecret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }

    return cfg, nil
}
```

### `pkg/logger/logger.go`

```go
var (
    initOnce    sync.Once
    logInstance *zap.Logger
)

func Get() *zap.Logger {
    initOnce.Do(func() {
        cfg := zap.NewProductionConfig()
        cfg.EncoderConfig.TimeKey = "timestamp"
        cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
        logInstance, _ = cfg.Build()
    })
    return logInstance
}
```

**Singleton pattern**: `sync.Once` đảm bảo logger chỉ khởi tạo 1 lần, dù được gọi từ nhiều goroutine.

---

## 14. Tổng Kết Flow Hoàn Chỉnh

### Flow 1: Đăng Ký User Mới

```
Client
  │ POST /api/v1/auth/register {email, password}
  ▼
API Gateway main.go (Gin, port 8080)
  │ Recovery → Logger → RateLimiter (middleware stack)
  ▼
authHandler.Register()
  │ jwtMgr và factory được inject vào handler
  ▼
factory.UserClient.Register(ctx, email, password)
  │ Hiện tại gọi MockUserClient (mock.go)
  ▼
MockUserClient.Register()
  │ Không gọi gRPC thật, trả fake response
  │ Trong tương lai: gRPC call đến :50051
  ▼
Trả JSON {user_id, email} cho client
```

### Flow 2: Đặt Lệnh (Authenticated)

```
Client
  │ POST /api/v1/orders {symbol: "BTC-USD", side: "BUY", quantity: 0.5}
  │ Header: Authorization: Bearer <jwt_token>
  ▼
API Gateway main.go
  │ Recovery → Logger → RateLimiter → RequireAuth (middleware)
  ▼
AuthMiddleware.RequireAuth()
  │ 1. Tách "Bearer <token>"
  │ 2. jwtMgr.ValidateToken(token) → Claims{userID, email}
  │ 3. c.Set("userID", userID), c.Set("email", email)
  ▼
orderHandler.PlaceOrder()
  │ userID := c.Get("userID") // lấy từ context
  │ Gọi factory.OrderClient.PlaceOrder(req)
  ▼
Trả JSON response
```

### Flow 3: User Đăng Nhập (Khi có gRPC thật)

```
Client
  │ POST /api/v1/auth/login {email, password}
  ▼
API Gateway → authHandler.Login() → factory.UserClient.Login()
  │
  │ (gRPC call over TCP to :50051)
  ▼
User Service (gRPC, port 50051)
  │
  ├── handler.Login() — nhận proto.LoginRequest
  │     │ Validate email/password không rỗng
  │     ▼
  ├── service.Login() — business logic
  │     │ 1. repo.GetUserByEmail(email) → tìm user
  │     │ 2. bcrypt.CompareHashAndPassword → check password
  │     │ 3. jwtMgr.GenerateToken(userID, email) → tạo JWT
  │     ▼
  └── repository.GetUserByEmail() — PostgreSQL
        │ SELECT * FROM users WHERE email = $1
        ▼
  └── Trả proto.LoginResponse{token, user_id, email}
        │
        ▼
API Gateway nhận response
  ▼
Trả JSON cho client {token, user_id, email}
```

---

## Cách Chạy Project

```bash
# 1. Copy env file
cp src/server/.env.example src/server/.env
# Sửa JWT_SECRET trong .env

# 2. Start infrastructure (postgres + kafka)
make docker-up

# 3. Apply migrations
make migrate

# 4. Build tất cả services
make build

# 5. Start tất cả services (dev mode)
make dev-up

# 6. Hoặc chạy từng service riêng:
cd src/server
go run ./cmd/api-gateway/      # Terminal 1
go run ./cmd/user-service/     # Terminal 2

# Test endpoint
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

---

## Tóm Tắt Những Điểm Quan Trọng Cho Người Mới

| Khái niệm | Giải thích ngắn |
|-----------|----------------|
| **Gin** | Web framework cho Go — nhẹ, nhanh, middleware-based |
| **gRPC** | Gọi function giữa 2 server qua TCP, nhanh hơn REST |
| **Middleware** | Bộ lọc chạy TRƯỚC handler — dùng cho auth, log, rate limit |
| **Layered Architecture** | Handler → Service → Repository — mỗi layer chỉ làm 1 việc |
| **JWT** | Token để xác thực user — có thời hạn, được sign bằng secret key |
| **pgxpool** | Connection pool cho PostgreSQL — tránh tạo connection mới mỗi request |
| **SELECT FOR UPDATE** | Row-level lock trong PostgreSQL — ngăn race condition |
| **bcrypt** | Thuật toán hash password — có salt tự động, chống rainbow table |
| **Protobuf** | Định nghĩa interface gRPC — sinh code tự động |
| **Docker multi-stage build** | Build stage (tooloan) → Runtime stage (image nặng ~5MB) |
| **Makefile** | Tập hợp lệnh hay dùng — `make dev-up`, `make migrate` |
| **Migration** | SQL version control — có UP (apply) và DOWN (rollback) |