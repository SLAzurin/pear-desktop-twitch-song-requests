import { Dispatch, UnknownAction } from "@reduxjs/toolkit";
import { setTwitchInfo } from "./twitchSlice";

export const handleWsMessages = (
	data: string,
	dispatch: Dispatch<UnknownAction>,
) => {
	// Change this later its ugly af

	const d: MsgTwitchInfo = JSON.parse(data);
	console.log(d);
	dispatch(
		setTwitchInfo({
			expires_in: d.expiry_date,
			twitch_song_request_reward_id: d.reward_id,
			login: d.login,
			login_bot: d.login_bot,
			expires_in_bot: d.expiry_date_bot,
		}),
	);
};

interface MsgTwitchInfo {
	type: "TWITCH_INFO";
	login: string;
	login_bot: string;
	expiry_date: string;
	expiry_date_bot: string;
	stream_online: string;
	reward_id: string;
}
