"use client";

import { useEffect, useState } from "react";
import AdminLayout from "@/components/layout/AdminLayout";
import api from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { PaymentOrder } from "@/types/api";

const statusLabels: Record<string, { label: string; variant: "default" | "secondary" | "destructive" }> = {
  pending: { label: "待支付", variant: "secondary" },
  paid: { label: "已支付", variant: "default" },
  failed: { label: "失败", variant: "destructive" },
  refunded: { label: "已退款", variant: "secondary" },
};

export default function AdminPaymentsPage() {
  const [orders, setOrders] = useState<PaymentOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const limit = 20;

  useEffect(() => {
    api.get(`/api/admin/payments?limit=${limit}&page=${page}`).then(({ data }) => {
      setOrders(data.data || []);
      setTotal(data.total || 0);
    });
  }, [page]);

  return (
    <AdminLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">支付订单</h1>

        <Card>
          <CardHeader>
            <CardTitle>订单列表（共 {total.toLocaleString()} 条）</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-500 border-b">
                    <th className="pb-2 pr-4">订单号</th>
                    <th className="pb-2 pr-4">用户ID</th>
                    <th className="pb-2 pr-4">渠道</th>
                    <th className="pb-2 pr-4">金额(元)</th>
                    <th className="pb-2 pr-4">Credits</th>
                    <th className="pb-2 pr-4">状态</th>
                    <th className="pb-2">创建时间</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {orders.map((order) => {
                    const s = statusLabels[order.status] || { label: order.status, variant: "secondary" as const };
                    return (
                      <tr key={order.id}>
                        <td className="py-2 pr-4 font-mono text-xs">{order.order_no}</td>
                        <td className="py-2 pr-4">{order.user_id}</td>
                        <td className="py-2 pr-4">{order.channel === "alipay" ? "支付宝" : "微信支付"}</td>
                        <td className="py-2 pr-4">¥{(order.amount_cny / 100).toFixed(2)}</td>
                        <td className="py-2 pr-4">{(order.credits_to_add ?? order.credits ?? 0).toLocaleString()}</td>
                        <td className="py-2 pr-4">
                          <Badge variant={s.variant}>{s.label}</Badge>
                        </td>
                        <td className="py-2 text-gray-400 text-xs">
                          {new Date(order.created_at).toLocaleString("zh-CN")}
                        </td>
                      </tr>
                    );
                  })}
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
