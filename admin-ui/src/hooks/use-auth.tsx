import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react";
import { useNavigate, useLocation } from "react-router-dom";
import {
  login as apiLogin,
  logout as apiLogout,
  getMe,
  type User,
} from "@/api/client";
import { sseBus } from "@/sdui/sse-bus";

interface AuthContextType {
  user: User | null;
  loading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    let cancelled = false;
    const refresh = () => {
      getMe()
        .then((u) => {
          if (!cancelled) setUser(u);
        })
        .catch(() => {
          if (!cancelled) setUser(null);
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    };
    refresh();
    // Refetch when the current user (or any user we might be) changes.
    const unsubscribe = sseBus.subscribe((event) => {
      if (event.type === "ENTITY_CHANGED" && event.entity === "user") {
        refresh();
      }
    });
    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, []);

  // Redirect to login if not authenticated and not already on login page
  useEffect(() => {
    if (!loading && !user && location.pathname !== "/admin/login") {
      navigate("/admin/login", { replace: true });
    }
  }, [loading, user, location.pathname, navigate]);

  const login = useCallback(
    async (email: string, password: string) => {
      await apiLogin({ email, password });
      const me = await getMe();
      setUser(me);
      navigate("/admin/dashboard", { replace: true });
    },
    [navigate]
  );

  const logout = useCallback(async () => {
    await apiLogout();
    setUser(null);
    navigate("/admin/login", { replace: true });
  }, [navigate]);

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated: !!user,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
