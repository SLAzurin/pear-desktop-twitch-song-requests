import { useEffect, useState } from "react";
import "./App.css";

function App() {
	const [params, setParams] = useState<URLSearchParams>();

	// on page load, set the oauth params
	useEffect(() => {
		const params = new URLSearchParams();
		params.append("response_type", "token");
		params.append(
			"client_id",
			import.meta.env.VITE_TWITCH_CLIENT_ID ?? "7k7nl6w8e0owouonj7nb9g3k5s6gs5",
		);
		params.append(
			"redirect_uri",
			"http://" + window.location.host + "/oauth/twitch",
		);
		params.append(
			"scope",
			[
				"chat:read",
				"chat:edit",
				"channel:moderate",
				"whispers:read",
				"whispers:edit",
				"moderator:manage:banned_users",
				"channel:read:redemptions",
				"user:read:chat",
				"user:write:chat",
				"user:bot",
			].join(" "),
		);
		setParams(params);
		/*
			example fragment: #access_token=73d0f8mkabpbmjp921asv2jaidwxn&scope=channel%3Amanage%3Apolls+channel%3Aread%3Apolls&state=c3ab8aa609ea11e793ae92361f002671&token_type=bearer
			example error: ?error=redirect_mismatch&error_description=Parameter+redirect_uri+does+not+match+registered+URI
		*/
	}, []);

	return (
		<>
			<a href={`https://id.twitch.tv/oauth2/authorize?${params}`}>
				Connect with Twitch
			</a>
		</>
	);
}

export default App;
