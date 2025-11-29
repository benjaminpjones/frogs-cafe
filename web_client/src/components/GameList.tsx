import React from 'react';
import { Game } from '../types';
import './GameList.css';

interface GameListProps {
  games: Game[];
  selectedGame: Game | null;
  onSelectGame: (game: Game) => void;
}

const GameList: React.FC<GameListProps> = ({ games, selectedGame, onSelectGame }) => {
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
              className={`game-item ${selectedGame?.id === game.id ? 'selected' : ''}`}
              onClick={() => onSelectGame(game)}
            >
              <div className="game-info">
                <span className="game-id">Game #{game.id}</span>
                <span className={`game-status status-${game.status}`}>
                  {game.status}
                </span>
              </div>
              <div className="game-details">
                <span>Board: {game.board_size}×{game.board_size}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default GameList;
