import { createSlice } from "@reduxjs/toolkit";
import type { RootState } from "../../app/store";

// Define a type for the slice state
export interface ITwitchState {
	hostname: string;
}

const initialState: ITwitchState = {
	hostname: "127.0.0.1:3999",
};

export const twitchStateSlice = createSlice({
	name: "twitchstate",
	initialState,
	reducers: {},
});

// export const {  } = twitchStateSlice.actions;

export const selectTwitchState = (state: RootState) => state.musicPlayerState;

export default twitchStateSlice.reducer;
