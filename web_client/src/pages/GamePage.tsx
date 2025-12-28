import React, { useState, useEffect } from "react";
import { useParams, Link } from "react-router";
import GameBoard from "../components/GameBoard";
import { Game } from "../types";
import { API_URL } from "../config";
import "./GamePage.css";

export function GamePage() {
  const { id } = useParams<{ id: string }>();
  const [game, setGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) {
      setError("Invalid game ID");
      setLoading(false);
      return;
    }

    fetchGame(id);
  }, [id]);

  const fetchGame = async (gameId: string) => {
    try {
      setLoading(true);
      const response = await fetch(`${API_URL}/api/v1/games/${gameId}`);

      if (response.status === 404) {
        setError("Game not found");
        setGame(null);
      } else if (!response.ok) {
        throw new Error("Failed to load game");
      } else {
        const data = await response.json();
        setGame(data);
        setError(null);
      }
    } catch (error) {
      console.error("Error fetching game:", error);
      setError("Failed to load game. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="game-page">
        <div className="loading">Loading game...</div>
      </div>
    );
  }

  if (error || !game) {
    return (
      <div className="game-page">
        <div className="error-container">
          <h2>Game Not Found</h2>
          <p>{error || "The game you're looking for doesn't exist."}</p>
          <Link to="/" className="back-link">
            ← Back to Lobby
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="game-page">
      <div className="game-page-header">
        <Link to="/" className="back-link">
          ← Back to Lobby
        </Link>
        <div className="game-meta">
          <h2>Game #{game.id}</h2>
          <span className={`game-status status-${game.status}`}>
            {game.status}
          </span>
        </div>
      </div>
      <div className="game-board-container">
        <GameBoard game={game} />
      </div>
    </div>
  );
}
