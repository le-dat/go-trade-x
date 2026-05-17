# GoTradeX Frontend Implementation Plan

## Context

The GoTradeX backend is operational with:
- **API Gateway** (port 8080): REST endpoints for auth and orders
- **Market Service** (port 8081): WebSocket endpoint at `/ws?symbol=<pair>` for real-time trades
- **JWT Authentication**: Bearer token auth with 24h expiry
- **Swagger docs**: Available at `/swagger/index.html`

The frontend is a fresh Next.js 16 + React 19 + Tailwind v4 scaffold — no trading UI exists yet.

---

## Phase 1: Foundation

### 1.1 Install Dependencies

```bash
pnpm add zustand axios
```

**Why Zustand**: Minimal boilerplate, TypeScript-friendly, no provider wrapping hell. Perfect for a trading dashboard where you need reactive state across many components (order book, positions, auth).

**Why Axios**: Better defaults than fetch, built-in interceptor support for auth headers.

### 1.2 Project Structure

```
frontend/
├── app/
│   ├── layout.tsx              # Root layout (keep)
│   ├── page.tsx                # Landing/login redirect
│   ├── (auth)/
│   │   ├── login/page.tsx
│   │   └── register/page.tsx
│   ├── (dashboard)/
│   │   ├── layout.tsx          # Auth wrapper
│   │   ├── page.tsx            # Main trading view
│   │   ├── orders/page.tsx     # Order history
│   │   └── positions/page.tsx  # Open positions
│   └── api/                    # Optional: API routes if SSR needed
├── components/
│   ├── ui/                     # Base UI (Button, Input, Card)
│   ├── order-book.tsx
│   ├── trade-form.tsx
│   ├── recent-trades.tsx
│   └── header.tsx
├── lib/
│   ├── api.ts                  # Axios instance + interceptors
│   ├── auth-store.ts           # Zustand store for auth
│   ├── websocket.ts            # Market WebSocket client
│   └── types.ts                # Shared TypeScript types
└── .env.local
```

### 1.3 API Client (`lib/api.ts`)

```typescript
import axios from 'axios'

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
})

// Request interceptor: attach JWT
api.interceptors.request.use((config) => {
  const token = authStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401 → logout
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      authStore.getState().logout()
    }
    return Promise.reject(err)
  }
)
```

### 1.4 Auth Store (`lib/auth-store.ts`)

```typescript
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  userId: string | null
  email: string | null
  login: (token: string, userId: string, email: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      userId: null,
      email: null,
      login: (token, userId, email) => set({ token, userId, email }),
      logout: () => set({ token: null, userId: null, email: null }),
    }),
    { name: 'auth' }
  )
)
```

---

## Phase 2: Authentication Pages

### `/login` - `app/(auth)/login/page.tsx`
- Email + password form
- On success: store JWT in Zustand (persisted), redirect to `/dashboard`
- On error: display message from API response

### `/register` - `app/(auth)/register/page.tsx`
- Email + password + confirm password form
- Validate password match client-side
- On success: redirect to `/login`

---

## Phase 3: Trading Dashboard

### Main View (`app/(dashboard)/page.tsx`)

**Layout**: Header + 3-column grid
```
[Header: Logo | Symbol Selector | Balance | Logout]
[Bid/Ask Ladder] [Trade Form] [Recent Trades]
[Order History Table]
```

### Order Book Component (`components/order-book.tsx`)
- Fetches initial bids/asks via REST (or WebSocket subscription)
- Displays price levels with size bars
- Highlights price on hover

### Trade Form (`components/trade-form.tsx`)
- Symbol input (dropdown with BTC/USD, ETH/USD, etc.)
- Side: BUY / SELL toggle
- Order type: MARKET / LIMIT
- Quantity input
- Price input (disabled for MARKET orders)
- "Place Order" button

On submit:
```typescript
const response = await api.post('/api/v1/orders', {
  symbol: 'BTC/USD',
  side: 'BUY',
  type: 'LIMIT',
  quantity: '0.01',
  price: '95000.00',
  idempotency_key: crypto.randomUUID()
})
// Store order in local state, show confirmation
```

### Recent Trades (`components/recent-trades.tsx`)
- Connects to WebSocket: `ws://localhost:8081/ws?symbol=BTC/USD`
- Receives `TradeEvent` messages:
  ```typescript
  interface TradeEvent {
    symbol: string
    price: string
    quantity: string
    timestamp: string // Unix epoch ms
  }
  ```
- Displays last N trades in scrolling list
- Auto-reconnects on disconnect

### WebSocket Client (`lib/websocket.ts`)

```typescript
type TradeHandler = (trade: TradeEvent) => void

class MarketSocket {
  private ws: WebSocket | null = null
  private handlers: Set<TradeHandler> = new Set()

  connect(symbol: string) {
    const url = `ws://localhost:8081/ws?symbol=${symbol}`
    this.ws = new WebSocket(url)

    this.ws.onmessage = (event) => {
      const trade = JSON.parse(event.data)
      this.handlers.forEach(h => h(trade))
    }

    this.ws.onclose = () => {
      // Reconnect after 1s
      setTimeout(() => this.connect(symbol), 1000)
    }
  }

  onTrade(handler: TradeHandler) {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler) // return unsubscribe fn
  }
}
```

---

## Phase 4: Order History

### `/orders` - `app/(dashboard)/orders/page.tsx`

- Fetches user's orders via `GET /api/v1/orders` (need to check if this endpoint exists or if we need `GET /api/v1/orders/:id` for each)
- Displays table: Symbol | Side | Type | Price | Qty | Filled | Status | Time
- Status badges: OPEN (yellow), FILLED (green), CANCELLED (red)

---

## Phase 5: Polish

- Connection status indicator (WebSocket connected/disconnected)
- Loading states on all async operations
- Error toasts for failed orders
- Responsive layout (mobile: stack columns vertically)
- Dark mode support (Tailwind `dark:` classes)

---

## Backend Contract Reference

### Auth
```
POST /api/v1/auth/register
Body: { "email": "user@example.com", "password": "secret123" }
Response: { "user_id": "uuid", "email": "user@example.com" }

POST /api/v1/auth/login
Body: { "email": "user@example.com", "password": "secret123" }
Response: { "token": "eyJ...", "user_id": "uuid", "email": "user@example.com" }
```

### Orders
```
POST /api/v1/orders
Headers: Authorization: Bearer <token>
Body: {
  "symbol": "BTC/USD",
  "side": "BUY",
  "type": "LIMIT",
  "quantity": "0.01",
  "price": "95000.00",
  "idempotency_key": "uuid"
}
Response: { "order_id": "uuid", ... }

GET /api/v1/orders/:id
Headers: Authorization: Bearer <token>
Response: { "order_id": "...", "status": "OPEN", ... }
```

### WebSocket
```
ws://localhost:8081/ws?symbol=BTC/USD
Messages: { "symbol": "BTC/USD", "price": "95000.00", "quantity": "0.01", "timestamp": 1716000000000 }
```

---

## Environment Variables

```env
# frontend/.env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8081
```

---

## Implementation Order

1. **Types + API client** (`lib/types.ts`, `lib/api.ts`)
2. **Auth store** (`lib/auth-store.ts`)
3. **Login + Register pages**
4. **Dashboard layout** with header
5. **Trade form** (POST order)
6. **WebSocket client** + Recent trades component
7. **Order history page**

Total estimated: ~15-20 files to create/modify.