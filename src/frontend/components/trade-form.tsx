'use client';

import { useState, type FormEvent } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { OrderRequest } from '@/lib/types';

interface TradeFormProps {
  symbol: string;
}

type Side = 'BUY' | 'SELL';
type OrderType = 'MARKET' | 'LIMIT';

function isValidPositiveNumber(value: string): boolean {
  if (!value || value.trim() === '') return false;
  const num = parseFloat(value);
  return !isNaN(num) && num > 0 && isFinite(num);
}

export function TradeForm({ symbol }: TradeFormProps) {
  const [side, setSide] = useState<Side>('BUY');
  const [orderType, setOrderType] = useState<OrderType>('LIMIT');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState('');
  const [error, setError] = useState('');

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setSuccess('');

    // Validate numeric inputs
    if (!isValidPositiveNumber(quantity)) {
      setError('Please enter a valid positive quantity');
      return;
    }
    if (orderType === 'LIMIT' && !isValidPositiveNumber(price)) {
      setError('Please enter a valid positive price');
      return;
    }

    setLoading(true);

    try {
      const req: OrderRequest = {
        symbol,
        side,
        type: orderType,
        quantity,
        price: orderType === 'LIMIT' ? price : undefined,
        idempotency_key: crypto.randomUUID(),
      };

      await api.post('/api/v1/orders', req);
      setSuccess(`${side} order placed successfully`);
      setQuantity('');
      setPrice('');
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
        'Failed to place order';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      {/* Side toggle */}
      <div className="grid grid-cols-2 gap-2">
        <button
          type="button"
          onClick={() => setSide('BUY')}
          className={`rounded-lg py-2 text-sm font-semibold transition-colors ${
            side === 'BUY'
              ? 'bg-green-600 text-white'
              : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-600'
          }`}
        >
          BUY
        </button>
        <button
          type="button"
          onClick={() => setSide('SELL')}
          className={`rounded-lg py-2 text-sm font-semibold transition-colors ${
            side === 'SELL'
              ? 'bg-red-600 text-white'
              : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-600'
          }`}
        >
          SELL
        </button>
      </div>

      {/* Order type */}
      <div className="grid grid-cols-2 gap-2">
        <button
          type="button"
          onClick={() => setOrderType('LIMIT')}
          className={`rounded-lg py-1.5 text-sm font-medium transition-colors ${
            orderType === 'LIMIT'
              ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
              : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-600'
          }`}
        >
          Limit
        </button>
        <button
          type="button"
          onClick={() => setOrderType('MARKET')}
          className={`rounded-lg py-1.5 text-sm font-medium transition-colors ${
            orderType === 'MARKET'
              ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
              : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-600'
          }`}
        >
          Market
        </button>
      </div>

      <Input
        label="Quantity"
        type="text"
        inputMode="decimal"
        value={quantity}
        onChange={(e) => setQuantity(e.target.value)}
        placeholder="0.00"
        required
      />

      {orderType === 'LIMIT' && (
        <Input
          label="Price"
          type="text"
          inputMode="decimal"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          placeholder="0.00"
          required
        />
      )}

      {error && (
        <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600 dark:bg-red-900/30 dark:text-red-400">
          {error}
        </p>
      )}

      {success && (
        <p className="rounded-lg bg-green-50 px-3 py-2 text-sm text-green-600 dark:bg-green-900/30 dark:text-green-400">
          {success}
        </p>
      )}

      <Button
        type="submit"
        variant={side === 'BUY' ? 'primary' : 'danger'}
        disabled={loading}
      >
        {loading ? 'Placing...' : `${side} ${symbol}`}
      </Button>
    </form>
  );
}