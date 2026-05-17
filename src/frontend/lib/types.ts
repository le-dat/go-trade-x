// API types for GoTradeX backend

export interface AuthResponse {
  token: string;
  user_id: string;
  email: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface OrderRequest {
  symbol: string;
  side: 'BUY' | 'SELL';
  type: 'MARKET' | 'LIMIT';
  quantity: string;
  price?: string;
  idempotency_key: string;
}

export interface Order {
  order_id: string;
  user_id: string;
  symbol: string;
  side: 'BUY' | 'SELL';
  type: 'MARKET' | 'LIMIT';
  price: string;
  quantity: string;
  filled_qty: string;
  status: 'OPEN' | 'FILLED' | 'CANCELLED';
  created_at: string;
  updated_at: string;
}

export interface TradeEvent {
  symbol: string;
  price: string;
  quantity: string;
  timestamp: number;
}

export interface ApiError {
  message: string;
}