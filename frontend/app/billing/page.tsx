"use client";

import { useEffect, useState, useCallback } from "react";
import UserLayout from "@/components/layout/UserLayout";
import api from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { CreditPackage } from "@/types/api";
import { QRCodeCanvas } from "qrcode.react";

interface OrderResult {
  order_no: string;
  channel: string;
  payment_url?: string;
  code_url?: string;
  expires_at: string;
}

export default function BillingPage() {
  const [packages, setPackages] = useState<CreditPackage[]>([]);
  const [balance, setBalance] = useState<number>(0);
  const [selectedChannel, setSelectedChannel] = useState<"alipay" | "wechat">("alipay");
  const [selectedPkg, setSelectedPkg] = useState<CreditPackage | null>(null);
  const [order, setOrder] = useState<OrderResult | null>(null);
  const [showPayDialog, setShowPayDialog] = useState(false);
  const [paying, setPaying] = useState(false);
  const [polling, setPolling] = useState(false);

  useEffect(() => {
    const load = async () => {
      const [pkgRes, balRes] = await Promise.allSettled([
        api.get("/api/billing/packages"),
        api.get("/api/billing/balance"),
      ]);
      if (pkgRes.status === "fulfilled") setPackages(pkgRes.value.data.data || []);
      if (balRes.status === "fulfilled") setBalance(balRes.value.data.balance);
    };
    load();
  }, []);

  const pollOrder = useCallback(async (orderNo: string) => {
    setPolling(true);
    const maxAttempts = 150; // 5 minutes at 2s intervals
    let attempts = 0;

    const interval = setInterval(async () => {
      attempts++;
      try {
        const { data } = await api.get(`/api/billing/orders/${orderNo}`);
        if (data.status === "paid") {
          clearInterval(interval);
          setPolling(false);
          setShowPayDialog(false);
          setOrder(null);
          toast.success(`充值成功！已获得 ${data.credits.toLocaleString()} Credits`);
          // Refresh balance
          const balRes = await api.get("/api/billing/balance");
          setBalance(balRes.data.balance);
        }
      } catch {
        // ignore polling errors
      }

      if (attempts >= maxAttempts) {
        clearInterval(interval);
        setPolling(false);
        toast.error("支付超时，如已付款请刷新页面");
      }
    }, 2000);
  }, []);

  const handlePay = async (pkg: CreditPackage) => {
    setSelectedPkg(pkg);
    setPaying(true);
    try {
      const { data } = await api.post("/api/billing/orders", {
        package_id: pkg.id,
        channel: selectedChannel,
      });
      setOrder(data);
      setShowPayDialog(true);
      pollOrder(data.order_no);
    } catch {
      toast.error("创建订单失败，请重试");
    } finally {
      setPaying(false);
    }
  };

  return (
    <UserLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">充值 Credits</h1>
          <div className="text-right">
            <p className="text-sm text-gray-500">当前余额</p>
            <p className="text-2xl font-bold">{balance.toLocaleString()} Credits</p>
          </div>
        </div>

        {balance <= 0 && (
          <Card className="border-red-200 bg-red-50">
            <CardContent className="pt-4">
              <p className="text-sm text-red-700">⚠️ 余额为零，请充值后才能调用 API</p>
            </CardContent>
          </Card>
        )}

        {/* Payment channel selector */}
        <Card>
          <CardHeader>
            <CardTitle>选择支付方式</CardTitle>
          </CardHeader>
          <CardContent>
            <Tabs value={selectedChannel} onValueChange={(v) => setSelectedChannel(v as "alipay" | "wechat")}>
              <TabsList>
                <TabsTrigger value="alipay">支付宝</TabsTrigger>
                <TabsTrigger value="wechat">微信支付</TabsTrigger>
              </TabsList>
            </Tabs>
          </CardContent>
        </Card>

        {/* Packages */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {packages.map((pkg) => {
            const totalCredits = pkg.credits + pkg.bonus_credits;
            const yuanAmount = pkg.amount_cny / 100;
            return (
              <Card key={pkg.id} className="hover:border-blue-400 transition-colors cursor-pointer">
                <CardHeader>
                  <CardTitle className="text-lg">{pkg.name}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div>
                    <p className="text-3xl font-bold">¥{yuanAmount}</p>
                    <p className="text-sm text-gray-500">
                      {pkg.credits.toLocaleString()} Credits
                      {pkg.bonus_credits > 0 && (
                        <span className="text-green-600 ml-1">
                          +{pkg.bonus_credits.toLocaleString()} 赠送
                        </span>
                      )}
                    </p>
                    <p className="text-xs text-gray-400">
                      总计 {totalCredits.toLocaleString()} Credits
                    </p>
                  </div>
                  {pkg.bonus_credits > 0 && (
                    <Badge variant="secondary" className="text-green-700 bg-green-50">
                      赠送 {Math.round((pkg.bonus_credits / pkg.credits) * 100)}%
                    </Badge>
                  )}
                  <Button
                    className="w-full"
                    onClick={() => handlePay(pkg)}
                    disabled={paying && selectedPkg?.id === pkg.id}
                  >
                    {paying && selectedPkg?.id === pkg.id ? "处理中..." :
                      selectedChannel === "alipay" ? "支付宝付款" : "微信扫码"}
                  </Button>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </div>

      {/* Payment dialog */}
      <Dialog open={showPayDialog} onOpenChange={(open) => {
        if (!open) {
          setShowPayDialog(false);
          setOrder(null);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {order?.channel === "alipay" ? "支付宝付款" : "微信扫码支付"}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 text-center">
            {order?.channel === "wechat" && order?.code_url && (
              <div className="flex justify-center">
                <QRCodeCanvas value={order.code_url} size={200} />
              </div>
            )}
            {order?.channel === "alipay" && order?.payment_url && (
              <Button
                className="w-full"
                onClick={() => window.open(order.payment_url, "_blank")}
              >
                打开支付宝付款
              </Button>
            )}
            <p className="text-sm text-gray-500">
              {polling ? "⏳ 等待付款确认中..." : ""}
            </p>
            <p className="text-xs text-gray-400">
              订单: {order?.order_no}
            </p>
          </div>
        </DialogContent>
      </Dialog>
    </UserLayout>
  );
}
