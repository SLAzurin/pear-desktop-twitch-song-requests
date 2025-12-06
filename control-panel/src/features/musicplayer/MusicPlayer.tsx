import { useEffect, useState } from "react";
import { useAppSelector, useAppDispatch } from "../../app/hooks";
import { setSongInfo } from "./musicPlayerSlice";
import { handleWsMessages } from "./handleWsMessages";

export function MusicPlayer() {
	const [, setWs] = useState<WebSocket | null>(null);
	const [resetWs, setResetWs] = useState(false);
	const playerState = useAppSelector((state) => state.musicPlayerState);
	const dispatch = useAppDispatch();

	// Auto reconnect ws
	useEffect(() => {
		if (!resetWs) return;
		try {
			const wsUrl = `ws://${playerState.hostname}/api/v1/ws`;
			console.log("Starting Pear Desktop WebSocket...");
			const ws = new WebSocket(wsUrl);

			ws.onopen = () => {
				console.log("Pear Desktop WebSocket connected for music updates");
				setWs(ws);
			};

			ws.onmessage = (event) => {
				if (event.type == "message") {
					handleWsMessages(event.data as string, dispatch, { setSongInfo });
				} else {
					console.log("PEAR_DESKTOP_WS bin_data", event);
				}
			};

			ws.onerror = (error) => {
				console.error("WebSocket error:", error);
			};

			ws.onclose = () => {
				setWs(null);
				console.log("Connection Closed, will reconnect in 3s...");
				setWs(null);

				setTimeout(() => {
					setResetWs(true);
				}, 3000);
			};
		} catch (err) {
			console.error("Failed to create WebSocket connection:", err);
			setWs(null);
			console.log("Attempting to re-connect to pear desktop in 3s..");
			setTimeout(() => {
				setResetWs(true);
			}, 3000);
		}
		setResetWs(false);
	}, [resetWs]);

	// connect ws on page load
	useEffect(() => {
		setResetWs(true);
	}, []);

	return <></>;
}
