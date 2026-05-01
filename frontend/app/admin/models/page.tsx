"use client";

import { useCallback, useEffect, useState } from "react";
import AdminLayout from "@/components/layout/AdminLayout";
import api from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Model } from "@/types/api";

export default function AdminModelsPage() {
  const [models, setModels] = useState<Model[]>([]);
  const [editing, setEditing] = useState<{ [id: number]: Partial<Model> }>({});
  const [saving, setSaving] = useState<number | null>(null);

  const loadModels = useCallback(async () => {
    const { data } = await api.get("/api/admin/models");
    setModels(data.data || []);
  }, []);

  useEffect(() => {
    let cancelled = false;
    api.get("/api/admin/models").then(({ data }) => {
      if (!cancelled) setModels(data.data || []);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleEdit = (model: Model, field: keyof Model, value: string | number) => {
    setEditing((prev) => ({
      ...prev,
      [model.id]: { ...prev[model.id], [field]: value },
    }));
  };

  const handleSave = async (model: Model) => {
    const changes = editing[model.id];
    if (!changes) return;
    setSaving(model.id);
    try {
      await api.put(`/api/admin/models/${model.id}`, {
        input_credits_per_1k: changes.input_credits_per_1k ?? model.input_credits_per_1k,
        output_credits_per_1k: changes.output_credits_per_1k ?? model.output_credits_per_1k,
        status: changes.status ?? model.status,
        display_name: changes.display_name ?? model.display_name,
      });
      toast.success("已保存");
      setEditing((prev) => { const n = { ...prev }; delete n[model.id]; return n; });
      loadModels();
    } catch {
      toast.error("保存失败");
    } finally {
      setSaving(null);
    }
  };

  const grouped = models.reduce((acc, m) => {
    if (!acc[m.provider_name]) acc[m.provider_name] = [];
    acc[m.provider_name].push(m);
    return acc;
  }, {} as Record<string, Model[]>);

  return (
    <AdminLayout>
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">模型定价管理</h1>
        <p className="text-sm text-gray-500">1 Credit = 0.001 CNY（即1元=1000 Credits）</p>

        {Object.entries(grouped).map(([provider, providerModels]) => (
          <Card key={provider}>
            <CardHeader>
              <CardTitle className="capitalize">{provider}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-gray-500 border-b">
                      <th className="pb-2 pr-4">模型 ID</th>
                      <th className="pb-2 pr-4">显示名称</th>
                      <th className="pb-2 pr-4">输入 Credits/1K</th>
                      <th className="pb-2 pr-4">输出 Credits/1K</th>
                      <th className="pb-2 pr-4">状态</th>
                      <th className="pb-2">操作</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y">
                    {providerModels.map((model) => {
                      const edit = editing[model.id] || {};
                      const isDirty = !!editing[model.id];
                      return (
                        <tr key={model.id}>
                          <td className="py-2 pr-4 font-mono text-xs">{model.model_id}</td>
                          <td className="py-2 pr-4">
                            <Input
                              value={edit.display_name ?? model.display_name}
                              onChange={(e) => handleEdit(model, "display_name", e.target.value)}
                              className="h-7 text-xs w-40"
                            />
                          </td>
                          <td className="py-2 pr-4">
                            <Input
                              type="number"
                              value={edit.input_credits_per_1k ?? model.input_credits_per_1k}
                              onChange={(e) => handleEdit(model, "input_credits_per_1k", parseInt(e.target.value))}
                              className="h-7 text-xs w-24"
                            />
                          </td>
                          <td className="py-2 pr-4">
                            <Input
                              type="number"
                              value={edit.output_credits_per_1k ?? model.output_credits_per_1k}
                              onChange={(e) => handleEdit(model, "output_credits_per_1k", parseInt(e.target.value))}
                              className="h-7 text-xs w-24"
                            />
                          </td>
                          <td className="py-2 pr-4">
                            <Badge
                              variant={model.status === "active" ? "default" : "secondary"}
                              className="cursor-pointer"
                              onClick={() => handleEdit(model, "status",
                                (edit.status ?? model.status) === "active" ? "disabled" : "active"
                              )}
                            >
                              {edit.status ?? model.status}
                            </Badge>
                          </td>
                          <td className="py-2">
                            {isDirty && (
                              <Button
                                size="sm"
                                onClick={() => handleSave(model)}
                                disabled={saving === model.id}
                              >
                                {saving === model.id ? "保存中" : "保存"}
                              </Button>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </AdminLayout>
  );
}
