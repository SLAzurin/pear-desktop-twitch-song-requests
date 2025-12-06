import { configureStore } from "@reduxjs/toolkit";
import musicPlayerReducer from "../features/musicplayer/musicPlayerSlice";

const store = configureStore({
	reducer: {
		musicPlayerState: musicPlayerReducer,
	},
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export default store;
