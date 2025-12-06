import {
	ActionCreatorWithPayload,
	Dispatch,
	UnknownAction,
} from "@reduxjs/toolkit";

interface IMsgTypeShuffleChanged {
	type: "SHUFFLE_CHANGED";
	shuffle: boolean;
}

interface IMsgTypeRepeatChanged {
	type: "REPEAT_CHANGED";
	repeat: "ALL" | "ONE" | "NONE";
}

interface IMsgTypePlayerStateChanged {
	type: "PLAYER_STATE_CHANGED";
	isPlaying: boolean;
	position: number;
}
interface IMsgTypeVolumeChanged {
	type: "VOLUME_CHANGED";
	volume: number;
	muted: boolean;
}

interface IMsgTypeVideoChanged {
	type: "VIDEO_CHANGED";
	song: SongInfo;
	position: number;
}

interface IMsgTypePositionChanged {
	type: "POSITION_CHANGED";
	position: number;
}

interface IMsgTypePlayerInfo {
	type: "PLAYER_INFO";
	isPlaying: boolean;
	muted: boolean;
	position: number;
	repeat: string;
	shuffle: boolean;
	volume: number;
	song: SongInfo;
}
export interface SongInfo {
	title: string;
	alternativeTitle?: string;
	artist: string;
	artistUrl?: string;
	views: number;
	uploadDate?: string;
	imageSrc?: string | null;
	image?: any | null;
	isPaused?: boolean;
	songDuration: number;
	elapsedSeconds?: number;
	url?: string;
	album?: string | null;
	videoId: string;
	playlistId?: string;
	mediaType: MediaType;
	tags?: string[];
}

enum MediaType {
	Audio = "AUDIO",
	OriginalMusicVideo = "ORIGINAL_MUSIC_VIDEO",
	UserGeneratedContent = "USER_GENERATED_CONTENT",
	PodcastEpisode = "PODCAST_EPISODE",
	OtherVideo = "OTHER_VIDEO",
}

export const handleWsMessages = (
	data: string,
	dispatch: Dispatch<UnknownAction>,
	{
		setSongInfo,
	}: {
		setSongInfo: ActionCreatorWithPayload<
			Partial<{
				albumArtUrl: string;
				artistName: string;
				elapsedTime: number;
				isPlaying: boolean;
				songLength: number;
				songName: string;
				videoUrl: string;
			}>,
			"musicplayerstate/setSongInfo"
		>;
	},
) => {
	try {
		const msgData:
			| IMsgTypePlayerInfo
			| IMsgTypePositionChanged
			| IMsgTypeVideoChanged
			| IMsgTypePlayerStateChanged
			| IMsgTypeVolumeChanged
			| IMsgTypeRepeatChanged
			| IMsgTypeShuffleChanged = JSON.parse(data);

		switch (msgData.type) {
			case "PLAYER_STATE_CHANGED":
				console.log(msgData);
				dispatch(
					setSongInfo({
						isPlaying: msgData.isPlaying,
						elapsedTime: msgData.position,
					}),
				);
				break;
			case "VIDEO_CHANGED":
				console.log(msgData);
				dispatch(
					setSongInfo({
						albumArtUrl: msgData.song.imageSrc ?? "",
						artistName: msgData.song.artist,
						elapsedTime: msgData.position,
						songLength: msgData.song.songDuration,
						songName: msgData.song.title,
						videoUrl: msgData.song.url,
					}),
				);
				break;
			case "PLAYER_INFO":
				console.log(msgData);
				dispatch(
					setSongInfo({
						albumArtUrl: msgData.song.imageSrc ?? "",
						artistName: msgData.song.artist,
						elapsedTime: msgData.position,
						isPlaying: msgData.isPlaying,
						songLength: msgData.song.songDuration,
						songName: msgData.song.title,
						videoUrl: msgData.song.url,
					}),
				);
				break;
			case "POSITION_CHANGED":
				// No logs, this gets updated every seconds
				dispatch(
					setSongInfo({
						elapsedTime: msgData.position,
					}),
				);
				break;
			case "VOLUME_CHANGED":
			case "REPEAT_CHANGED":
			case "SHUFFLE_CHANGED":
				break;
			default:
				console.log(msgData);
		}
	} catch (e) {
		console.log(data);
	}
};
