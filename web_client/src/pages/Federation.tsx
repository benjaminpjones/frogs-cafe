import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { useAuth } from "../contexts/AuthContext";
import { BIKChallenge, Game } from "../types";
import { API_URL } from "../config";
import "./Federation.css";

export function Federation() {
  const [challenges, setChallenges] = useState<BIKChallenge[]>([]);
  const [activeGames, setActiveGames] = useState<Game[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [accepting, setAccepting] = useState<string | null>(null);
  const { player, token, setShowAuthModal } = useAuth();
  const navigate = useNavigate();

  // Create form state
  const [boardSize, setBoardSize] = useState(19);
  const [colorAssignment, setColorAssignment] = useState("random");
  const [timeSystem, setTimeSystem] = useState("absolute");
  const [mainTime, setMainTime] = useState(300);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    fetchChallenges();
    fetchActiveGames();
  }, []);

  const fetchChallenges = async () => {
    try {
      setLoading(true);
      const response = await fetch(`${API_URL}/api/v1/bik/challenges`);
      const data = await response.json();
      setChallenges(data || []);
      setError(null);
    } catch (err) {
      console.error("Error fetching challenges:", err);
      setError("Failed to load challenges.");
    } finally {
      setLoading(false);
    }
  };

  const fetchActiveGames = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/games?status=active`);
      const data = await response.json();
      setActiveGames(data || []);
    } catch (err) {
      console.error("Error fetching active games:", err);
    }
  };

  const createChallenge = async () => {
    if (!token) {
      setShowAuthModal(true);
      return;
    }

    try {
      setCreating(true);
      const response = await fetch(`${API_URL}/api/v1/bik/challenges`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          boardSize,
          timeControl: {
            system: timeSystem,
            mainTime,
          },
          colorAssignment,
        }),
      });

      if (!response.ok) {
        throw new Error("Failed to create challenge");
      }

      setShowCreateForm(false);
      fetchChallenges();
    } catch (err) {
      console.error("Error creating challenge:", err);
      alert("Failed to create challenge.");
    } finally {
      setCreating(false);
    }
  };

  const acceptChallenge = async (challenge: BIKChallenge) => {
    if (!token || !player) {
      setShowAuthModal(true);
      return;
    }

    try {
      setAccepting(challenge.uri);

      // Ask our own server to proxy the Accept to the remote inbox
      const response = await fetch(`${API_URL}/api/v1/bik/challenges/accept`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          challengeUri: challenge.uri,
          inboxUrl: challenge.inboxUrl,
        }),
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to accept challenge");
      }

      const data = await response.json();
      navigate(`/game/${data.gameId}`);
    } catch (err) {
      console.error("Error accepting challenge:", err);
      alert(`Failed to accept challenge: ${err}`);
    } finally {
      setAccepting(null);
    }
  };

  const formatTimeRemaining = (expiresAt: string) => {
    const remaining = new Date(expiresAt).getTime() - Date.now();
    if (remaining <= 0) return "Expired";
    const minutes = Math.floor(remaining / 60000);
    if (minutes < 60) return `${minutes}m left`;
    const hours = Math.floor(minutes / 60);
    return `${hours}h ${minutes % 60}m left`;
  };

  const formatTime = (seconds: number) => {
    if (seconds >= 3600) return `${seconds / 3600}h`;
    if (seconds >= 60) return `${seconds / 60}m`;
    return `${seconds}s`;
  };

  return (
    <div className="federation-page">
      <div className="play-header">
        <h1>Federation</h1>
        <p>Cross-server challenges via the BIK protocol</p>
      </div>

      <div className="play-actions">
        <button
          onClick={() => {
            if (!token) {
              setShowAuthModal(true);
              return;
            }
            setShowCreateForm(!showCreateForm);
          }}
          className="create-game-btn"
        >
          + Create Challenge
        </button>
        <button onClick={() => { fetchChallenges(); fetchActiveGames(); }} className="refresh-btn">
          Refresh
        </button>
      </div>

      {showCreateForm && (
        <div className="create-challenge-form">
          <h3>New Federation Challenge</h3>

          <div className="form-row">
            <label>Board Size</label>
            <div className="board-size-options">
              {[9, 13, 19].map((size) => (
                <button
                  key={size}
                  className={`size-btn ${boardSize === size ? "active" : ""}`}
                  onClick={() => setBoardSize(size)}
                >
                  {size}x{size}
                </button>
              ))}
            </div>
          </div>

          <div className="form-row">
            <label>Your Color</label>
            <div className="color-options">
              {[
                { value: "random", label: "Random" },
                { value: "black", label: "Black" },
                { value: "white", label: "White" },
              ].map((opt) => (
                <button
                  key={opt.value}
                  className={`color-btn ${colorAssignment === opt.value ? "active" : ""}`}
                  onClick={() => setColorAssignment(opt.value)}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="form-row">
            <label>Time Control</label>
            <select
              value={timeSystem}
              onChange={(e) => setTimeSystem(e.target.value)}
              className="form-select"
            >
              <option value="absolute">Absolute</option>
              <option value="byoyomi">Byo-yomi</option>
            </select>
          </div>

          <div className="form-row">
            <label>Main Time</label>
            <div className="time-options">
              {[
                { value: 300, label: "5m" },
                { value: 600, label: "10m" },
                { value: 1800, label: "30m" },
                { value: 3600, label: "1h" },
              ].map((opt) => (
                <button
                  key={opt.value}
                  className={`time-btn ${mainTime === opt.value ? "active" : ""}`}
                  onClick={() => setMainTime(opt.value)}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="form-actions">
            <button
              onClick={createChallenge}
              className="create-game-btn"
              disabled={creating}
            >
              {creating ? "Creating..." : "Post Challenge"}
            </button>
            <button
              onClick={() => setShowCreateForm(false)}
              className="refresh-btn"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {activeGames.length > 0 && (
        <div className="active-games-section">
          <h2>Active Games ({activeGames.length})</h2>
          <div className="challenge-list">
            {activeGames.map((game) => (
              <div
                key={game.id}
                className="challenge-card active-game"
                onClick={() => navigate(`/game/${game.id}`)}
                style={{ cursor: "pointer" }}
              >
                <div className="challenge-header">
                  <span className="challenge-creator">Game #{game.id}</span>
                  <span className="challenge-badge active-badge">
                    Active
                  </span>
                </div>
                <div className="challenge-details">
                  <span className="challenge-tag">
                    {game.board_size}x{game.board_size}
                  </span>
                  <span className="challenge-tag">{game.status}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading challenges...</div>
      ) : error ? (
        <div className="error-message">{error}</div>
      ) : (
        <>
          <h2>Open Challenges ({challenges.length})</h2>
          {challenges.length === 0 ? (
            <div className="no-games">
              <p>No open challenges right now.</p>
              <p>Create one and other servers can accept it!</p>
            </div>
          ) : (
            <div className="challenge-list">
              {challenges.map((challenge) => (
                <div
                  key={challenge.uri}
                  className={`challenge-card ${challenge.remote ? "remote" : "local"}`}
                >
                  <div className="challenge-header">
                    <span className="challenge-creator">
                      {challenge.creator}
                    </span>
                    <div className="challenge-header-right">
                      {challenge.remote && (
                        <span className="challenge-badge remote-badge">
                          Remote
                        </span>
                      )}
                      <span className="challenge-expiry">
                        {formatTimeRemaining(challenge.expiresAt)}
                      </span>
                    </div>
                  </div>
                  <div className="challenge-details">
                    <span className="challenge-tag">
                      {challenge.boardSize}x{challenge.boardSize}
                    </span>
                    <span className="challenge-tag">
                      {challenge.timeControl?.system || "absolute"}{" "}
                      {challenge.timeControl?.mainTime
                        ? formatTime(challenge.timeControl.mainTime)
                        : ""}
                    </span>
                    <span className="challenge-tag">
                      {challenge.colorAssignment}
                    </span>
                  </div>
                  {challenge.remote && token && (
                    <button
                      className="accept-btn"
                      disabled={accepting === challenge.uri}
                      onClick={() => acceptChallenge(challenge)}
                    >
                      {accepting === challenge.uri
                        ? "Accepting..."
                        : "Accept Challenge"}
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
