"use client";

import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { isLoggedIn, getUser, clearAuth } from "@/lib/auth";
import api from "@/lib/api";
import { Badge } from "@/components/ui/badge";

const navItems = [
  { href: "/dashboard", label: "控制台" },
  { href: "/keys", label: "API Keys" },
  { href: "/usage", label: "用量记录" },
  { href: "/billing", label: "充值" },
  { href: "/settings", label: "设置" },
];

export default function UserLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [balance, setBalance] = useState<number | null>(null);
  const user = getUser();

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    fetchBalance();
    const interval = setInterval(fetchBalance, 30000);
    return () => clearInterval(interval);
  }, [router]);

  const fetchBalance = async () => {
    try {
      const { data } = await api.get("/api/billing/balance");
      setBalance(data.balance);
    } catch {
      // ignore
    }
  };

  const handleLogout = async () => {
    try {
      await api.post("/api/auth/logout");
    } finally {
      clearAuth();
      router.replace("/login");
    }
  };

  return (
    <div className="min-h-screen flex">
      {/* Sidebar */}
      <aside className="w-56 bg-white border-r border-gray-200 flex flex-col">
        <div className="p-4 border-b">
          <h1 className="font-bold text-lg text-gray-900">AI API Platform</h1>
          <p className="text-xs text-gray-500 truncate">{user?.email}</p>
        </div>

        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`block px-3 py-2 rounded-md text-sm transition-colors ${
                pathname === item.href
                  ? "bg-blue-50 text-blue-700 font-medium"
                  : "text-gray-600 hover:bg-gray-100"
              }`}
            >
              {item.label}
            </Link>
          ))}
          {user?.role === "admin" && (
            <Link
              href="/admin/dashboard"
              className="block px-3 py-2 rounded-md text-sm text-purple-600 hover:bg-purple-50"
            >
              管理后台
            </Link>
          )}
        </nav>

        {/* Balance display */}
        <div className="p-4 border-t">
          {balance !== null && (
            <div className="mb-3">
              <p className="text-xs text-gray-500">Credits 余额</p>
              <div className="flex items-center gap-2">
                <span className="font-semibold text-gray-900">{balance.toLocaleString()}</span>
                {balance <= 0 && (
                  <Badge variant="destructive" className="text-xs">
                    需充值
                  </Badge>
                )}
              </div>
            </div>
          )}
          <button
            onClick={handleLogout}
            className="text-sm text-gray-500 hover:text-gray-800"
          >
            退出登录
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="max-w-6xl mx-auto p-8">{children}</div>
      </main>
    </div>
  );
}
