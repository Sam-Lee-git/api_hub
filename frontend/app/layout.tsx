import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Toaster } from "@/components/ui/sonner";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "AI API Platform",
  description: "Unified AI API gateway with per-token billing",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" className={`${inter.className} h-full antialiased`}>
      <body className="min-h-full bg-gray-50">
        {children}
        <Toaster position="top-right" />
      </body>
    </html>
  );
}
