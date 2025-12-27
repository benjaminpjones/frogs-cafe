import React, { useEffect, useState } from "react";
import { Game } from "../types";
import { useAuth } from "../contexts/AuthContext";
import { API_URL, WS_URL } from "../config";
import "./GameBoard.css";

interface GameBoardProps {
  game: Game;
}

const GameBoard: React.FC<GameBoardProps> = ({ game }) => {
  const [board, setBoard] = useState<(string | null)[][]>([]);
  const [ws, setWs] = useState<WebSocket | null>(null);
  const [moveCount, setMoveCount] = useState(0);
  const { token, player } = useAuth();

  const cellSize = 30;
  const padding = 20;
  const boardSize = game.board_size;
  const svgSize = (boardSize - 1) * cellSize + padding * 2;

  // Determine which color the current player is
  const getMyColor = (): "black" | "white" | null => {
    if (!player) return null;
    return getColorForPlayer(player.id);
  };

  // Determine color based on player ID
  const getColorForPlayer = (playerId: number): "black" | "white" | null => {
    if (playerId === game.black_player_id) return "black";
    if (playerId === game.white_player_id) return "white";
    return null;
  };

  useEffect(() => {
    // Initialize empty board
    const newBoard = Array(game.board_size)
      .fill(null)
      .map(() => Array(game.board_size).fill(null));

    // Load existing moves from the server
    fetch(`${API_URL}/api/v1/games/${game.id}/moves`)
      .then((res) => res.json())
      .then((moves) => {
        moves.forEach((move: any) => {
          // Determine color based on which player made the move
          const color = getColorForPlayer(move.player_id);
          if (color) {
            newBoard[move.y][move.x] = color;
          }
        });
        setMoveCount(moves.length);
      })
      .catch((err) => console.error("Error loading moves:", err))
      .finally(() => {
        // Always set the board, even if fetch fails
        setBoard(newBoard);
      });

    // Connect to WebSocket for all users (authenticated and guests)
    // Token is optional - guests can watch games without authentication
    const wsUrl = token
      ? `${WS_URL}/ws?game_id=${game.id}&token=${token}`
      : `${WS_URL}/ws?game_id=${game.id}`;

    const websocket = new WebSocket(wsUrl);

    websocket.onopen = () => {
      console.log("WebSocket connected");
    };

    websocket.onmessage = (event) => {
      const message = JSON.parse(event.data);
      console.log("Received:", message);

      // Handle incoming moves from other players
      if (message.type === "move" && message.data) {
        const { x, y, player_id } = message.data;
        const color = getColorForPlayer(player_id);

        if (color) {
          setBoard((prevBoard) => {
            const newBoard = prevBoard.map((row) => [...row]);
            newBoard[y][x] = color;
            return newBoard;
          });
          setMoveCount((prevCount) => prevCount + 1);
        }
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

  // Effect to upgrade WebSocket connection when user logs in
  useEffect(() => {
    if (ws && ws.readyState === WebSocket.OPEN && token) {
      // Send authentication upgrade message
      ws.send(
        JSON.stringify({
          type: "authenticate",
          data: { token },
        }),
      );
    }
  }, [token, ws]);

  const handleIntersectionClick = (x: number, y: number) => {
    // Check if board is initialized and position is empty
    if (board.length === 0 || !board[y] || board[y][x]) {
      return;
    }

    // Get the color for the current player
    const myColor = getMyColor();
    if (!myColor) {
      console.error("You are not a player in this game");
      return;
    }

    // TODO: Check if it's this player's turn based on move count and color

    // Send move via WebSocket with player_id
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(
        JSON.stringify({
          type: "move",
          data: { x, y, game_id: game.id, player_id: player?.id },
        }),
      );
    }

    // Update local board optimistically
    const newBoard = board.map((row) => [...row]);
    newBoard[y][x] = myColor;
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
