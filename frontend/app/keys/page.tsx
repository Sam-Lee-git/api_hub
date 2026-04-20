"use client";

import { useEffect, useState } from "react";
import UserLayout from "@/components/layout/UserLayout";
import api from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { APIKey } from "@/types/api";

export default function KeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [creating, setCreating] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [newKey, setNewKey] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);

  const loadKeys = async () => {
    try {
      const { data } = await api.get("/api/keys");
      setKeys(data.data || []);
    } catch {
      toast.error("获取 API Keys 失败");
    }
  };

  useEffect(() => {
    loadKeys();
  }, []);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const { data } = await api.post("/api/keys", { name: keyName || "Default Key" });
      setNewKey(data.key);
      setShowDialog(true);
      setKeyName("");
      loadKeys();
    } catch {
      toast.error("创建 API Key 失败");
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (id: number) => {
    if (!confirm("确认注销此 API Key？注销后无法恢复")) return;
    try {
      await api.delete(`/api/keys/${id}`);
      toast.success("API Key 已注销");
      loadKeys();
    } catch {
      toast.error("注销失败");
    }
  };

  const copyKey = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey);
      toast.success("已复制到剪贴板");
    }
  };

  return (
    <UserLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
          <div className="flex gap-2">
            <Input
              placeholder="Key 名称（可选）"
              value={keyName}
              onChange={(e) => setKeyName(e.target.value)}
              className="w-48"
            />
            <Button onClick={handleCreate} disabled={creating}>
              {creating ? "创建中..." : "创建新 Key"}
            </Button>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>您的 API Keys</CardTitle>
          </CardHeader>
          <CardContent>
            {keys.length === 0 ? (
              <p className="text-gray-400 text-sm py-4 text-center">暂无 API Key，点击右上角创建</p>
            ) : (
              <div className="space-y-3">
                {keys.map((key) => (
                  <div
                    key={key.id}
                    className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                  >
                    <div>
                      <p className="font-medium text-sm">{key.name}</p>
                      <p className="font-mono text-xs text-gray-500">{key.key_prefix}••••••••••••••••••••</p>
                      <p className="text-xs text-gray-400">
                        {key.last_used_at
                          ? `最后使用: ${new Date(key.last_used_at).toLocaleString("zh-CN")}`
                          : "从未使用"}
                      </p>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge variant={key.status === "active" ? "default" : "secondary"}>
                        {key.status === "active" ? "有效" : "已注销"}
                      </Badge>
                      {key.status === "active" && (
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => handleRevoke(key.id)}
                        >
                          注销
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Usage guide */}
        <Card>
          <CardHeader>
            <CardTitle>使用方式</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <p className="text-sm text-gray-600">将以下地址设为 API Base URL，使用您的平台 Key 即可：</p>
            <code className="block bg-gray-100 px-3 py-2 rounded text-sm font-mono">
              {typeof window !== "undefined" ? window.location.origin : "https://yourdomain.com"}/v1
            </code>
            <p className="text-sm text-gray-500">完全兼容 OpenAI SDK，无需修改代码。</p>
          </CardContent>
        </Card>
      </div>

      {/* Show new key dialog - shown only once */}
      <Dialog open={showDialog} onOpenChange={(open) => { if (!open) { setNewKey(null); setShowDialog(false); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key 已创建</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <p className="text-sm text-red-600 font-medium">
              ⚠️ 请立即复制并妥善保存，此 Key 只显示一次！
            </p>
            <code className="block bg-gray-100 p-3 rounded text-sm font-mono break-all">
              {newKey}
            </code>
            <Button onClick={copyKey} className="w-full">
              复制 Key
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </UserLayout>
  );
}
