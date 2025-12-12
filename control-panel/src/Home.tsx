import { Link } from "react-router";
import { useAppSelector } from "./app/hooks";

export function Home() {
	const twitchState = useAppSelector((state) => state.twitchState);
	return (
		<div>
			<Link to="/oauth/twitch-connect">
				{twitchState.expires_in !== ""
					? "Refresh Twitch token"
					: "Connect with twitch"}
			</Link>
			<h3>
				{twitchState.expires_in == ""
					? "No Twitch token configured"
					: "Twitch token for " +
						twitchState.login +
						" expires on " +
						twitchState.expires_in}
			</h3>
			<br />
			<Link to="/oauth/twitch-connect">
				{twitchState.expires_in !== ""
					? "Refresh Twitch token"
					: "Connect with twitch"}
			</Link>
			<h3>
				{twitchState.expires_in_bot == ""
					? "No bot Twitch token configured"
					: "Twitch token for " +
						twitchState.login_bot +
						" expires on " +
						twitchState.expires_in_bot}
			</h3>
			<br />
			<br />
			<br />
			<Link to="/settings">Configure settings</Link>
		</div>
	);
}
