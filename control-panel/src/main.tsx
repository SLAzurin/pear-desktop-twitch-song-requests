import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
import ConnectWithTwitchEntry from "./features/oauthtwitch/ConnectWithTwitchEntry.tsx";
import store from "./app/store";
import { Provider } from "react-redux";
import { BrowserRouter, Routes, Route } from "react-router";
import { ProcessTwitchOAuth } from "./features/oauthtwitch/ProcessTwitchOAuth.tsx";
import { MusicPlayer } from "./features/musicplayer/MusicPlayer.tsx";
import { TwitchSuccess } from "./features/oauthtwitch/TwitchSuccess.tsx";
import { TwitchWS } from "./features/twitchws/TwitchWS.tsx";
import { Home } from "./Home.tsx";
import { Settings } from "./components/Settings.tsx";

const root = ReactDOM.createRoot(
	document.getElementById("root") as HTMLElement,
);

root.render(
	<React.StrictMode>
		<Provider store={store}>
			<MusicPlayer />
			<TwitchWS />
			<BrowserRouter>
				<Routes>
					<Route path="/" element={<Home />} />
					<Route path="/settings" element={<Settings />} />
					<Route path="/queue" element={<>queue</>} />
					<Route path="/oauth">
						<Route
							path="twitch-connect"
							element={<ConnectWithTwitchEntry forBot={false} />}
						/>
						<Route
							path="twitch-connect-bot"
							element={<ConnectWithTwitchEntry forBot={true} />}
						/>
						<Route path="twitch-success" element={<TwitchSuccess />} />
						<Route path="twitch" element={<ProcessTwitchOAuth />} />
					</Route>
				</Routes>
			</BrowserRouter>
		</Provider>
	</React.StrictMode>,
);
