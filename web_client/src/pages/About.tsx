import React from "react";
import "./About.css";

export function About() {
  return (
    <div className="about-page">
      <div className="page-header">
        <h1>About Frogs Café</h1>
        <p>A place for Go enthusiasts</p>
      </div>
      <div className="about-content">
        <section className="about-section">
          <h2>🐸 Welcome to Frogs Café</h2>
          <p>
            Frogs Café is an open-source Go (Baduk/Weiqi) server where players
            can enjoy the ancient game of Go online. Whether you're a beginner
            or an experienced player, you're welcome here!
          </p>
        </section>

        <section className="about-section">
          <h2>🎮 Features</h2>
          <ul>
            <li>Play Go games online with other players</li>
            <li>Watch ongoing games and learn from others</li>
            <li>Real-time game updates via WebSocket</li>
            <li>Player rankings and ratings</li>
            <li>Open source and community-driven</li>
          </ul>
        </section>

        <section className="about-section">
          <h2>💻 Open Source</h2>
          <p>
            Frogs Café is open source and welcomes contributions! Check out our
            code, report issues, or contribute features on GitHub.
          </p>
          <a
            href="https://github.com/benjaminpjones/frogs-cafe"
            target="_blank"
            rel="noopener noreferrer"
            className="github-link"
          >
            <span>View on GitHub →</span>
          </a>
        </section>

        <section className="about-section">
          <h2>🤝 Community</h2>
          <p>
            Join our community of Go players! Connect with others in chat,
            discuss strategies, and help make Frogs Café even better.
          </p>
        </section>
      </div>
    </div>
  );
}
