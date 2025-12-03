import { useEffect, useState } from "react";
import { Container, Row, Col, Card, Alert } from "react-bootstrap";
import { MusicPlayer } from "./features/musicplayer/MusicPlayer";
import "./App.css";

function App() {
	const [params, setParams] = useState<URLSearchParams>();
	const [oauthSuccess, setOauthSuccess] = useState(false);

	// on page load, set the oauth params
	useEffect(() => {
		const params = new URLSearchParams();
		params.append("client_id", import.meta.env.VITE_TWITCH_CLIENT_ID || "7k7nl6w8e0owouonj7nb9g3k5s6gs5");
		params.append("redirect_uri", "http://" + window.location.host + "/oauth/twitch-connect");
		params.append("response_type", "code");
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

		// Check if we have OAuth code in URL (after redirect)
		const urlParams = new URLSearchParams(window.location.search);
		const code = urlParams.get("code");
		if (code) {
			setOauthSuccess(true);
			// In a real app, you'd send this code to the backend
			console.log("OAuth code received:", code);
		}
	}, []);

	return (
		<Container className="py-4">
			<Row>
				<Col>
					<h1 className="mb-4">
						Pear Desktop - Twitch Song Request Control Panel
					</h1>

					{oauthSuccess && (
						<Alert variant="success" className="mb-4">
							<strong>Twitch OAuth successful!</strong> You are now connected to
							Twitch.
						</Alert>
					)}

					<Card className="mb-4">
						<Card.Header>
							<h5 className="mb-0">Twitch Integration</h5>
						</Card.Header>
						<Card.Body>
							<p>
								Connect your Twitch account to enable song requests from chat.
							</p>
							<a
								href={`https://id.twitch.tv/oauth2/authorize?${params}`}
								className="btn btn-primary"
							>
								Connect with Twitch
							</a>
						</Card.Body>
					</Card>

					<MusicPlayer />
				</Col>
			</Row>
		</Container>
	);
}

export default App;
