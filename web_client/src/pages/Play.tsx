import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../contexts/AuthContext";
import GameList from "../components/GameList";
import { Game } from "../types";
import { API_URL } from "../config";
import "./Play.css";

export function Play() {
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { player, token, setShowAuthModal } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    fetchGames();
  }, []);

  const fetchGames = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_URL}/api/v1/games?status=waiting`);
      const data = await response.json();
      setGames(data || []);
      setError(null);
    } catch (error) {
      console.error("Error fetching games:", error);
      setError("Failed to load games. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const createNewGame = async () => {
    if (!token) {
      setShowAuthModal(true);
      return;
    }

    try {
      const response = await fetch(`${API_URL}/api/v1/games`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          board_size: 19,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to create game");
      }

      const newGame = await response.json();
      setGames([newGame, ...games]);
    } catch (error) {
      console.error("Error creating game:", error);
      alert("Failed to create game. Please try again.");
    }
  };

  const joinGame = async (gameId: number) => {
    if (!token) {
      setShowAuthModal(true);
      return;
    }

    try {
      const response = await fetch(`${API_URL}/api/v1/games/${gameId}/join`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error);
      }

      // Successfully joined, navigate to the game
      navigate(`/game/${gameId}`);
    } catch (error) {
      console.error("Error joining game:", error);
      alert(`Failed to join game: ${error}`);
    }
  };

  const handleGameClick = (gameId: number) => {
    navigate(`/game/${gameId}`);
  };

  return (
    <div className="play-page">
      <div className="play-header">
        <h1>Play Go</h1>
        <p>Create a new game or join an existing challenge</p>
      </div>

      <div className="play-actions">
        <button onClick={createNewGame} className="create-game-btn">
          + Create New Game
        </button>
        <button onClick={fetchGames} className="refresh-btn">
          ↻ Refresh
        </button>
      </div>

      {loading ? (
        <div className="loading">Loading games...</div>
      ) : error ? (
        <div className="error-message">{error}</div>
      ) : (
        <>
          <h2>Available Games ({games.length})</h2>
          <GameList
            games={games}
            onGameClick={handleGameClick}
            onJoinGame={joinGame}
            currentPlayerId={player?.id || null}
            showJoinButton={true}
          />
        </>
      )}
    </div>
  );
}
