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
  platform_super_admin?: boolean;
}

interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: string;
}

export interface TenantRoleSummary {
  id: string;
  name: string;
}

interface AuthContextValue {
  user: User | null;
  tenantId: string;
  activeTenantId: string;
  activeTenantName: string;
  tenants: Tenant[];
  /** Roles bound to the current user in the active tenant (empty if tenant scope does not match token). */
  myRolesInTenant: TenantRoleSummary[];
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  switchTenant: (tenantId: string) => void;
  refreshMyRoles: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue>({
  user: null,
  tenantId: "",
  activeTenantId: "",
  activeTenantName: "",
  tenants: [],
  myRolesInTenant: [],
  loading: true,
  login: async () => {},
  logout: () => {},
  switchTenant: () => {},
  refreshMyRoles: async () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [tenantId, setTenantId] = useState("");
  const [activeTenantId, setActiveTenantId] = useState("");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [myRolesInTenant, setMyRolesInTenant] = useState<TenantRoleSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  const fetchTenants = useCallback(async () => {
    try {
      const data = await api.get<Tenant[]>("/tenants");
      setTenants(Array.isArray(data) ? data : []);
    } catch {
      setTenants([]);
    }
  }, []);

  const refreshMyRoles = useCallback(async () => {
    const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;
    if (!token || !user) {
      setMyRolesInTenant([]);
      return;
    }
    if (!activeTenantId) {
      setMyRolesInTenant([]);
      return;
    }
    if (!user.platform_super_admin && activeTenantId !== tenantId) {
      setMyRolesInTenant([]);
      return;
    }
    try {
      const data = await api.get<{ roles: TenantRoleSummary[] }>(`/tenants/${activeTenantId}/me/roles`);
      setMyRolesInTenant(Array.isArray(data?.roles) ? data.roles : []);
    } catch {
      setMyRolesInTenant([]);
    }
  }, [user, activeTenantId, tenantId]);

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
        const saved = localStorage.getItem("activeTenantId");
        setActiveTenantId(saved || data.tenant_id);
        localStorage.setItem("user", JSON.stringify(data.user));
        return fetchTenants();
      })
      .catch(() => {
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        if (!pathname.startsWith("/login")) {
          router.replace("/login");
        }
      })
      .finally(() => setLoading(false));
  }, [pathname, router, fetchTenants]);

  useEffect(() => {
    if (!loading && user) {
      void refreshMyRoles();
    }
  }, [loading, user, refreshMyRoles]);

  const login = useCallback(
    async (email: string, password: string) => {
      const resp = await api.post<{
        access_token: string;
        expires_in: number;
        user: { id: string; tenant_id: string; email: string; display_name: string; platform_super_admin?: boolean };
      }>("/auth/login", { email, password });

      localStorage.setItem("token", resp.access_token);
      localStorage.setItem("user", JSON.stringify(resp.user));
      setUser({
        ...resp.user,
        status: "active",
        platform_super_admin: !!resp.user.platform_super_admin,
      });
      setTenantId(resp.user.tenant_id);
      setActiveTenantId(resp.user.tenant_id);
      localStorage.setItem("activeTenantId", resp.user.tenant_id);
      await fetchTenants();
      router.push("/overview");
    },
    [router, fetchTenants]
  );

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    localStorage.removeItem("activeTenantId");
    setUser(null);
    setTenantId("");
    setActiveTenantId("");
    setTenants([]);
    setMyRolesInTenant([]);
    router.push("/login");
  }, [router]);

  const switchTenant = useCallback((tid: string) => {
    setActiveTenantId(tid);
    localStorage.setItem("activeTenantId", tid);
  }, []);

  const activeTenantName = tenants.find(t => t.id === activeTenantId)?.name || "";

  return (
    <AuthContext.Provider value={{
      user, tenantId, activeTenantId, activeTenantName, tenants,
      myRolesInTenant,
      loading, login, logout, switchTenant, refreshMyRoles,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
