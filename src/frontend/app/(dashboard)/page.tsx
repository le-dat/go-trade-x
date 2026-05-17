'use client';

import { useState, useEffect, useCallback } from 'react';
import { marketSocket } from '@/lib/websocket';
import type { TradeEvent } from '@/lib/types';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { TradeForm } from '@/components/trade-form';
import { RecentTrades } from '@/components/recent-trades';
import { OrderBook } from '@/components/order-book';

const SYMBOLS = ['BTC/USD', 'ETH/USD', 'SOL/USD', 'DOGE/USD'] as const;
type Symbol = (typeof SYMBOLS)[number];

export default function DashboardPage() {
  const [symbol, setSymbol] = useState<Symbol>('BTC/USD');
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    marketSocket.connect(symbol);

    const unsub = marketSocket.onTrade(() => {
      setConnected(true);
    });

    return () => {
      unsub();
      marketSocket.disconnect();
    };
  }, [symbol]);

  return (
    <div className="flex flex-col gap-4">
      {/* Symbol selector + connection status */}
      <div className="flex items-center justify-between">
        <div className="flex gap-2">
          {SYMBOLS.map((s) => (
            <button
              key={s}
              onClick={() => setSymbol(s)}
              className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors ${
                s === symbol
                  ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
                  : 'bg-white text-zinc-600 hover:bg-zinc-100 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700'
              }`}
            >
              {s}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <div
            className={`h-2 w-2 rounded-full ${connected ? 'bg-green-500' : 'bg-yellow-500 animate-pulse'}`}
          />
          <span className="text-sm text-zinc-500">
            {connected ? 'Live' : 'Connecting...'}
          </span>
        </div>
      </div>

      {/* Main 3-column grid */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* Order Book */}
        <Card>
          <CardHeader>
            <CardTitle>Order Book</CardTitle>
          </CardHeader>
          <OrderBook symbol={symbol} />
        </Card>

        {/* Trade Form */}
        <Card>
          <CardHeader>
            <CardTitle>Place Order</CardTitle>
          </CardHeader>
          <TradeForm symbol={symbol} />
        </Card>

        {/* Recent Trades */}
        <Card>
          <CardHeader>
            <CardTitle>Recent Trades</CardTitle>
          </CardHeader>
          <RecentTrades symbol={symbol} />
        </Card>
      </div>
    </div>
  );
}