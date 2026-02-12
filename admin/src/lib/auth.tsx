import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  type ReactNode,
} from "react";
import { api, setAccessToken, ApiRequestError } from "./api";

type Admin = {
  id: string;
  email: string;
};

type AuthState =
  | { status: "loading" }
  | { status: "authenticated"; admin: Admin }
  | { status: "unauthenticated" };

type AuthContextValue = {
  state: AuthState;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading" });

  // Attempt silent refresh on mount to restore session
  useEffect(() => {
    let cancelled = false;

    async function restore() {
      const refreshed = await api.silentRefresh();
      if (cancelled) return;

      if (refreshed) {
        try {
          const admin = await api.get<Admin>("/admin/api/auth/me");
          if (!cancelled) {
            setState({ status: "authenticated", admin });
          }
        } catch {
          if (!cancelled) {
            setAccessToken(null);
            setState({ status: "unauthenticated" });
          }
        }
      } else {
        setState({ status: "unauthenticated" });
      }
    }

    restore();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    try {
      const result = await api.post<{ access_token: string }>(
        "/admin/api/auth/login",
        { email, password },
      );
      setAccessToken(result.access_token);
      const admin = await api.get<Admin>("/admin/api/auth/me");
      setState({ status: "authenticated", admin });
    } catch (err) {
      setAccessToken(null);
      if (err instanceof ApiRequestError) {
        throw err;
      }
      throw new Error("Login failed. Please try again.");
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await api.post("/admin/api/auth/logout");
    } catch {
      // Logout may fail if already expired -- that's fine
    } finally {
      setAccessToken(null);
      setState({ status: "unauthenticated" });
    }
  }, []);

  return (
    <AuthContext.Provider value={{ state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
