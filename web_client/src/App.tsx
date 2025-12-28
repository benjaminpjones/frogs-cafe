import React from "react";
import { BrowserRouter, Routes, Route } from "react-router";
import { AuthProvider } from "./contexts/AuthContext";
import { Layout } from "./components/Layout";
import { Play } from "./pages/Play";
import { Watch } from "./pages/Watch";
import { GamePage } from "./pages/GamePage";
import { Chat } from "./pages/Chat";
import { About } from "./pages/About";
import { Player } from "./pages/Player";
import "./App.css";

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<Play />} />
            <Route path="watch" element={<Watch />} />
            <Route path="game/:id" element={<GamePage />} />
            <Route path="chat" element={<Chat />} />
            <Route path="about" element={<About />} />
            <Route path="player/:username" element={<Player />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
