import React, { useEffect, useState } from "react";
import { Game } from "../types";
import "./GameBoard.css";

interface GameBoardProps {
  game: Game;
}

const GameBoard: React.FC<GameBoardProps> = ({ game }) => {
  const [board, setBoard] = useState<(string | null)[][]>([]);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [moveCount, setMoveCount] = useState(0);

  const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080";
  const cellSize = 30;
  const padding = 20;
  const boardSize = game.board_size;
  const svgSize = (boardSize - 1) * cellSize + padding * 2;

  useEffect(() => {
    // Initialize empty board
    const newBoard = Array(game.board_size)
      .fill(null)
      .map(() => Array(game.board_size).fill(null));

    // Load existing moves from the server
    const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";
    fetch(`${API_URL}/api/v1/games/${game.id}/moves`)
      .then((res) => res.json())
      .then((moves) => {
        moves.forEach((move: any) => {
          // Determine color based on move number (odd = black, even = white)
          const color = move.move_number % 2 === 1 ? "black" : "white";
          newBoard[move.y][move.x] = color;
        });
        setMoveCount(moves.length);
      })
      .catch((err) => console.error("Error loading moves:", err))
      .finally(() => {
        // Always set the board, even if fetch fails
        setBoard(newBoard);
      });

    // Connect to WebSocket
    const websocket = new WebSocket(
      `${WS_URL}/ws?game_id=${game.id}&user_id=1`,
    );

    websocket.onopen = () => {
      console.log("WebSocket connected");
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      console.log("Received:", message);

      // Handle incoming moves from other players
      if (message.type === "move" && message.data) {
        const { x, y } = message.data;
        setBoard((prevBoard) => {
          const newBoard = prevBoard.map((row) => [...row]);
          setMoveCount((prevCount) => {
            const color = prevCount % 2 === 0 ? "black" : "white";
            newBoard[y][x] = color;
            return prevCount + 1;
          });
          return newBoard;
        });
      }
    };

    websocket.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    websocket.onclose = () => {
      console.log("WebSocket disconnected");
    };

    setWs(websocket);

    return () => {
      websocket.close();
    };
  }, [game.id, game.board_size, WS_URL]);

  const handleIntersectionClick = (x: number, y: number) => {
    // Check if board is initialized and position is empty
    if (board.length === 0 || !board[y] || board[y][x]) {
      return;
    }

    // Determine color based on move count
    const color = moveCount % 2 === 0 ? "black" : "white";

    // Send move via WebSocket
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({
          type: "move",
          data: { x, y, game_id: game.id },
        }),
      );
    }

    // Update local board optimistically
    const newBoard = board.map((row) => [...row]);
    newBoard[y][x] = color;
    setBoard(newBoard);
    setMoveCount(moveCount + 1);
  };

  const renderGridLines = () => {
    const lines = [];

    for (let i = 0; i < boardSize; i++) {
      // Vertical lines
      lines.push(
        <line
          key={`v-${i}`}
          x1={padding + i * cellSize}
          y1={padding}
          x2={padding + i * cellSize}
          y2={padding + (boardSize - 1) * cellSize}
          stroke="#000"
          strokeWidth="1"
        />,
      );

      // Horizontal lines
      lines.push(
        <line
          key={`h-${i}`}
          x1={padding}
          y1={padding + i * cellSize}
          x2={padding + (boardSize - 1) * cellSize}
          y2={padding + i * cellSize}
          stroke="#000"
          strokeWidth="1"
        />,
      );
    }

    return lines;
  };

  const renderStarPoints = () => {
    if (boardSize !== 19) return null;

    const starPoints = [
      [3, 3],
      [3, 9],
      [3, 15],
      [9, 3],
      [9, 9],
      [9, 15],
      [15, 3],
      [15, 9],
      [15, 15],
    ];

    return starPoints.map(([x, y], idx) => (
      <circle
        key={`star-${idx}`}
        cx={padding + x * cellSize}
        cy={padding + y * cellSize}
        r="4"
        fill="#000"
      />
    ));
  };

  const renderIntersections = () => {
    const intersections = [];

    for (let y = 0; y < boardSize; y++) {
      for (let x = 0; x < boardSize; x++) {
        intersections.push(
          <circle
            key={`int-${x}-${y}`}
            cx={padding + x * cellSize}
            cy={padding + y * cellSize}
            r={cellSize * 0.45}
            fill="transparent"
            cursor="pointer"
            onClick={() => handleIntersectionClick(x, y)}
            className="intersection"
          />,
        );
      }
    }

    return intersections;
  };

  const renderStones = () => {
    const stones: JSX.Element[] = [];

    board.forEach((row, y) => {
      row.forEach((stone, x) => {
        if (stone) {
          stones.push(
            <circle
              key={`stone-${x}-${y}`}
              cx={padding + x * cellSize}
              cy={padding + y * cellSize}
              r={cellSize * 0.4}
              fill={stone === "black" ? "#000" : "#fff"}
              stroke="#000"
              strokeWidth="1"
              pointerEvents="none"
            />,
          );
        }
      });
    });

    return stones;
  };

  return (
    <div className="game-board">
      <div className="game-header">
        <h2>Game #{game.id}</h2>
        <div className="game-meta">
          <span className={`status status-${game.status}`}>{game.status}</span>
          <span>
            Board Size: {game.board_size}×{game.board_size}
          </span>
        </div>
      </div>
      <div className="board-container">
        <svg width={svgSize} height={svgSize} className="board-svg">
          <rect width={svgSize} height={svgSize} fill="#DEB887" />
          {renderGridLines()}
          {renderStarPoints()}
          {renderIntersections()}
          {renderStones()}
        </svg>
      </div>
    </div>
  );
};

export default GameBoard;
