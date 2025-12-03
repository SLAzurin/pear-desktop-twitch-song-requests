import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [react()],
	server: {
		open: true,
		proxy: {
			"/api": {
				target:
					"http://" +
					(process.env.HOST || "127.0.0.1") +
					":" +
					(process.env.PORT || "3999"),
				changeOrigin: true,
				secure: false,
			},
			"/api/v1/music/ws": {
				target:
					"ws://" +
					(process.env.HOST || "127.0.0.1") +
					":" +
					(process.env.PORT || "3999"),
					ws: true,
					rewriteWsOrigin: true,
			}
		},
	},
	build: {
		outDir: "build",
		sourcemap: true,
	},
	test: {
		globals: true,
		environment: "jsdom",
		setupFiles: "src/setupTests",
		mockReset: true,
	},
});
