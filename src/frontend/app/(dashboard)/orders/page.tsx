'use client';

import { useState, useEffect } from 'react';
import { api } from '@/lib/api';
import type { Order } from '@/lib/types';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';

const STATUS_STYLES = {
  OPEN: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  FILLED: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  CANCELLED: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleString();
}

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    async function fetchOrders() {
      try {
        // In production this would be GET /api/v1/orders with pagination
        // For now we show a placeholder since backend may not have this endpoint yet
        setOrders([]);
      } catch {
        setError('Failed to load orders');
      } finally {
        setLoading(false);
      }
    }
    fetchOrders();
  }, []);

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold text-zinc-900 dark:text-zinc-100">Order History</h1>

      <Card>
        <CardHeader>
          <CardTitle>Your Orders</CardTitle>
        </CardHeader>

        {loading && (
          <div className="flex items-center justify-center py-8">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-900" />
          </div>
        )}

        {!loading && error && (
          <p className="text-red-600 dark:text-red-400">{error}</p>
        )}

        {!loading && orders.length === 0 && !error && (
          <div className="flex flex-col items-center justify-center py-8 text-sm text-zinc-400">
            <p>No orders yet</p>
            <p className="text-xs">Place your first order from the dashboard</p>
          </div>
        )}

        {!loading && orders.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-zinc-200 text-left dark:border-zinc-700">
                  <th className="pb-2 font-medium text-zinc-500">Symbol</th>
                  <th className="pb-2 font-medium text-zinc-500">Side</th>
                  <th className="pb-2 font-medium text-zinc-500">Type</th>
                  <th className="pb-2 font-medium text-zinc-500">Price</th>
                  <th className="pb-2 font-medium text-zinc-500">Qty</th>
                  <th className="pb-2 font-medium text-zinc-500">Filled</th>
                  <th className="pb-2 font-medium text-zinc-500">Status</th>
                  <th className="pb-2 font-medium text-zinc-500">Time</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => (
                  <tr
                    key={order.order_id}
                    className="border-b border-zinc-100 dark:border-zinc-800"
                  >
                    <td className="py-2 font-medium">{order.symbol}</td>
                    <td className={`py-2 ${order.side === 'BUY' ? 'text-green-600' : 'text-red-600'}`}>
                      {order.side}
                    </td>
                    <td className="py-2">{order.type}</td>
                    <td className="py-2 font-mono">{parseFloat(order.price).toLocaleString()}</td>
                    <td className="py-2 font-mono">{parseFloat(order.quantity).toFixed(6)}</td>
                    <td className="py-2 font-mono">{parseFloat(order.filled_qty).toFixed(6)}</td>
                    <td className="py-2">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          STATUS_STYLES[order.status] || ''
                        }`}
                      >
                        {order.status}
                      </span>
                    </td>
                    <td className="py-2 text-xs text-zinc-400">{formatDate(order.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}