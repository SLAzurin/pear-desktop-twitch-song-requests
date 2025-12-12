import { useEffect, useState } from "react";
import "./ConnectWithTwitchEntry.css";

function ConnectWithTwitchEntry(props: { forBot: boolean }) {
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
		const scopes = [
			"user:read:chat",
			"user:write:chat",
			"user:bot",
			"channel:bot",
		];
		if (!props.forBot) {
			scopes.push(
				"channel:read:redemptions",
				"channel:read:vips",
				"moderation:read",
				"channel:read:subscriptions",
			);
		}
		params.append("scope", scopes.join(" "));
		if (props.forBot) {
			params.append("state", "bot");
		}
		setParams(params);
		/*
			example fragment: #access_token=73d0f8mkabpbmjp921asv2jaidwxn&scope=channel%3Amanage%3Apolls+channel%3Aread%3Apolls&state=c3ab8aa609ea11e793ae92361f002671&token_type=bearer
			example error: ?error=redirect_mismatch&error_description=Parameter+redirect_uri+does+not+match+registered+URI
		*/
	}, []);

	return (
		<>
			<a href={`https://id.twitch.tv/oauth2/authorize?${params}`}>
				{props.forBot
					? "Connect with Twitch bot account"
					: "Connect with Twitch main account"}
			</a>
		</>
	);
}

export default ConnectWithTwitchEntry;
