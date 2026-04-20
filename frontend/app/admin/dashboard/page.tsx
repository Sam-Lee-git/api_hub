"use client";

import { useEffect, useState } from "react";
import AdminLayout from "@/components/layout/AdminLayout";
import api from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface Stats {
  total_calls: number;
  total_tokens: number;
  total_credits: number;
}

export default function AdminDashboard() {
  const [today, setToday] = useState<Stats | null>(null);
  const [month, setMonth] = useState<Stats | null>(null);

  useEffect(() => {
    api.get("/api/admin/dashboard").then(({ data }) => {
      setToday(data.today);
      setMonth(data.month);
    });
  }, []);

  const statsCards = [
    { label: "今日调用次数", value: today?.total_calls },
    { label: "今日消耗 Credits", value: today?.total_credits },
    { label: "今日 Tokens", value: today?.total_tokens },
    { label: "本月调用次数", value: month?.total_calls },
    { label: "本月消耗 Credits", value: month?.total_credits },
    { label: "本月 Tokens", value: month?.total_tokens },
  ];

  return (
    <AdminLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">数据概览</h1>

        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {statsCards.map((card) => (
            <Card key={card.label}>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-gray-500">{card.label}</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {card.value?.toLocaleString() ?? "—"}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </AdminLayout>
  );
}
