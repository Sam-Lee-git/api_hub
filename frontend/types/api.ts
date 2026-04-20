export interface User {
  id: number;
  email: string;
  display_name: string;
  role: "user" | "admin";
  status: "active" | "suspended";
  balance: number;
  created_at: string;
}

export interface APIKey {
  id: number;
  user_id: number;
  key_prefix: string;
  name: string;
  status: "active" | "revoked";
  last_used_at: string | null;
  created_at: string;
}

export interface CreditTransaction {
  id: number;
  type: "topup" | "deduction" | "refund" | "admin_adjust";
  amount: number;
  balance_after: number;
  ref_id: string;
  description: string;
  created_at: string;
}

export interface CreditPackage {
  id: number;
  name: string;
  amount_cny: number; // in fen
  credits: number;
  bonus_credits: number;
  is_active: boolean;
  display_order: number;
}

export interface PaymentOrder {
  id: number;
  user_id: number;
  order_no: string;
  channel: "alipay" | "wechat";
  amount_cny: number;
  credits: number;
  credits_to_add: number;
  status: "pending" | "paid" | "failed" | "refunded";
  paid_at: string | null;
  expires_at: string;
  created_at: string;
}

export interface Model {
  id: number;
  provider_name: string;
  model_id: string;
  display_name: string;
  input_credits_per_1k: number;
  output_credits_per_1k: number;
  context_window: number;
  supports_streaming: boolean;
  supports_vision: boolean;
  status: "active" | "hidden" | "disabled";
}

export interface UsageRecord {
  id: number;
  user_id: number;
  model_name: string;
  request_id: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  credits_charged: number;
  status: string;
  latency_ms: number;
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
}
