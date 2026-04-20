"use client";

import { useEffect, useState } from "react";
import AdminLayout from "@/components/layout/AdminLayout";
import api from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import type { User } from "@/types/api";

interface UserWithBalance extends User {
  balance: number;
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<UserWithBalance[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [adjustUser, setAdjustUser] = useState<UserWithBalance | null>(null);
  const [adjustAmount, setAdjustAmount] = useState("");
  const [adjustDesc, setAdjustDesc] = useState("");
  const limit = 20;

  const loadUsers = async () => {
    const { data } = await api.get(`/api/admin/users?limit=${limit}&page=${page}`);
    setUsers(data.data || []);
    setTotal(data.total || 0);
  };

  useEffect(() => { loadUsers(); }, [page]);

  const handleStatusChange = async (user: UserWithBalance) => {
    const newStatus = user.status === "active" ? "suspended" : "active";
    if (!confirm(`确认${newStatus === "suspended" ? "封禁" : "解封"}用户 ${user.email}？`)) return;
    try {
      await api.put(`/api/admin/users/${user.id}/status`, { status: newStatus });
      toast.success("状态已更新");
      loadUsers();
    } catch {
      toast.error("操作失败");
    }
  };

  const handleAdjustCredits = async () => {
    if (!adjustUser) return;
    try {
      await api.post(`/api/admin/users/${adjustUser.id}/credits`, {
        amount: parseInt(adjustAmount),
        description: adjustDesc,
      });
      toast.success("Credits 已调整");
      setAdjustUser(null);
      setAdjustAmount("");
      setAdjustDesc("");
      loadUsers();
    } catch {
      toast.error("调整失败");
    }
  };

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">用户管理</h1>
          <p className="text-sm text-gray-500">共 {total} 个用户</p>
        </div>

        <Card>
          <CardContent className="pt-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-500 border-b">
                    <th className="py-3 pr-4">邮箱</th>
                    <th className="py-3 pr-4">昵称</th>
                    <th className="py-3 pr-4">Credits 余额</th>
                    <th className="py-3 pr-4">角色</th>
                    <th className="py-3 pr-4">状态</th>
                    <th className="py-3 pr-4">注册时间</th>
                    <th className="py-3">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {users.map((user) => (
                    <tr key={user.id}>
                      <td className="py-3 pr-4">{user.email}</td>
                      <td className="py-3 pr-4">{user.display_name || "—"}</td>
                      <td className="py-3 pr-4 font-mono">{user.balance.toLocaleString()}</td>
                      <td className="py-3 pr-4">
                        <Badge variant={user.role === "admin" ? "destructive" : "secondary"}>
                          {user.role}
                        </Badge>
                      </td>
                      <td className="py-3 pr-4">
                        <Badge variant={user.status === "active" ? "default" : "secondary"}>
                          {user.status === "active" ? "正常" : "已封禁"}
                        </Badge>
                      </td>
                      <td className="py-3 pr-4 text-gray-400 text-xs">
                        {new Date(user.created_at).toLocaleDateString("zh-CN")}
                      </td>
                      <td className="py-3">
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => { setAdjustUser(user); setAdjustAmount(""); }}
                          >
                            调整Credits
                          </Button>
                          <Button
                            size="sm"
                            variant={user.status === "active" ? "destructive" : "default"}
                            onClick={() => handleStatusChange(user)}
                          >
                            {user.status === "active" ? "封禁" : "解封"}
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <div className="flex justify-between items-center mt-4 pt-4 border-t">
              <p className="text-sm text-gray-500">共 {total} 条</p>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}>
                  上一页
                </Button>
                <Button variant="outline" size="sm" onClick={() => setPage(p => p + 1)} disabled={page * limit >= total}>
                  下一页
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Dialog open={!!adjustUser} onOpenChange={(open) => { if (!open) setAdjustUser(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>调整 Credits - {adjustUser?.email}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-gray-500">当前余额: {adjustUser?.balance.toLocaleString()}</p>
            <div className="space-y-2">
              <label className="text-sm font-medium">调整数量（正数增加，负数减少）</label>
              <Input
                type="number"
                value={adjustAmount}
                onChange={(e) => setAdjustAmount(e.target.value)}
                placeholder="例如: 10000 或 -5000"
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">备注</label>
              <Input
                value={adjustDesc}
                onChange={(e) => setAdjustDesc(e.target.value)}
                placeholder="调整原因"
              />
            </div>
            <div className="flex gap-2">
              <Button onClick={handleAdjustCredits} className="flex-1">确认调整</Button>
              <Button variant="outline" onClick={() => setAdjustUser(null)} className="flex-1">取消</Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </AdminLayout>
  );
}
