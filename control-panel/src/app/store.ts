import { configureStore } from "@reduxjs/toolkit";
import musicPlayerReducer from "../features/musicplayer/musicPlayerSlice";
import twitchReducer from "../features/twitchws/twitchSlice";

const store = configureStore({
	reducer: {
		musicPlayerState: musicPlayerReducer,
		twitchState: twitchReducer,
	},
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export default store;
