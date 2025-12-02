import React from "react";
import ReactDOM from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import store from "./app/store";
import { Provider } from "react-redux";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { OAuthTwitch } from "./features/oauthtwitch/oauthtwitch.tsx";

const root = ReactDOM.createRoot(
	document.getElementById("root") as HTMLElement,
);

root.render(
	<React.StrictMode>
		<Provider store={store}>
			<BrowserRouter>
				<Routes>
					<Route path="/" element={<App />} />
					<Route path="/settings" element={<></>} />
					<Route path="/queue" element={<></>} />
					<Route path="/oauth/twitch-connect" element={<OAuthTwitch />} />
					<Route path="/oauth/twitch-result" element={<OAuthTwitch />} />
					<Route path="/oauth/twitch" element={<OAuthTwitch />} />
				</Routes>
			</BrowserRouter>
		</Provider>
	</React.StrictMode>,
);
