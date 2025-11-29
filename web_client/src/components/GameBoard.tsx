import React, { useEffect, useState, useRef } from 'react';
import { Game } from '../types';
import './GameBoard.css';

interface GameBoardProps {
  game: Game;
}

const GameBoard: React.FC<GameBoardProps> = ({ game }) => {
  const [board, setBoard] = useState<(string | null)[][]>([]);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080';

  useEffect(() => {
    // Initialize empty board
    const newBoard = Array(game.board_size)
      .fill(null)
      .map(() => Array(game.board_size).fill(null));
    setBoard(newBoard);

    // Connect to WebSocket
    const websocket = new WebSocket(`${WS_URL}/ws?game_id=${game.id}&user_id=1`);
    
    websocket.onopen = () => {
      console.log('WebSocket connected');
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      console.log('Received:', message);
      // Handle incoming moves
    };

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    websocket.onclose = () => {
      console.log('WebSocket disconnected');
    };

    setWs(websocket);

    return () => {
      websocket.close();
    };
  }, [game.id, game.board_size, WS_URL]);

  useEffect(() => {
    drawBoard();
  }, [board]);

  const drawBoard = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const cellSize = 30;
    const padding = 20;
    const boardSize = game.board_size;
    
    canvas.width = boardSize * cellSize + padding * 2;
    canvas.height = boardSize * cellSize + padding * 2;

    // Clear canvas
    ctx.fillStyle = '#DEB887';
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    // Draw grid lines
    ctx.strokeStyle = '#000';
    ctx.lineWidth = 1;

    for (let i = 0; i < boardSize; i++) {
      // Vertical lines
      ctx.beginPath();
      ctx.moveTo(padding + i * cellSize, padding);
      ctx.lineTo(padding + i * cellSize, padding + (boardSize - 1) * cellSize);
      ctx.stroke();

      // Horizontal lines
      ctx.beginPath();
      ctx.moveTo(padding, padding + i * cellSize);
      ctx.lineTo(padding + (boardSize - 1) * cellSize, padding + i * cellSize);
      ctx.stroke();
    }

    // Draw star points (for 19x19 board)
    if (boardSize === 19) {
      const starPoints = [
        [3, 3], [3, 9], [3, 15],
        [9, 3], [9, 9], [9, 15],
        [15, 3], [15, 9], [15, 15]
      ];
      
      starPoints.forEach(([x, y]) => {
        ctx.beginPath();
        ctx.arc(padding + x * cellSize, padding + y * cellSize, 4, 0, 2 * Math.PI);
        ctx.fillStyle = '#000';
        ctx.fill();
      });
    }

    // Draw stones
    board.forEach((row, y) => {
      row.forEach((stone, x) => {
        if (stone) {
          ctx.beginPath();
          ctx.arc(
            padding + x * cellSize,
            padding + y * cellSize,
            cellSize * 0.4,
            0,
            2 * Math.PI
          );
          ctx.fillStyle = stone === 'black' ? '#000' : '#fff';
          ctx.fill();
          ctx.strokeStyle = '#000';
          ctx.lineWidth = 1;
          ctx.stroke();
        }
      });
    });
  };

  const handleBoardClick = (event: React.MouseEvent<HTMLCanvasElement>) => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    const cellSize = 30;
    const padding = 20;
    
    const x = Math.round((event.clientX - rect.left - padding) / cellSize);
    const y = Math.round((event.clientY - rect.top - padding) / cellSize);

    if (x >= 0 && x < game.board_size && y >= 0 && y < game.board_size) {
      if (!board[y][x]) {
        // Send move via WebSocket
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({
            type: 'move',
            data: { x, y, game_id: game.id }
          }));
        }

        // Update local board (simplified - should wait for server confirmation)
        const newBoard = board.map(row => [...row]);
        newBoard[y][x] = 'black'; // Simplified - should track current player
        setBoard(newBoard);
      }
    }
  };

  return (
    <div className="game-board">
      <div className="game-header">
        <h2>Game #{game.id}</h2>
        <div className="game-meta">
          <span className={`status status-${game.status}`}>{game.status}</span>
          <span>Board Size: {game.board_size}×{game.board_size}</span>
        </div>
      </div>
      <div className="board-container">
        <canvas
          ref={canvasRef}
          onClick={handleBoardClick}
          className="board-canvas"
        />
      </div>
    </div>
  );
};

export default GameBoard;
