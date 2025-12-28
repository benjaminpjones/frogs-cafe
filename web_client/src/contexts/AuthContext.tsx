import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { Player, AuthResponse } from "../types";
import { API_URL } from "../config";

interface AuthContextType {
  player: Player | null;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  register: (
    username: string,
    email: string,
    password: string,
  ) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
  showAuthModal: boolean;
  setShowAuthModal: (show: boolean) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps) => {
  const [player, setPlayer] = useState<Player | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showAuthModal, setShowAuthModal] = useState(false);

  useEffect(() => {
    // Check for stored token on mount
    const storedToken = localStorage.getItem("token");
    const storedPlayer = localStorage.getItem("player");

    if (storedToken && storedPlayer) {
      setToken(storedToken);
      setPlayer(JSON.parse(storedPlayer));
    }
    setIsLoading(false);
  }, []);

  const login = async (username: string, password: string) => {
    const response = await fetch(`${API_URL}/api/v1/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error || "Login failed");
    }

    const data: AuthResponse = await response.json();
    setToken(data.token);
    setPlayer(data.player);
    localStorage.setItem("token", data.token);
    localStorage.setItem("player", JSON.stringify(data.player));
  };

  const register = async (
    username: string,
    email: string,
    password: string,
  ) => {
    const response = await fetch(`${API_URL}/api/v1/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ username, email, password }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error || "Registration failed");
    }

    const data: AuthResponse = await response.json();
    setToken(data.token);
    setPlayer(data.player);
    localStorage.setItem("token", data.token);
    localStorage.setItem("player", JSON.stringify(data.player));
  };

  const logout = () => {
    setToken(null);
    setPlayer(null);
    localStorage.removeItem("token");
    localStorage.removeItem("player");
  };

  return (
    <AuthContext.Provider
      value={{
        player,
        token,
        login,
        register,
        logout,
        isLoading,
        showAuthModal,
        setShowAuthModal,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};
