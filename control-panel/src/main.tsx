import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import store from "./app/store";
import { Provider } from "react-redux";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { OAuthTwitch } from "./features/oauthtwitch/oauthtwitch.tsx";
import { MusicPlayer } from "./features/musicplayer/MusicPlayer.tsx";

const root = ReactDOM.createRoot(
	document.getElementById("root") as HTMLElement,
);

root.render(
	<React.StrictMode>
		<Provider store={store}>
			<MusicPlayer></MusicPlayer>
			<BrowserRouter>
				<Routes>
					<Route path="/" element={<>home</>} />
					<Route path="/settings" element={<>settings</>} />
					<Route path="/queue" element={<>queue</>} />
					<Route path="/oauth/twitch-connect" element={<App />} />
					<Route path="/oauth/twitch-result" element={<>twitch result</>} />
					<Route path="/oauth/twitch" element={<OAuthTwitch />} />
				</Routes>
			</BrowserRouter>
		</Provider>
	</React.StrictMode>,
);
