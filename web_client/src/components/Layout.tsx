import React from "react";
import { Outlet, NavLink } from "react-router";
import { useAuth } from "../contexts/AuthContext";
import { Auth } from "./Auth";
import "./Layout.css";

export function Layout() {
  const { player, logout, showAuthModal, setShowAuthModal } = useAuth();

  return (
    <div className="layout">
      <header className="layout-header">
        <div className="header-content">
          <div className="header-left">
            <h1 className="site-title">🐸 Frogs Café</h1>
            <nav className="main-nav">
              <NavLink to="/" end>
                Play
              </NavLink>
              <NavLink to="/watch">Watch</NavLink>
              <NavLink to="/chat">Chat</NavLink>
              <NavLink to="/about">About</NavLink>
            </nav>
          </div>
          <div className="auth-section">
            {player ? (
              <>
                <span className="welcome-text">
                  {player.username} ({player.rating})
                </span>
                <button onClick={logout} className="logout-btn">
                  Logout
                </button>
              </>
            ) : (
              <button
                onClick={() => setShowAuthModal(true)}
                className="login-btn"
              >
                Login
              </button>
            )}
          </div>
        </div>
      </header>

      <main className="layout-main">
        <Outlet />
      </main>

      {showAuthModal && (
        <div
          className="auth-modal-overlay"
          onClick={() => setShowAuthModal(false)}
        >
          <div className="auth-modal" onClick={(e) => e.stopPropagation()}>
            <button
              className="close-modal"
              onClick={() => setShowAuthModal(false)}
            >
              ×
            </button>
            <Auth onSuccess={() => setShowAuthModal(false)} />
          </div>
        </div>
      )}
    </div>
  );
}
