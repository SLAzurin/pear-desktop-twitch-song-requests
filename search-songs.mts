import something from './response3.json' with { type: 'json' };

type ValueOf<T> = T[keyof T];
const EMUSIC_VIDEO_TYPE = {
    ATV: 'MUSIC_VIDEO_TYPE_ATV', // Uusally Album Music 
    OMV: 'MUSIC_VIDEO_TYPE_OMV', // Usually Original Music Video
    UGC: 'MUSIC_VIDEO_TYPE_UGC', // Usually User Generated Content
} as const;

type TMusicVideoType = ValueOf<typeof EMUSIC_VIDEO_TYPE>;

const EMUSIC_PAGE_TYPE = {
    USER_CHANNEL: 'MUSIC_PAGE_TYPE_USER_CHANNEL',
    ARTIST: 'MUSIC_PAGE_TYPE_ARTIST',
} as const;


const contents = something.contents.tabbedSearchResultsRenderer.tabs;

for (const tab of contents) {
    const tabContent = tab.tabRenderer.content.sectionListRenderer.contents;
    if (!tabContent) continue;
    let videoId: string = '';
    for (const content of tabContent) {
        if (content.musicCardShelfRenderer) {
            // This is the main "pushed" result from yt, usually the more popular click
            // Not always a video or music
            let title: string | null = null;
            let artistOrUploader: string | null = null;
            const validRun = content.musicCardShelfRenderer?.title.runs.find((v) => {
                if ((v.navigationEndpoint as any).watchEndpoint) {
                    return true;
                }
            });
            if (!validRun) continue;
            const artistData = content.musicCardShelfRenderer?.subtitle.runs.find((v) => {
                if (v.navigationEndpoint?.browseEndpoint?.browseEndpointContextSupportedConfigs?.browseEndpointContextMusicConfig?.pageType === EMUSIC_PAGE_TYPE.ARTIST || v.navigationEndpoint?.browseEndpoint?.browseEndpointContextSupportedConfigs?.browseEndpointContextMusicConfig?.pageType === EMUSIC_PAGE_TYPE.USER_CHANNEL) {
                    return true;
                }
            });
            artistOrUploader = artistData ? artistData.text : null;
            videoId = (validRun as any).navigationEndpoint.watchEndpoint.videoId;
            title = validRun.text;
            if (title) console.log(`${title} - ${artistOrUploader} = ${videoId}`);
        }

        if (content.musicShelfRenderer) {
            // This is the list of other results
            const { contents } = content.musicShelfRenderer;
            for (const content of contents) {
                let mediaTitle = '';
                let videoId = '';
                let artistOrUploader = '';
                let mediaType: TMusicVideoType | null = null;

                if (content.musicResponsiveListItemRenderer.overlay?.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint?.watchEndpointMusicSupportedConfigs.watchEndpointMusicConfig.musicVideoType === EMUSIC_VIDEO_TYPE.ATV) mediaType = EMUSIC_VIDEO_TYPE.ATV;
                if (content.musicResponsiveListItemRenderer.overlay?.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint?.watchEndpointMusicSupportedConfigs.watchEndpointMusicConfig.musicVideoType === EMUSIC_VIDEO_TYPE.UGC) mediaType = EMUSIC_VIDEO_TYPE.UGC;
                if (content.musicResponsiveListItemRenderer.overlay?.musicItemThumbnailOverlayRenderer.content.musicPlayButtonRenderer.playNavigationEndpoint.watchEndpoint?.watchEndpointMusicSupportedConfigs.watchEndpointMusicConfig.musicVideoType === EMUSIC_VIDEO_TYPE.OMV) mediaType = EMUSIC_VIDEO_TYPE.OMV;
                if (!mediaType) {
                    continue;
                }

                // get media title and artist / uploader
                for (const flexColumn of content.musicResponsiveListItemRenderer.flexColumns) {
                    for (const run of flexColumn.musicResponsiveListItemFlexColumnRenderer.text.runs) {
                        // get title
                        if ((run as any).navigationEndpoint?.watchEndpoint?.watchEndpointMusicSupportedConfigs?.watchEndpointMusicConfig?.musicVideoType === EMUSIC_VIDEO_TYPE.ATV || (run as any).navigationEndpoint?.watchEndpoint?.watchEndpointMusicSupportedConfigs?.watchEndpointMusicConfig?.musicVideoType === EMUSIC_VIDEO_TYPE.UGC || (run as any).navigationEndpoint?.watchEndpoint?.watchEndpointMusicSupportedConfigs?.watchEndpointMusicConfig?.musicVideoType === EMUSIC_VIDEO_TYPE.OMV) {
                            // This is the title text
                            mediaTitle = run.text;
                            videoId = (run as any).navigationEndpoint?.watchEndpoint?.videoId;
                        }
                        if ((run as any).navigationEndpoint?.browseEndpoint?.browseEndpointContextSupportedConfigs?.browseEndpointContextMusicConfig?.pageType === (mediaType === EMUSIC_VIDEO_TYPE.UGC ? EMUSIC_PAGE_TYPE.USER_CHANNEL : EMUSIC_PAGE_TYPE.ARTIST)) {
                            // channel name
                            artistOrUploader = run.text;
                        }
                    }
                }
                console.log(`${mediaTitle} - ${artistOrUploader} = ${videoId}`);
            }
        }
    }
}