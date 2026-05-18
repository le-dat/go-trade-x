import type { TradeEvent } from './types';

type TradeHandler = (trade: TradeEvent) => void;

class MarketSocket {
  private ws: WebSocket | null = null;
  private handlers: Set<TradeHandler> = new Set();
  private symbol: string = '';
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempt: number = 0;
  private readonly maxReconnectDelay = 30000;
  private readonly baseReconnectDelay = 1000;

  connect(symbol: string) {
    // Clear any pending reconnect first
    this.cancelReconnect();

    this.symbol = symbol;
    this.reconnectAttempt = 0;

    // Clean up existing connection
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.close();
      this.ws = null;
    }
    this.handlers.clear();

    const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081';
    const url = `${WS_URL}/ws?symbol=${encodeURIComponent(symbol)}`;

    try {
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        console.log(`[MarketSocket] Connected to ${symbol}`);
      };

      this.ws.onmessage = (event) => {
        try {
          const trade: TradeEvent = JSON.parse(event.data);
          this.handlers.forEach((h) => h(trade));
        } catch {
          console.warn('[MarketSocket] Failed to parse message');
        }
      };

      this.ws.onclose = () => {
        console.log(`[MarketSocket] Disconnected from ${this.symbol}`);
        this.scheduleReconnect();
      };

      this.ws.onerror = () => {
        console.warn(`[MarketSocket] Error on ${this.symbol}`);
      };
    } catch {
      this.scheduleReconnect();
    }
  }

  private cancelReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private scheduleReconnect() {
    this.cancelReconnect();

    // Exponential backoff with jitter: min(baseDelay * 2^attempt, maxReconnectDelay)
    // Add jitter of up to 20% to avoid thundering herd
    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempt),
      this.maxReconnectDelay
    );
    const jitter = delay * 0.2 * Math.random();
    const actualDelay = delay + jitter;

    this.reconnectAttempt++;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.symbol) {
        console.log(`[MarketSocket] Reconnecting to ${this.symbol} (attempt ${this.reconnectAttempt})...`);
        this.connect(this.symbol);
      }
    }, actualDelay);
  }

  disconnect() {
    this.cancelReconnect();
    this.reconnectAttempt = 0;
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.close();
      this.ws = null;
    }
    this.handlers.clear();
  }

  onTrade(handler: TradeHandler): () => void {
    this.handlers.add(handler);
    return () => {
      this.handlers.delete(handler);
    };
  }

  get connected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

// Singleton instance
export const marketSocket = new MarketSocket();