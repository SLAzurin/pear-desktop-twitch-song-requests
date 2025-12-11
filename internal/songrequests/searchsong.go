package songrequests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var pearDesktopHost = "127.0.0.1:26538"

func GetPearDesktopHost() string {
	return pearDesktopHost
}

type SongResult struct {
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	VideoID      string `json:"videoId"`
	RawTimeData  string `json:"-"`
	ImageUrl     string `json:"imageUrl"`
	SearchOrigin string `json:"-"`
}

type apiSearchSongResult struct {
	Contents struct {
		TabbedSearchResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Content struct {
						SectionListRenderer struct {
							Contents *[]struct {
								MusicShelfRenderer *struct {
									Contents []struct {
										MusicResponsiveListItemRenderer struct {
											Thumbnail struct {
												MusicThumbnailRenderer struct {
													Thumbnail struct {
														Thumbnails []struct {
															Url string `json:"url"`
														} `json:"thumbnails"`
													} `json:"thumbnail"`
												} `json:"musicThumbnailRenderer"`
											} `json:"thumbnail"`
											FlexColumns []struct {
												MusicResponsiveListItemFlexColumnRenderer struct {
													Text struct {
														Runs []struct {
															Text               string `json:"text"`
															NavigationEndpoint *struct {
																BrowseEndpoint *struct {
																	BrowseEndpointContextSupportedConfigs *struct {
																		BrowseEndpointContextMusicConfig *struct {
																			PageType string `json:"pageType"`
																		} `json:"browseEndpointContextMusicConfig"`
																	} `json:"browseEndpointContextSupportedConfigs"`
																} `json:"browseEndpoint"`
																WatchEndpoint *struct {
																	VideoId                            string `json:"videoId"`
																	WatchEndpointMusicSupportedConfigs *struct {
																		WatchEndpointMusicConfig *struct {
																			MusicVideoType string `json:"musicVideoType"`
																		} `json:"watchEndpointMusicConfig"`
																	} `json:"watchEndpointMusicSupportedConfigs"`
																} `json:"watchEndpoint"`
															} `json:"navigationEndpoint"`
														} `json:"runs"`
													} `json:"text"`
												} `json:"musicResponsiveListItemFlexColumnRenderer"`
											} `json:"flexColumns"`
											Overlay *struct {
												MusicItemThumbnailOverlayRenderer struct {
													Content struct {
														MusicPlayButtonRenderer struct {
															PlayNavigationEndpoint struct {
																WatchEndpoint *struct {
																	WatchEndpointMusicSupportedConfigs struct {
																		WatchEndpointMusicConfig struct {
																			MusicVideoType string `json:"musicVideoType"`
																		} `json:"watchEndpointMusicConfig"`
																	} `json:"watchEndpointMusicSupportedConfigs"`
																} `json:"watchEndpoint"`
															} `json:"playNavigationEndpoint"`
														} `json:"musicPlayButtonRenderer"`
													} `json:"content"`
												} `json:"musicItemThumbnailOverlayRenderer"`
											} `json:"overlay"`
										} `json:"musicResponsiveListItemRenderer"`
									} `json:"contents"`
								} `json:"musicShelfRenderer"`
								MusicCardShelfRenderer *struct {
									Thumbnail struct {
										MusicThumbnailRenderer struct {
											Thumbnail struct {
												Thumbnails []struct {
													Url string `json:"url"`
												} `json:"thumbnails"`
											} `json:"thumbnail"`
										} `json:"musicThumbnailRenderer"`
									} `json:"thumbnail"`
									Subtitle struct {
										Runs []struct {
											Text               string `json:"text"`
											NavigationEndpoint *struct {
												BrowseEndpoint *struct {
													BrowseEndpointContextSupportedConfigs *struct {
														BrowseEndpointContextMusicConfig *struct {
															PageType string `json:"pageType"`
														} `json:"browseEndpointContextMusicConfig"`
													} `json:"browseEndpointContextSupportedConfigs"`
												} `json:"browseEndpoint"`
											} `json:"navigationEndpoint"`
										} `json:"runs"`
									} `json:"subtitle"`
									Title struct {
										Runs []struct {
											Text               string `json:"text"`
											NavigationEndpoint *struct {
												WatchEndpoint *struct {
													VideoId string `json:"videoId"`
												} `json:"watchEndpoint"`
											} `json:"navigationEndpoint"`
										} `json:"runs"`
									} `json:"title"`
								} `json:"musicCardShelfRenderer"`
							} `json:"contents"`
						} `json:"sectionListRenderer"`
					} `json:"content"`
				} `json:"tabRenderer"`
			} `json:"tabs"`
		} `json:"tabbedSearchResultsRenderer"`
	} `json:"contents"`
}

const (
	MUSIC_VIDEO_TYPE_ATV             = "MUSIC_VIDEO_TYPE_ATV"
	MUSIC_VIDEO_TYPE_OMV             = "MUSIC_VIDEO_TYPE_OMV"
	MUSIC_VIDEO_TYPE_UGC             = "MUSIC_VIDEO_TYPE_UGC"
	MUSIC_VIDEO_TYPE_PODCAST_EPISODE = "MUSIC_VIDEO_TYPE_PODCAST_EPISODE"
	MUSIC_VIDEO_TYPE_OTHER_VIDEO     = "MUSIC_VIDEO_TYPE_OTHER_VIDEO"

	MUSIC_PAGE_TYPE_USER_CHANNEL = "MUSIC_PAGE_TYPE_USER_CHANNEL"
	MUSIC_PAGE_TYPE_ARTIST       = "MUSIC_PAGE_TYPE_ARTIST"
)

// make sure to sanitize url for music.youtube.com / youtu.be / youtube.com/watch?v=
func SearchSong(query string, minLength int, maxLength int) (*SongResult, error) {
	inBody := echo.Map{
		"query": strings.TrimSpace(query),
	}
	ib, _ := json.Marshal(inBody)
	resp, err := http.Post("http://"+pearDesktopHost+"/api/v1/search", "application/json", bytes.NewBuffer(ib))
	if err != nil {
		return nil, err
	}
	outBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResults := apiSearchSongResult{}
	err = json.Unmarshal(outBody, &rawResults)
	if err != nil {
		return nil, err
	}

	songResults := []SongResult{}

	// Start of contents inside search-songs.mts
	contents := rawResults.Contents.TabbedSearchResultsRenderer.Tabs
	for _, content := range contents {
		tabContent := content.TabRenderer.Content.SectionListRenderer.Contents
		if tabContent == nil {
			continue
		}
		videoId := ""
		for _, content := range *tabContent {
			if content.MusicCardShelfRenderer != nil {
				// This is the main "pushed" result from yt, usually the more popular click
				// Not always a video or music
				title := ""
				artistOrUploader := ""
				var validRun *struct {
					Text               string `json:"text"`
					NavigationEndpoint *struct {
						WatchEndpoint *struct {
							VideoId string `json:"videoId"`
						} `json:"watchEndpoint"`
					} `json:"navigationEndpoint"`
				} = nil
				// const validRun = content.musicCardShelfRenderer?.title.runs.find
				for _, v := range content.MusicCardShelfRenderer.Title.Runs {
					if v.NavigationEndpoint.WatchEndpoint != nil {
						validRun = &v
						break
					}
				}
				// end find
				if validRun == nil {
					continue
				}
				var artistData *struct {
					Text               string `json:"text"`
					NavigationEndpoint *struct {
						BrowseEndpoint *struct {
							BrowseEndpointContextSupportedConfigs *struct {
								BrowseEndpointContextMusicConfig *struct {
									PageType string "json:\"pageType\""
								} "json:\"browseEndpointContextMusicConfig\""
							} "json:\"browseEndpointContextSupportedConfigs\""
						} "json:\"browseEndpoint\""
					} "json:\"navigationEndpoint\""
				} = nil
				// const artistData = content.musicCardShelfRenderer?.subtitle.runs.find
				for _, v := range content.MusicCardShelfRenderer.Subtitle.Runs {
					thisPageType := ""
					if v.NavigationEndpoint != nil {
						if v.NavigationEndpoint.BrowseEndpoint != nil {
							if v.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs != nil {
								if v.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig != nil {
									thisPageType = v.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig.PageType
								}
							}
						}
					}
					if thisPageType == MUSIC_PAGE_TYPE_ARTIST || thisPageType == MUSIC_PAGE_TYPE_USER_CHANNEL {
						artistData = &v
						break
					}
				}
				// end find
				if artistData != nil {
					artistOrUploader = artistData.Text
				}
				videoId = validRun.NavigationEndpoint.WatchEndpoint.VideoId
				title = validRun.Text

				timeData := content.MusicCardShelfRenderer.Subtitle.Runs[len(content.MusicCardShelfRenderer.Subtitle.Runs)-1].Text

				songResults = append(songResults, SongResult{
					Title:        title,
					Artist:       artistOrUploader,
					VideoID:      videoId,
					RawTimeData:  timeData,
					ImageUrl:     content.MusicCardShelfRenderer.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails[0].Url,
					SearchOrigin: "MusicCardShelfRenderer",
				})
			}

			if content.MusicShelfRenderer != nil {
				// This is the list of other results below promoted result
				contents := content.MusicShelfRenderer.Contents
				for _, content := range contents {
					mediaTitle := ""
					videoId := ""
					artistOrUploader := ""
					mediaType := ""
					timeData := ""
					imageUrl := ""

					if content.MusicResponsiveListItemRenderer.Overlay != nil {
						if content.MusicResponsiveListItemRenderer.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer.PlayNavigationEndpoint.WatchEndpoint != nil {
							mediaType = content.MusicResponsiveListItemRenderer.Overlay.MusicItemThumbnailOverlayRenderer.Content.MusicPlayButtonRenderer.PlayNavigationEndpoint.WatchEndpoint.WatchEndpointMusicSupportedConfigs.WatchEndpointMusicConfig.MusicVideoType
							imageUrl = content.MusicResponsiveListItemRenderer.Thumbnail.MusicThumbnailRenderer.Thumbnail.Thumbnails[0].Url
						}
					}

					switch mediaType {
					case MUSIC_VIDEO_TYPE_ATV:
						fallthrough
					case MUSIC_VIDEO_TYPE_OMV:
						fallthrough
					case MUSIC_VIDEO_TYPE_UGC:
						// do nothing, is valid
					case MUSIC_VIDEO_TYPE_OTHER_VIDEO:
						fallthrough
					case MUSIC_VIDEO_TYPE_PODCAST_EPISODE:
						// unsupported video type
						continue
					default:
						mediaType = ""
					}

					if mediaType == "" {
						continue
					}

					for _, flexColumn := range content.MusicResponsiveListItemRenderer.FlexColumns {
						for _, run := range flexColumn.MusicResponsiveListItemFlexColumnRenderer.Text.Runs {
							compareMusicVideoType := ""
							if run.NavigationEndpoint != nil {
								if run.NavigationEndpoint.WatchEndpoint != nil {
									if run.NavigationEndpoint.WatchEndpoint.WatchEndpointMusicSupportedConfigs != nil {
										if run.NavigationEndpoint.WatchEndpoint.WatchEndpointMusicSupportedConfigs.WatchEndpointMusicConfig != nil {
											compareMusicVideoType = run.NavigationEndpoint.WatchEndpoint.WatchEndpointMusicSupportedConfigs.WatchEndpointMusicConfig.MusicVideoType
										}
									}
								}
							}
							if compareMusicVideoType == MUSIC_VIDEO_TYPE_ATV || compareMusicVideoType == MUSIC_VIDEO_TYPE_OMV || compareMusicVideoType == MUSIC_VIDEO_TYPE_UGC {
								mediaTitle = run.Text
								videoId = run.NavigationEndpoint.WatchEndpoint.VideoId
							}
							compareMusicPageType := ""
							if run.NavigationEndpoint != nil {
								if run.NavigationEndpoint.BrowseEndpoint != nil {
									if run.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs != nil {
										if run.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig != nil {
											compareMusicPageType = run.NavigationEndpoint.BrowseEndpoint.BrowseEndpointContextSupportedConfigs.BrowseEndpointContextMusicConfig.PageType
										}
									}
								}
							}
							expectedMusicPageType := ""
							if compareMusicPageType != "" {
								if mediaType == MUSIC_VIDEO_TYPE_UGC {
									expectedMusicPageType = MUSIC_PAGE_TYPE_USER_CHANNEL
								} else {
									expectedMusicPageType = MUSIC_PAGE_TYPE_ARTIST
								}
								if compareMusicPageType == expectedMusicPageType {
									artistOrUploader = run.Text
									timeData = flexColumn.MusicResponsiveListItemFlexColumnRenderer.Text.Runs[len(flexColumn.MusicResponsiveListItemFlexColumnRenderer.Text.Runs)-1].Text
								}
							}
						}
					}

					songResults = append(songResults, SongResult{
						Title:        mediaTitle,
						Artist:       artistOrUploader,
						VideoID:      videoId,
						RawTimeData:  timeData,
						ImageUrl:     imageUrl,
						SearchOrigin: "MusicShelfRenderer",
					})
				}
			}
		}
	}

	// end of search logic from ts port
	var selectedSong *SongResult = nil
	for _, v := range songResults {
		if v.VideoID == query {
			selectedSong = &v
			break
		}
	}

	if selectedSong == nil && len(songResults) > 0 {
		selectedSong = &songResults[0]
	}

	if selectedSong != nil {
		// "1:00:04" or "10:00" or "1:00" or err
		// err usually means its safe
		isValid := validateTime(selectedSong.RawTimeData, minLength, maxLength)
		if !isValid {
			return nil, errors.New("search songs: song duration exceeds max allowed")
		}
		return selectedSong, nil
	}

	return nil, errors.New("search songs: no results")
}

func validateTime(s string, min, max int) bool {
	template := "2000-01-01 00:00:00"
	if len(s) >= len(template) {
		// Loose return when error occurs
		return true
	}
	s = template[0:len(template)-len(s)] + s
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		// Loose return when error occurs
		return true
	}
	tmax, _ := time.Parse(time.DateTime, template)
	tmax = tmax.Add(time.Duration(max) * time.Second)
	tmin, _ := time.Parse(time.DateTime, template)
	tmin = tmin.Add(time.Duration(min) * time.Second)
	return !(t.After(tmax) || t.Before(tmin))
}
