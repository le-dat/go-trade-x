# Step 03 — User Service: Register/Login/Balance (gRPC)

## Goal

Implement the gRPC User Service with user registration, login, and balance management.

> **Prerequisite**: [Step 02 — API Gateway](../02-api-gateway/02-api-gateway.md)

---

## Proto Contract

```protobuf
service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
  rpc DeductBalance(DeductBalanceRequest) returns (DeductBalanceResponse);
  rpc CreditBalance(CreditBalanceRequest) returns (CreditBalanceResponse);
}
```

---

## Database Schema

`migrations/001_create_users.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
  user_id UUID REFERENCES users(id),
  asset TEXT NOT NULL,
  available NUMERIC(20,8) NOT NULL DEFAULT 0,
  locked NUMERIC(20,8) NOT NULL DEFAULT 0,
  PRIMARY KEY (user_id, asset)
);
```

---

## Step 3.1 — Create Proto

`proto/user.proto`:

```protobuf
syntax = "proto3";
package user;
option go_package = "github.com/verno/gotradex/pkg/proto/user";

service UserService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc GetBalance(GetBalanceRequest) returns (GetBalanceResponse);
  rpc DeductBalance(DeductBalanceRequest) returns (DeductBalanceResponse);
  rpc CreditBalance(CreditBalanceRequest) returns (CreditBalanceResponse);
}
```

Generate code:
```bash
make proto
```

---

## Step 3.2 — Repository

`internal/user/repository.go`:

```go
func (r *userRepository) DeductBalance(ctx context.Context, tx pgx.Tx, userID, asset string, amount decimal.Decimal) (decimal.Decimal, error) {
    row := tx.QueryRow(ctx,
        `SELECT available FROM balances WHERE user_id = $1 AND asset = $2 FOR UPDATE`,
        userID, asset)

    var available decimal.Decimal
    if err := row.Scan(&available); err != nil {
        return decimal.Zero, err
    }

    newBalance := available.Sub(amount)
    if newBalance.IsNegative() {
        return decimal.Zero, ErrInsufficientBalance
    }

    _, err := tx.Exec(ctx,
        `UPDATE balances SET available = $1 WHERE user_id = $2 AND asset = $3`,
        newBalance, userID, asset)
    return newBalance, err
}
```

Key: `SELECT ... FOR UPDATE` inside a transaction prevents race conditions.

---

## Step 3.3 — Service

`internal/user/service.go`:

```go
func (s *userService) Login(ctx context.Context, email, password string) (string, error) {
    user, err := s.repo.GetByEmail(ctx, email)
    if err != nil {
        return "", fmt.Errorf("user not found: %w", err)
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return "", fmt.Errorf("invalid password: %w", err)
    }

    token, err := IssueToken(user.ID, s.jwtSecret, s.jwtExpiry)
    return token, err
}
```

---

## Step 3.4 — Server

`cmd/user-service/main.go`:

```go
func main() {
    cfg := config.Load()
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    db, err := database.NewPostgres(cfg.DatabaseURL)
    if err != nil {
        logger.Fatal("failed to connect to database", zap.Error(err))
    }
    defer db.Close()

    repo := userrepo.New(db)
    svc := usersvc.New(repo, cfg.JWTSecret, cfg.JWTExpiry)
    handler := usergrpc.New(svc)

    lis, _ := net.Listen("tcp", ":50051")
    grpcServer := grpc.NewServer()
    userpb.RegisterUserServiceServer(grpcServer, handler)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        grpcServer.GracefulStop()
    }()

    logger.Info("user-service listening on :50051")
    grpcServer.Serve(lis)
}
```

---

## Step 3.5 — Verify

```bash
make run-user

grpcurl -plaintext -d '{"email":"test@test.com","password":"secret"}' \
  localhost:50051 user.UserService/Register
# → {"userId":"...","email":"test@test.com"}

grpcurl -plaintext -d '{"email":"test@test.com","password":"secret"}' \
  localhost:50051 user.UserService/Login
# → {"token":"eyJhbGciOiJIUzI1NiIs...","userId":"..."}
```

---

## Verification Checklist

- [ ] `grpcurl localhost:50051 list` shows `user.UserService`
- [ ] Register returns user ID
- [ ] Login returns valid JWT token (24h expiry)
- [ ] DeductBalance uses `SELECT FOR UPDATE`

> ➡️ Next: [Step 04 — Kafka Integration](../03-services/04-kafka.md)