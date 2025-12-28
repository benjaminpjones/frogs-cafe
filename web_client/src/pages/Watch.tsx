import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../contexts/AuthContext";
import GameList from "../components/GameList";
import { Game } from "../types";
import { API_URL } from "../config";
import "./Watch.css";

export function Watch() {
  const [games, setGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { player } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    fetchGames();
  }, []);

  const fetchGames = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_URL}/api/v1/games?status=active`);
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

  const handleGameClick = (gameId: number) => {
    navigate(`/game/${gameId}`);
  };

  return (
    <div className="watch-page">
      <div className="watch-header">
        <h1>Watch Games</h1>
        <p>Spectate ongoing games and learn from others</p>
      </div>

      <div className="watch-actions">
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
          <h2>Ongoing Games ({games.length})</h2>
          {games.length === 0 ? (
            <div className="no-games">
              <p>No games in progress right now.</p>
              <p>Check back later or create your own game!</p>
            </div>
          ) : (
            <GameList
              games={games}
              onGameClick={handleGameClick}
              currentPlayerId={player?.id || null}
              showJoinButton={false}
            />
          )}
        </>
      )}
    </div>
  );
}
