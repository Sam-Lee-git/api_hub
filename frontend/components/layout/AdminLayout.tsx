"use client";

import { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { isLoggedIn, getUser, clearAuth } from "@/lib/auth";
import api from "@/lib/api";

const adminNav = [
  { href: "/admin/dashboard", label: "数据概览" },
  { href: "/admin/users", label: "用户管理" },
  { href: "/admin/models", label: "模型定价" },
  { href: "/admin/usage", label: "用量记录" },
  { href: "/admin/payments", label: "支付订单" },
];

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const user = getUser();

  useEffect(() => {
    if (!isLoggedIn()) {
      router.replace("/login");
      return;
    }
    if (user?.role !== "admin") {
      router.replace("/dashboard");
    }
  }, [router, user]);

  const handleLogout = async () => {
    try { await api.post("/api/auth/logout"); } finally {
      clearAuth();
      router.replace("/login");
    }
  };

  return (
    <div className="min-h-screen flex">
      <aside className="w-56 bg-gray-900 text-white flex flex-col">
        <div className="p-4 border-b border-gray-700">
          <h1 className="font-bold text-lg">管理后台</h1>
          <p className="text-xs text-gray-400 truncate">{user?.email}</p>
        </div>
        <nav className="flex-1 p-4 space-y-1">
          {adminNav.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={`block px-3 py-2 rounded-md text-sm transition-colors ${
                pathname === item.href
                  ? "bg-gray-700 text-white"
                  : "text-gray-400 hover:bg-gray-700 hover:text-white"
              }`}
            >
              {item.label}
            </Link>
          ))}
          <Link href="/dashboard" className="block px-3 py-2 rounded-md text-sm text-blue-400 hover:bg-gray-700">
            用户视图
          </Link>
        </nav>
        <div className="p-4 border-t border-gray-700">
          <button onClick={handleLogout} className="text-sm text-gray-400 hover:text-white">
            退出登录
          </button>
        </div>
      </aside>
      <main className="flex-1 bg-gray-50 overflow-auto">
        <div className="max-w-7xl mx-auto p-8">{children}</div>
      </main>
    </div>
  );
}
