"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import api from "@/lib/api";
import { setAccessToken, setUser } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";

export default function RegisterPage() {
  const router = useRouter();
  const [form, setForm] = useState({ email: "", password: "", confirmPassword: "", displayName: "" });
  const [loading, setLoading] = useState(false);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (form.password !== form.confirmPassword) {
      toast.error("两次输入的密码不一致");
      return;
    }
    setLoading(true);
    try {
      await api.post("/api/auth/register", {
        email: form.email,
        password: form.password,
        display_name: form.displayName,
      });
      toast.success("注册成功！请登录");

      // Auto-login after registration
      const { data } = await api.post("/api/auth/login", {
        email: form.email,
        password: form.password,
      });
      setAccessToken(data.access_token);
      setUser(data.user);

      // Redirect to billing with a welcome message
      toast.info("请先充值 Credits 才能使用 API");
      router.replace("/billing");
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      toast.error(error.response?.data?.error || "注册失败，请重试");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">创建账户</CardTitle>
          <CardDescription>注册后需要充值才能使用 API</CardDescription>
        </CardHeader>
        <form onSubmit={handleRegister}>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="displayName">昵称</Label>
              <Input
                id="displayName"
                placeholder="您的昵称"
                value={form.displayName}
                onChange={(e) => setForm({ ...form, displayName: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">邮箱</Label>
              <Input
                id="email"
                type="email"
                placeholder="your@email.com"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码（至少8位）</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required
                minLength={8}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">确认密码</Label>
              <Input
                id="confirmPassword"
                type="password"
                placeholder="••••••••"
                value={form.confirmPassword}
                onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
                required
              />
            </div>
          </CardContent>
          <CardFooter className="flex flex-col gap-3">
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "注册中..." : "注册"}
            </Button>
            <p className="text-sm text-gray-500 text-center">
              已有账户？{" "}
              <Link href="/login" className="text-blue-600 hover:underline">
                立即登录
              </Link>
            </p>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
