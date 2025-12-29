import React from "react";
import "./Chat.css";

export function Chat() {
  return (
    <div className="chat-page">
      <div className="page-header">
        <h1>Chat</h1>
        <p>Connect with other Go players</p>
      </div>
      <div className="placeholder-content">
        <div className="placeholder-icon">💬</div>
        <h2>Chat Coming Soon</h2>
        <p>
          We're building a chat system where you can discuss games, strategies,
          and connect with the Go community.
        </p>
        <p className="placeholder-note">Stay tuned for updates!</p>
      </div>
    </div>
  );
}
