"use client";

import { createContext, useContext, useEffect, useState, ReactNode, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";
import { api } from "@/lib/api/client";

interface User {
  id: string;
  tenant_id: string;
  email: string;
  display_name: string;
  status: string;
}

interface AuthContextValue {
  user: User | null;
  tenantId: string;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  tenantId: "",
  loading: true,
  login: async () => {},
  logout: () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [tenantId, setTenantId] = useState("");
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      setLoading(false);
      if (!pathname.startsWith("/login")) {
        router.replace("/login");
      }
      return;
    }

    api
      .get<{ user: User; tenant_id: string }>("/me")
      .then((data) => {
        setUser(data.user);
        setTenantId(data.tenant_id);
        localStorage.setItem("user", JSON.stringify(data.user));
      })
      .catch(() => {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        if (!pathname.startsWith("/login")) {
          router.replace("/login");
        }
      })
      .finally(() => setLoading(false));
  }, [pathname, router]);

  const login = useCallback(
    async (email: string, password: string) => {
      const resp = await api.post<{
        access_token: string;
        expires_in: number;
        user: { id: string; tenant_id: string; email: string; display_name: string };
      }>("/auth/login", { email, password });

      localStorage.setItem("token", resp.access_token);
      localStorage.setItem("user", JSON.stringify(resp.user));
      setUser({
        ...resp.user,
        status: "active",
      });
      setTenantId(resp.user.tenant_id);
      router.push("/overview");
    },
    [router]
  );

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    setUser(null);
    setTenantId("");
    router.push("/login");
  }, [router]);

  return (
    <AuthContext.Provider value={{ user, tenantId, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
