'use client';

import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/auth-store';

export function Header() {
  const router = useRouter();
  const { email, logout } = useAuthStore();

  function handleLogout() {
    logout();
    router.push('/login');
  }

  return (
    <header className="flex h-14 items-center justify-between border-b border-zinc-200 bg-white px-6 dark:border-zinc-800 dark:bg-zinc-900">
      <div className="flex items-center gap-4">
        <a href="/dashboard" className="text-xl font-bold text-zinc-900 dark:text-zinc-100">
          GoTradeX
        </a>
        <span className="text-sm text-zinc-500">|</span>
        <span className="text-sm text-zinc-600 dark:text-zinc-400">Trading</span>
      </div>

      <div className="flex items-center gap-4">
        <span className="text-sm text-zinc-600 dark:text-zinc-400">{email}</span>
        <button
          onClick={handleLogout}
          className="rounded-lg px-3 py-1.5 text-sm font-medium text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
        >
          Logout
        </button>
      </div>
    </header>
  );
}