'use client';

import { useState, useEffect } from 'react';

interface OrderBookProps {
  symbol: string;
}

// Static order book data - in production this would come from WebSocket
const MOCK_ASKS = [
  { price: '95100.00', size: '0.5' },
  { price: '95050.00', size: '1.2' },
  { price: '95000.00', size: '0.8' },
  { price: '94950.00', size: '2.0' },
  { price: '94900.00', size: '1.5' },
];

const MOCK_BIDS = [
  { price: '94850.00', size: '1.0' },
  { price: '94800.00', size: '0.6' },
  { price: '94750.00', size: '1.8' },
  { price: '94700.00', size: '0.4' },
  { price: '94650.00', size: '2.2' },
];

export function OrderBook({ symbol }: OrderBookProps) {
  const [asks, setAsks] = useState(MOCK_ASKS);
  const [bids, setBids] = useState(MOCK_BIDS);

  // In production, would subscribe to WebSocket for real-time order book updates
  useEffect(() => {
    // Reset when symbol changes
    setAsks(MOCK_ASKS);
    setBids(MOCK_BIDS);
  }, [symbol]);

  const maxSize = Math.max(
    ...asks.map((a) => parseFloat(a.size)),
    ...bids.map((b) => parseFloat(b.size))
  );

  return (
    <div className="flex flex-col gap-3 text-sm">
      {/* Asks (sells) - reversed order so lowest ask is at bottom */}
      <div className="flex flex-col gap-1">
        {[...asks].reverse().map((ask, i) => {
          const pct = (parseFloat(ask.size) / maxSize) * 100;
          return (
            <div key={`ask-${i}`} className="relative flex items-center gap-2">
              <div
                className="absolute right-0 h-full bg-red-500/10 dark:bg-red-500/20"
                style={{ width: `${pct}%` }}
              />
              <span className="relative w-1/2 font-mono text-red-600 dark:text-red-400">
                {ask.price}
              </span>
              <span className="relative ml-auto font-mono text-zinc-600 dark:text-zinc-400">
                {ask.size}
              </span>
            </div>
          );
        })}
      </div>

      {/* Spread indicator */}
      <div className="flex items-center justify-center border-y border-zinc-200 py-2 dark:border-zinc-700">
        <span className="text-xs text-zinc-500">Spread: 50.00</span>
      </div>

      {/* Bids (buys) */}
      <div className="flex flex-col gap-1">
        {bids.map((bid, i) => {
          const pct = (parseFloat(bid.size) / maxSize) * 100;
          return (
            <div key={`bid-${i}`} className="relative flex items-center gap-2">
              <div
                className="absolute right-0 h-full bg-green-500/10 dark:bg-green-500/20"
                style={{ width: `${pct}%` }}
              />
              <span className="relative w-1/2 font-mono text-green-600 dark:text-green-400">
                {bid.price}
              </span>
              <span className="relative ml-auto font-mono text-zinc-600 dark:text-zinc-400">
                {bid.size}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}