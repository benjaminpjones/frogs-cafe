import React, { useEffect, useState, useRef, useCallback } from "react";
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
  const [currentGame, setCurrentGame] = useState<Game>(game);
  const [nextToPlay, setNextToPlay] = useState<"black" | "white">("black");
  const currentGameRef = useRef<Game>(game);
  const { token, player } = useAuth();

  const isRemote = !!game.remote_ws_url;

  useEffect(() => {
    currentGameRef.current = currentGame;
  }, [currentGame]);

  const cellSize = 30;
  const padding = 20;
  const boardSize = currentGame.board_size;
  const svgSize = (boardSize - 1) * cellSize + padding * 2;

  const getMyColor = (): "black" | "white" | null => {
    if (!player) return null;
    if (player.id === currentGameRef.current.black_player_id) return "black";
    if (player.id === currentGameRef.current.white_player_id) return "white";
    return null;
  };

  const getColorForPlayer = (playerId: number): "black" | "white" | null => {
    const g = currentGameRef.current;
    if (playerId === g.black_player_id) return "black";
    if (playerId === g.white_player_id) return "white";
    return null;
  };

  // Apply a list of BIK-format moves to a fresh board
  const applyMoves = useCallback(
    (moves: (number[] | null)[]) => {
      const newBoard = Array(boardSize)
        .fill(null)
        .map(() => Array(boardSize).fill(null));
      moves.forEach((move, i) => {
        if (move === null) return; // pass
        const color = i % 2 === 0 ? "black" : "white";
        const [col, row] = move;
        if (row >= 0 && row < boardSize && col >= 0 && col < boardSize) {
          newBoard[row][col] = color;
        }
      });
      setBoard(newBoard);
      setMoveCount(moves.length);
      setNextToPlay(moves.length % 2 === 0 ? "black" : "white");
    },
    [boardSize],
  );

  useEffect(() => {
    // Initialize empty board
    const emptyBoard = Array(game.board_size)
      .fill(null)
      .map(() => Array(game.board_size).fill(null));
    setBoard(emptyBoard);

    const connectWs = async () => {
      let wsUrl: string;

      if (isRemote && token) {
        // Get a BIK token for the remote game
        try {
          const resp = await fetch(
            `${API_URL}/api/v1/bik/token/${game.id}?gameURI=${encodeURIComponent(game.remote_game_uri!)}`,
            {
              method: "POST",
              headers: { Authorization: `Bearer ${token}` },
            },
          );
          const data = await resp.json();
          wsUrl = `${game.remote_ws_url}?token=${data.token}`;
        } catch (err) {
          console.error("Failed to get BIK token:", err);
          return;
        }
      } else if (isRemote) {
        // Guest watching a remote game
        wsUrl = game.remote_ws_url!;
      } else {
        // Local game
        if (!isRemote) {
          // Load existing moves via REST for local games
          try {
            const res = await fetch(`${API_URL}/api/v1/games/${game.id}/moves`);
            const moves = await res.json();
            if (moves && Array.isArray(moves)) {
              const newBoard = emptyBoard.map((row) => [...row]);
              moves.forEach((move: any) => {
                const color = getColorForPlayer(move.player_id);
                if (color) {
                  newBoard[move.y][move.x] = color;
                }
              });
              setBoard(newBoard);
              setMoveCount(moves.length);
            }
          } catch (err) {
            console.error("Error loading moves:", err);
          }
        }
        wsUrl = token
          ? `${WS_URL}/ws/games/${game.id}?token=${token}`
          : `${WS_URL}/ws/games/${game.id}`;
      }

      const websocket = new WebSocket(wsUrl);

      websocket.onopen = () => {
        console.log("WebSocket connected", isRemote ? "(remote)" : "(local)");
      };

      websocket.onmessage = (event) => {
        const message = JSON.parse(event.data);

        // BIK spec: game_state — full sync on connect
        if (message.type === "game_state" && message.data) {
          const { moves, boardSize: bs, nextToPlay: ntp } = message.data;
          if (bs) {
            setCurrentGame((g) => ({ ...g, board_size: bs }));
          }
          if (Array.isArray(moves)) {
            applyMoves(moves);
          }
          if (ntp) setNextToPlay(ntp);
        }

        // BIK spec: move_played — incremental update
        if (message.type === "move_played" && message.data) {
          const { pos, color, moveNumber, nextToPlay: ntp } = message.data;
          if (pos !== null && Array.isArray(pos)) {
            const [col, row] = pos;
            setBoard((prev) => {
              const newBoard = prev.map((r) => [...r]);
              if (row >= 0 && row < boardSize && col >= 0 && col < boardSize) {
                newBoard[row][col] = color;
              }
              return newBoard;
            });
          }
          // Use authoritative moveNumber from server to avoid double-counting
          // from optimistic updates
          if (typeof moveNumber === "number") {
            setMoveCount(moveNumber);
          } else {
            setMoveCount((c) => c + 1);
          }
          if (ntp) setNextToPlay(ntp);
        }

        // Legacy local format: move with player_id
        if (message.type === "move" && message.data?.player_id) {
          const { x, y, player_id } = message.data;
          const color = getColorForPlayer(player_id);
          if (color) {
            setBoard((prev) => {
              const newBoard = prev.map((r) => [...r]);
              newBoard[y][x] = color;
              return newBoard;
            });
            setMoveCount((c) => c + 1);
          }
        }

        // Game status updates
        if (message.type === "game_update" && message.data?.game) {
          const updatedGame = {
            ...message.data.game,
            black_player_id: message.data.black_player_id,
            white_player_id: message.data.white_player_id,
            status: message.data.status,
          };
          setCurrentGame(updatedGame);
        }

        if (message.type === "game_over" && message.data) {
          setCurrentGame((g) => ({ ...g, status: "finished" }));
        }
      };

      websocket.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      websocket.onclose = () => {
        console.log("WebSocket disconnected");
      };

      setWs(websocket);

      return websocket;
    };

    let websocket: WebSocket | undefined;
    connectWs().then((ws) => {
      websocket = ws;
    });

    return () => {
      websocket?.close();
    };
  }, [game.id, game.board_size, isRemote]);

  // Auth upgrade for local games
  useEffect(() => {
    if (ws && ws.readyState === WebSocket.OPEN && token && !isRemote) {
      ws.send(JSON.stringify({ type: "authenticate", data: { token } }));
    }
  }, [token, ws, isRemote]);

  const handleIntersectionClick = (x: number, y: number) => {
    if (board.length === 0 || !board[y] || board[y][x]) return;

    const myColor = getMyColor();
    if (!myColor) return;
    if (myColor !== nextToPlay) return;

    if (ws && ws.readyState === WebSocket.OPEN) {
      // BIK spec format: pos is [col, row]
      ws.send(JSON.stringify({ type: "move", data: { pos: [x, y] } }));
    }

    // Optimistic update
    const newBoard = board.map((row) => [...row]);
    newBoard[y][x] = myColor;
    setBoard(newBoard);
    setMoveCount(moveCount + 1);
    setNextToPlay(myColor === "black" ? "white" : "black");
  };

  const renderGridLines = () => {
    const lines = [];
    for (let i = 0; i < boardSize; i++) {
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
        <h2>Game #{currentGame.id}</h2>
        <div className="game-meta">
          <span className={`status status-${currentGame.status}`}>
            {currentGame.status}
          </span>
          <span>
            Board Size: {currentGame.board_size}x{currentGame.board_size}
          </span>
          {isRemote && <span className="remote-indicator">Federation</span>}
        </div>
      </div>
      <div className="turn-indicator">
        {currentGame.status === "active" && (
          <span>
            {nextToPlay === getMyColor()
              ? "Your turn"
              : `${nextToPlay} to play`}
          </span>
        )}
      </div>
      <div className="board-container">
        <svg viewBox={`0 0 ${svgSize} ${svgSize}`} className="board-svg">
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
