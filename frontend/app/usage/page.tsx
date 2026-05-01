"use client";

import { useCallback, useEffect, useState } from "react";
import UserLayout from "@/components/layout/UserLayout";
import api from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import type { UsageRecord } from "@/types/api";

export default function UsagePage() {
  const [records, setRecords] = useState<UsageRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ from: "", to: "", model: "" });
  const limit = 20;

  const loadUsage = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        limit: String(limit),
        page: String(page),
        ...(filters.from && { from: filters.from }),
        ...(filters.to && { to: filters.to }),
        ...(filters.model && { model: filters.model }),
      });
      const { data } = await api.get(`/api/usage?${params}`);
      setRecords(data.data || []);
      setTotal(data.total || 0);
    } catch {
      // ignore
    }
  }, [filters.from, filters.model, filters.to, page]);

  useEffect(() => {
    let cancelled = false;
    const params = new URLSearchParams({
      limit: String(limit),
      page: String(page),
      ...(filters.from && { from: filters.from }),
      ...(filters.to && { to: filters.to }),
      ...(filters.model && { model: filters.model }),
    });
    api.get(`/api/usage?${params}`)
      .then(({ data }) => {
        if (cancelled) return;
        setRecords(data.data || []);
        setTotal(data.total || 0);
      })
      .catch(() => {
        // ignore
      });
    return () => {
      cancelled = true;
    };
  }, [filters.from, filters.model, filters.to, page]);

  return (
    <UserLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">用量记录</h1>

        {/* Filters */}
        <Card>
          <CardContent className="pt-4">
            <div className="flex gap-3 flex-wrap">
              <Input
                type="date"
                placeholder="开始日期"
                value={filters.from}
                onChange={(e) => setFilters({ ...filters, from: e.target.value })}
                className="w-40"
              />
              <Input
                type="date"
                placeholder="结束日期"
                value={filters.to}
                onChange={(e) => setFilters({ ...filters, to: e.target.value })}
                className="w-40"
              />
              <Input
                placeholder="模型名称"
                value={filters.model}
                onChange={(e) => setFilters({ ...filters, model: e.target.value })}
                className="w-48"
              />
              <Button onClick={() => { setPage(1); loadUsage(); }} variant="outline">
                筛选
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* Usage table */}
        <Card>
          <CardHeader>
            <CardTitle>调用记录（共 {total.toLocaleString()} 条）</CardTitle>
          </CardHeader>
          <CardContent>
            {records.length === 0 ? (
              <p className="text-gray-400 text-sm py-4 text-center">暂无记录</p>
            ) : (
              <>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-left text-gray-500 border-b">
                        <th className="pb-2 pr-4">模型</th>
                        <th className="pb-2 pr-4">输入</th>
                        <th className="pb-2 pr-4">输出</th>
                        <th className="pb-2 pr-4">Credits</th>
                        <th className="pb-2 pr-4">延迟</th>
                        <th className="pb-2 pr-4">状态</th>
                        <th className="pb-2">时间</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {records.map((rec) => (
                        <tr key={rec.id}>
                          <td className="py-2 pr-4 font-mono text-xs">{rec.model_name}</td>
                          <td className="py-2 pr-4">{rec.input_tokens.toLocaleString()}</td>
                          <td className="py-2 pr-4">{rec.output_tokens.toLocaleString()}</td>
                          <td className="py-2 pr-4">{rec.credits_charged}</td>
                          <td className="py-2 pr-4">{rec.latency_ms}ms</td>
                          <td className="py-2 pr-4">
                            <span className={`text-xs px-2 py-0.5 rounded-full ${
                              rec.status === "success" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"
                            }`}>
                              {rec.status === "success" ? "成功" : "失败"}
                            </span>
                          </td>
                          <td className="py-2 text-gray-400 text-xs whitespace-nowrap">
                            {new Date(rec.created_at).toLocaleString("zh-CN")}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                <div className="flex items-center justify-between mt-4">
                  <p className="text-sm text-gray-500">
                    第 {(page - 1) * limit + 1}–{Math.min(page * limit, total)} 条，共 {total} 条
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage(p => Math.max(1, p - 1))}
                      disabled={page <= 1}
                    >
                      上一页
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage(p => p + 1)}
                      disabled={page * limit >= total}
                    >
                      下一页
                    </Button>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </UserLayout>
  );
}
