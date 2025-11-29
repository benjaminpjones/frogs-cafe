import React, { useState, useEffect } from 'react';
import './App.css';
import GameList from './components/GameList';
import GameBoard from './components/GameBoard';
import { Game } from './types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

function App() {
  const [games, setGames] = useState<Game[]>([]);
  const [selectedGame, setSelectedGame] = useState<Game | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchGames();
  }, []);

  const fetchGames = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/games`);
      const data = await response.json();
      setGames(data || []);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching games:', error);
      setLoading(false);
    }
  };

  const createNewGame = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/games`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          board_size: 19,
        }),
      });
      const newGame = await response.json();
      setGames([newGame, ...games]);
      setSelectedGame(newGame);
    } catch (error) {
      console.error('Error creating game:', error);
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>🐸 Frogs Café - Go Server</h1>
        <p>Play the ancient game of Go (Baduk/Weiqi)</p>
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
    </div>
  );
}

export default App;
