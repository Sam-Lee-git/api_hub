"use client";

import { useEffect, useState } from "react";
import UserLayout from "@/components/layout/UserLayout";
import api from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from "recharts";
import type { UsageRecord } from "@/types/api";

interface Balance {
  balance: number;
  total_spent: number;
  total_topped: number;
}

export default function DashboardPage() {
  const [balance, setBalance] = useState<Balance | null>(null);
  const [recentUsage, setRecentUsage] = useState<UsageRecord[]>([]);
  const [summary, setSummary] = useState<{ total_calls: number; total_tokens: number; total_credits: number } | null>(null);

  useEffect(() => {
    const load = async () => {
      const [balRes, usageRes, summaryRes] = await Promise.allSettled([
        api.get("/api/billing/balance"),
        api.get("/api/usage?limit=5"),
        api.get("/api/usage/summary"),
      ]);

      if (balRes.status === "fulfilled") setBalance(balRes.value.data);
      if (usageRes.status === "fulfilled") setRecentUsage(usageRes.value.data.data || []);
      if (summaryRes.status === "fulfilled") setSummary(summaryRes.value.data);
    };
    load();
  }, []);

  return (
    <UserLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">控制台</h1>

        {/* Stats cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">Credits 余额</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{balance?.balance.toLocaleString() ?? "—"}</div>
              {balance && balance.balance <= 0 && (
                <p className="text-xs text-red-500 mt-1">余额不足，请充值</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">近7天调用次数</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{summary?.total_calls?.toLocaleString() ?? "—"}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-gray-500">近7天消耗 Credits</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{summary?.total_credits?.toLocaleString() ?? "—"}</div>
            </CardContent>
          </Card>
        </div>

        {/* Quick actions */}
        {balance && balance.balance <= 1000 && (
          <Card className="border-yellow-200 bg-yellow-50">
            <CardContent className="pt-4">
              <p className="text-sm text-yellow-800">
                ⚠️ Credits 余额较低，
                <a href="/billing" className="font-medium underline">点击充值</a>
                避免 API 调用中断
              </p>
            </CardContent>
          </Card>
        )}

        {/* Recent usage table */}
        <Card>
          <CardHeader>
            <CardTitle>最近调用记录</CardTitle>
          </CardHeader>
          <CardContent>
            {recentUsage.length === 0 ? (
              <p className="text-gray-400 text-sm py-4 text-center">暂无记录</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-gray-500 border-b">
                      <th className="pb-2 pr-4">模型</th>
                      <th className="pb-2 pr-4">输入 Tokens</th>
                      <th className="pb-2 pr-4">输出 Tokens</th>
                      <th className="pb-2 pr-4">Credits</th>
                      <th className="pb-2">时间</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {recentUsage.map((rec) => (
                      <tr key={rec.id} className="py-2">
                        <td className="py-2 pr-4 font-mono text-xs">{rec.model_name}</td>
                        <td className="py-2 pr-4">{rec.input_tokens.toLocaleString()}</td>
                        <td className="py-2 pr-4">{rec.output_tokens.toLocaleString()}</td>
                        <td className="py-2 pr-4">{rec.credits_charged}</td>
                        <td className="py-2 text-gray-400 text-xs">
                          {new Date(rec.created_at).toLocaleString("zh-CN")}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Placeholder chart */}
        <Card>
          <CardHeader>
            <CardTitle>Credits 消耗趋势（近7天）</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={[]}>
                <XAxis dataKey="date" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="credits" fill="#3b82f6" />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>
    </UserLayout>
  );
}
