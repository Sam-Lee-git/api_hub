export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("access_token");
}

export function setAccessToken(token: string) {
  localStorage.setItem("access_token", token);
}

export function clearAuth() {
  localStorage.removeItem("access_token");
  localStorage.removeItem("user");
}

export function getUser() {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem("user");
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

export function setUser(user: object) {
  localStorage.setItem("user", JSON.stringify(user));
}

export function isLoggedIn(): boolean {
  return !!getAccessToken();
}
