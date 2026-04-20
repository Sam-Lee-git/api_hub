"use client";

import { useEffect, useState } from "react";
import UserLayout from "@/components/layout/UserLayout";
import api from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function SettingsPage() {
  const [profile, setProfile] = useState({ display_name: "", email: "" });
  const [passwords, setPasswords] = useState({ old: "", new: "", confirm: "" });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.get("/api/user/me").then(({ data }) => {
      setProfile({ display_name: data.display_name, email: data.email });
    });
  }, []);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwords.new && passwords.new !== passwords.confirm) {
      toast.error("新密码两次输入不一致");
      return;
    }
    setSaving(true);
    try {
      await api.put("/api/user/me", {
        display_name: profile.display_name,
        old_password: passwords.old,
        new_password: passwords.new,
      });
      toast.success("设置已保存");
      setPasswords({ old: "", new: "", confirm: "" });
    } catch {
      toast.error("保存失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <UserLayout>
      <div className="space-y-6 max-w-lg">
        <h1 className="text-2xl font-bold text-gray-900">账户设置</h1>

        <form onSubmit={handleSave} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>邮箱</Label>
                <Input value={profile.email} disabled />
              </div>
              <div className="space-y-2">
                <Label htmlFor="displayName">昵称</Label>
                <Input
                  id="displayName"
                  value={profile.display_name}
                  onChange={(e) => setProfile({ ...profile, display_name: e.target.value })}
                />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>修改密码</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="oldPass">当前密码</Label>
                <Input
                  id="oldPass"
                  type="password"
                  value={passwords.old}
                  onChange={(e) => setPasswords({ ...passwords, old: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="newPass">新密码</Label>
                <Input
                  id="newPass"
                  type="password"
                  value={passwords.new}
                  onChange={(e) => setPasswords({ ...passwords, new: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirmPass">确认新密码</Label>
                <Input
                  id="confirmPass"
                  type="password"
                  value={passwords.confirm}
                  onChange={(e) => setPasswords({ ...passwords, confirm: e.target.value })}
                />
              </div>
            </CardContent>
          </Card>

          <Button type="submit" disabled={saving}>
            {saving ? "保存中..." : "保存设置"}
          </Button>
        </form>
      </div>
    </UserLayout>
  );
}
