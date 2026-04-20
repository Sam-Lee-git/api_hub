"use client";

import { useEffect, useState } from "react";
import AdminLayout from "@/components/layout/AdminLayout";
import api from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { UsageRecord } from "@/types/api";

export default function AdminUsagePage() {
  const [records, setRecords] = useState<UsageRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ from: "", to: "", model: "" });
  const limit = 20;

  const load = async () => {
    const params = new URLSearchParams({
      limit: String(limit),
      page: String(page),
      ...(filters.from && { from: filters.from }),
      ...(filters.to && { to: filters.to }),
      ...(filters.model && { model: filters.model }),
    });
    const { data } = await api.get(`/api/admin/usage?${params}`);
    setRecords(data.data || []);
    setTotal(data.total || 0);
  };

  useEffect(() => { load(); }, [page]);

  return (
    <AdminLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">全局用量记录</h1>

        <Card>
          <CardContent className="pt-4">
            <div className="flex gap-3 flex-wrap">
              <Input type="date" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} className="w-40" />
              <Input type="date" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} className="w-40" />
              <Input placeholder="模型名称" value={filters.model} onChange={(e) => setFilters({ ...filters, model: e.target.value })} className="w-48" />
              <Button onClick={() => { setPage(1); load(); }} variant="outline">筛选</Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>用量记录（共 {total.toLocaleString()} 条）</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-500 border-b">
                    <th className="pb-2 pr-4">用户ID</th>
                    <th className="pb-2 pr-4">模型</th>
                    <th className="pb-2 pr-4">输入</th>
                    <th className="pb-2 pr-4">输出</th>
                    <th className="pb-2 pr-4">Credits</th>
                    <th className="pb-2 pr-4">状态</th>
                    <th className="pb-2">时间</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {records.map((rec) => (
                    <tr key={rec.id}>
                      <td className="py-2 pr-4">{rec.user_id}</td>
                      <td className="py-2 pr-4 font-mono text-xs">{rec.model_name}</td>
                      <td className="py-2 pr-4">{rec.input_tokens.toLocaleString()}</td>
                      <td className="py-2 pr-4">{rec.output_tokens.toLocaleString()}</td>
                      <td className="py-2 pr-4">{rec.credits_charged}</td>
                      <td className="py-2 pr-4">
                        <span className={`text-xs px-2 py-0.5 rounded-full ${rec.status === "success" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
                          {rec.status}
                        </span>
                      </td>
                      <td className="py-2 text-gray-400 text-xs">{new Date(rec.created_at).toLocaleString("zh-CN")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div className="flex justify-between mt-4 pt-4 border-t">
              <p className="text-sm text-gray-500">共 {total} 条</p>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}>上一页</Button>
                <Button variant="outline" size="sm" onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}>下一页</Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
