'use client';

import { useState, useEffect } from 'react';
import { marketSocket } from '@/lib/websocket';
import type { TradeEvent } from '@/lib/types';

interface RecentTradesProps {
  symbol: string;
}

export function RecentTrades({ symbol }: RecentTradesProps) {
  const [trades, setTrades] = useState<TradeEvent[]>([]);

  useEffect(() => {
    setTrades([]);

    const unsub = marketSocket.onTrade((trade) => {
      setTrades((prev) => [trade, ...prev].slice(0, 50));
    });

    return () => unsub();
  }, [symbol]);

  function formatTime(ts: number) {
    return new Date(ts).toLocaleTimeString();
  }

  if (trades.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-sm text-zinc-400">
        <p>No trades yet</p>
        <p className="text-xs">Waiting for market data...</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 text-sm">
      <div className="grid grid-cols-3 gap-2 border-b border-zinc-200 pb-2 text-xs font-medium text-zinc-500 dark:border-zinc-700">
        <span>Price</span>
        <span className="text-right">Qty</span>
        <span className="text-right">Time</span>
      </div>

      {trades.map((trade, i) => (
        <div
          key={`${trade.timestamp}-${i}`}
          className="grid grid-cols-3 gap-2 border-b border-zinc-100 py-1.5 dark:border-zinc-800"
        >
          <span className="font-mono text-zinc-900 dark:text-zinc-100">
            {parseFloat(trade.price).toLocaleString(undefined, {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}
          </span>
          <span className="text-right font-mono text-zinc-600 dark:text-zinc-400">
            {parseFloat(trade.quantity).toFixed(6)}
          </span>
          <span className="text-right text-xs text-zinc-400">
            {formatTime(trade.timestamp)}
          </span>
        </div>
      ))}
    </div>
  );
}