import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import type { RootState } from "../../app/store";

// Define a type for the slice state
export interface ITwitchState {
	expires_in: string;
	hostname: string;
	twitch_song_request_reward_id: string;
	login: string;
	expires_in_bot: string;
	login_bot: string;
}

const initialState: ITwitchState = {
	expires_in: "",
	hostname: "127.0.0.1:3999",
	twitch_song_request_reward_id: "",
	login: "",
	login_bot: "",
	expires_in_bot: "",
};

export const twitchStateSlice = createSlice({
	name: "twitchstate",
	initialState,
	reducers: {
		setTwitchInfo: (state, action: PayloadAction<Partial<ITwitchState>>) => {
			for (const [key, value] of Object.entries(action.payload)) {
				(state as any)[key] = value as any;
			}
		},
	},
});

export const { setTwitchInfo } = twitchStateSlice.actions;

export const selectTwitchState = (state: RootState) => state.musicPlayerState;

export default twitchStateSlice.reducer;
