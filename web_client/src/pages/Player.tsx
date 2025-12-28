import React from "react";
import { useParams } from "react-router";
import "./Player.css";

export function Player() {
  const { username } = useParams<{ username: string }>();

  return (
    <div className="player-page">
      <div className="page-header">
        <h1>Player Profile</h1>
        {username && <p>@{username}</p>}
      </div>
      <div className="placeholder-content">
        <div className="placeholder-icon">👤</div>
        <h2>Player Profiles Coming Soon</h2>
        <p>
          We're working on detailed player profiles where you can view stats,
          game history, and achievements.
        </p>
        {username && (
          <p className="viewing-player">
            You're trying to view: <strong>@{username}</strong>
          </p>
        )}
        <p className="placeholder-note">Check back soon!</p>
      </div>
    </div>
  );
}
