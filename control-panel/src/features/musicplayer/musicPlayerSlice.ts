import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import type { RootState } from "../../app/store";

// Define a type for the slice state
export interface IMusicPlayerState {
	isPlaying: boolean;
	songName: string;
	artistName: string;
	elapsedTime: number;
	songLength: number;
	albumArtUrl: string;
	videoUrl: string;
	isConnectedToBackend: boolean;
	isConenctedToPear: boolean;
	isConnectedToTwitch: boolean;
	hostname: string;
}

const initialState: IMusicPlayerState = {
	isPlaying: false,
	songName: "",
	albumArtUrl: "",
	artistName: "",
	elapsedTime: 0,
	isConenctedToPear: false,
	isConnectedToBackend: false,
	isConnectedToTwitch: false,
	songLength: 1,
	videoUrl: "",
	hostname: "localhost:26538",
};

export type TSongInfo = {
	songName: string;
	albumArtUrl: "";
	artistName: "";
};

export const musicPlayerStateSlice = createSlice({
	name: "musicplayerstate",
	initialState,
	reducers: {
		setSongInfo: (
			state,
			action: PayloadAction<
				Partial<{
					albumArtUrl: string;
					artistName: string;
					elapsedTime: number;
					isPlaying: boolean;
					songLength: number;
					songName: string;
					videoUrl: string;
				}>
			>,
		) => {
			for (const [key, value] of Object.entries(action.payload)) {
				(state as any)[key] = value as any;
			}
		},
	},
});

export const { setSongInfo } = musicPlayerStateSlice.actions;

export const selectMusicState = (state: RootState) => state.musicPlayerState;

export default musicPlayerStateSlice.reducer;
