// API service for communicating with the backend
const API_BASE_URL = "http://" + window.location.host + "/api/v1";

export interface MusicPlayerState {
	isPlaying: boolean;
	currentSong?: string;
	artist?: string;
	url?: string;
	songDuration?: number;
	imageSrc?: string;
	elapsedSeconds?: number;
	volume?: number;
}

export interface ApiResponse<T> {
	data?: T;
	error?: string;
}

// Generic fetch wrapper with error handling
async function apiRequest<T>(
	endpoint: string,
	options?: RequestInit,
): Promise<ApiResponse<T>> {
	try {
		const response = await fetch(`${API_BASE_URL}${endpoint}`, {
			headers: {
				"Content-Type": "application/json",
				...options?.headers,
			},
			...options,
		});

		if (!response.ok) {
			return { error: `HTTP ${response.status}: ${response.statusText}` };
		}

		const data = await response.json();
		return { data };
	} catch (error) {
		return { error: error instanceof Error ? error.message : "Unknown error" };
	}
}

// Health check for Pear Desktop service
export async function checkServiceHealth(): Promise<boolean> {
	try {
		// The backend should have a health endpoint or we can use the music state endpoint as health check
		const response = await fetch(`${API_BASE_URL}/music/state`, {
			method: "GET",
			headers: {
				"Content-Type": "application/json",
			},
		});
		return response.ok;
	} catch {
		return false;
	}
}

// Get current music player state
export async function getMusicPlayerState(): Promise<
	ApiResponse<MusicPlayerState>
> {
	return apiRequest<MusicPlayerState>("/music/state");
}

// Set music player state
export async function setMusicPlayerState(
	state: MusicPlayerState,
): Promise<ApiResponse<any>> {
	return apiRequest("/music/state", {
		method: "POST",
		body: JSON.stringify(state),
	});
}
