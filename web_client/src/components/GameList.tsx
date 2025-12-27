import React from 'react';
import { Game } from '../types';
import './GameList.css';

interface GameListProps {
  games: Game[];
  selectedGame: Game | null;
  onSelectGame: (game: Game) => void;
  onJoinGame: (gameId: number) => void;
  currentPlayerId: number | null;
}

const GameList: React.FC<GameListProps> = ({
  games,
  selectedGame,
  onSelectGame,
  onJoinGame,
  currentPlayerId,
}) => {
  const canJoinGame = (game: Game) => {
    // Can't join if not logged in or if it's your own game
    if (!currentPlayerId) return false;
    return game.status === "waiting" && game.creator_id !== currentPlayerId;
  };

  const isMyGame = (game: Game) => {
    if (!currentPlayerId) return false;
    return (
      game.creator_id === currentPlayerId ||
      game.black_player_id === currentPlayerId ||
      game.white_player_id === currentPlayerId
    );
  };

  return (
    <div className="game-list">
      <h2>Games</h2>
      {games.length === 0 ? (
        <p className="no-games">No games available</p>
      ) : (
        <ul>
          {games.map((game) => (
            <li
              key={game.id}
              className={`game-item ${
                selectedGame?.id === game.id ? "selected" : ""
              } ${isMyGame(game) ? "my-game" : ""}`}
              onClick={() => onSelectGame(game)}
            >
              <div className="game-info">
                <span className="game-id">Game #{game.id}</span>
                <span className={`game-status status-${game.status}`}>
                  {game.status}
                </span>
              </div>
              <div className="game-details">
                <span>
                  Board: {game.board_size}×{game.board_size}
                </span>
                {isMyGame(game) && (
                  <span className="my-game-badge">Your Game</span>
                )}
              </div>
              {canJoinGame(game) && (
                <button
                  className="join-btn"
                  onClick={(e) => {
                    e.stopPropagation();
                    onJoinGame(game.id);
                  }}
                >
                  Join Game
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default GameList;
