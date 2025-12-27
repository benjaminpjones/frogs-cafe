import React, { useState, useEffect } from 'react';
import './App.css';
import GameList from './components/GameList';
import GameBoard from './components/GameBoard';
import { Auth } from "./components/Auth";
import { AuthProvider, useAuth } from "./contexts/AuthContext";
import { Game } from './types';
import { API_URL } from './config';

function AppContent() {
  const [games, setGames] = useState<Game[]>([]);
  const [selectedGame, setSelectedGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);
  const [showAuth, setShowAuth] = useState(false);
  const { player, token, logout, isLoading } = useAuth();

  useEffect(() => {
    // Fetch games for everyone, not just logged in users
    fetchGames();
  }, []);

  const fetchGames = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/games`);
      const data = await response.json();
      setGames(data || []);
      setLoading(false);
    } catch (error) {
      console.error("Error fetching games:", error);
      setLoading(false);
    }
  };

  const createNewGame = async () => {
    if (!token) {
      setShowAuth(true);
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
      setSelectedGame(newGame);
    } catch (error) {
      console.error("Error creating game:", error);
      alert("Failed to create game");
    }
  };

  const joinGame = async (gameId: number) => {
    if (!token) {
      setShowAuth(true);
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

      const updatedGame = await response.json();

      // Update the games list
      setGames(games.map((g) => (g.id === updatedGame.id ? updatedGame : g)));
      setSelectedGame(updatedGame);

      alert("Successfully joined game!");
    } catch (error) {
      console.error("Error joining game:", error);
      alert(`Failed to join game: ${error}`);
    }
  };

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="App">
      <header className="App-header">
        <h1>🐸 Frogs Café - Go Server</h1>
        <p>Play the ancient game of Go (Baduk/Weiqi)</p>
        <div className="user-info">
          {player ? (
            <>
              <span>
                Welcome, {player.username}! (Rating: {player.rating})
              </span>
              <button onClick={logout} className="logout-btn">
                Logout
              </button>
            </>
          ) : (
            <button onClick={() => setShowAuth(true)} className="login-btn">
              Login
            </button>
          )}
        </div>
      </header>

      <main className="App-main">
        <div className="sidebar">
          <button onClick={createNewGame} className="create-game-btn">
            Create New Game
          </button>
          {loading ? (
            <p>Loading games...</p>
          ) : (
            <GameList
              games={games}
              selectedGame={selectedGame}
              onSelectGame={setSelectedGame}
              onJoinGame={joinGame}
              currentPlayerId={player?.id || null}
            />
          )}
        </div>

        <div className="game-area">
          {selectedGame ? (
            <GameBoard game={selectedGame} />
          ) : (
            <div className="no-game-selected">
              <p>Select a game or create a new one to start playing</p>
            </div>
          )}
        </div>
      </main>

      {showAuth && (
        <div className="auth-modal-overlay" onClick={() => setShowAuth(false)}>
          <div className="auth-modal" onClick={(e) => e.stopPropagation()}>
            <button className="close-modal" onClick={() => setShowAuth(false)}>
              ×
            </button>
            <Auth onSuccess={() => setShowAuth(false)} />
          </div>
        </div>
      )}
    </div>
  );
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;
