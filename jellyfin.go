package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// API definitions: https://swagger.emby.media/ & https://api.jellyfin.org/
// Docs: https://github.com/mediabrowser/emby/wiki

func addJellyfinHandlers(s *mux.Router) {
	r := s.UseEncodedPath()

	r.HandleFunc("/System/Info/Public", systemInfoHandler)

	r.HandleFunc("/Users/AuthenticateByName", authenticateByNameHandler).Methods("POST")
	r.HandleFunc("/DisplayPreferences/usersettings", displayPreferencesHandler)

	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5", usersHandler)
	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/Views", userViewsHandler)
	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/GroupingOptions", usersGroupingOptionsHandler)

	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items", usersItemsHandler)
	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/Latest", usersItemsLatestHandler)
	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/{item}", usersItemHandler)
	r.HandleFunc("/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/Resume", usersItemsResumeHandler)

	r.HandleFunc("/Library/VirtualFolders", libraryVirtualFoldersHandler)
	r.HandleFunc("/Shows/NextUp", showsNextUpHandler)
	r.HandleFunc("/Shows/{show}/Seasons", showsSeasonsHandler)
	r.HandleFunc("/Shows/{show}/Episodes", showsEpisodesHandler)

	r.HandleFunc("/Items/{item}/Images/{type}", itemsImagesHandler)
	r.HandleFunc("/Items/{item}/PlaybackInfo", itemsPlaybackInfoHandler)
	r.HandleFunc("/MediaSegments/{item}", itemsMediaSegmentsHandler)
	r.HandleFunc("/Videos/{item}/stream", videoStreamHandler)

	r.HandleFunc("/Persons", personsHandler)

	r.HandleFunc("/Sessions/Playing", sessionsPlayingHandler).Methods("POST")
	r.HandleFunc("/Sessions/Playing/Progress", sessionsPlayingHandler).Methods("POST")

}

const (
	serverID             = "2b11644442754f02a0c1e45d2a9f5c71"
	userID               = "2b1ec0a52b09456c9823a367d84ac9e5"
	collectionRootID     = "e9d5075a555c1cbc394eec4cef295274"
	displayPreferencesID = "f137a2dd21bbc1b99aa5c0f6bf02a805"

	// item prefixes
	itemprefix_collection = "collection_"
	itemprefix_show       = "show_"
	itemprefix_episode    = "episode_"

	// imagetag starting this will get HTTP-redirected
	tagprefix_redirect = "redirect_"
	// imagetag starting with file_ means we'll serve the filename from local disk
	tagprefix_file = "file_"
)

var loggedInUser = JFUser{
	Name:                      "erik",
	ServerId:                  serverID,
	Id:                        userID,
	HasPassword:               true,
	HasConfiguredPassword:     true,
	HasConfiguredEasyPassword: false,
	EnableAutoLogin:           false,
	LastLoginDate:             time.Now(),
	LastActivityDate:          time.Now(),
}

// curl -v http://127.0.0.1:9090/System/Info/Public

func systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	response := JFSystemInfoResponse{
		Id:           serverID,
		LocalAddress: "http://192.168.1.223:9090",
		// Jellyfin native client checks for exact productname :facepalm:
		// https://github.com/jellyfin/jellyfin-expo/blob/7dedbc72fb53fc4b83c3967c9a8c6c071916425b/utils/ServerValidator.js#L82C49-L82C64
		ProductName: "Jellyfin Server",
		ServerName:  "jellyfin",
		Version:     "10.10.3",
	}
	serveJSON(response, w)
}

// curl -v -X POST http://127.0.0.1:9090/Users/AuthenticateByName

// Authenticates a user by name.
// (POST /Users/AuthenticateByName)
func authenticateByNameHandler(w http.ResponseWriter, r *http.Request) {
	response := JFAuthenticateByNameResponse{
		User: loggedInUser,
		SessionInfo: JFSessionInfo{
			RemoteEndPoint:     "192.168.1.223",
			Id:                 "e3a869b7a901f8894de8ee65688db6c0",
			UserId:             loggedInUser.Id,
			UserName:           loggedInUser.Name,
			Client:             "Infuse-Direct",
			LastActivityDate:   time.Now(),
			DeviceName:         "Apple TV",
			DeviceId:           "F3913A92-6378-48FF-A862-1EFB91C13355",
			ApplicationVersion: "8.0",
			IsActive:           true,
		},
		AccessToken: "83a6cca4f70f419288bc9f42ba7fa18c",
		ServerId:    serverID,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/DisplayPreferences/usersettings?userId=2b1ec0a52b09456c9823a367d84ac9e5&client=emby'

func displayPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	serveJSON(DisplayPreferencesResponse{
		ID:                 "3ce5b65d-e116-d731-65d1-efc4a30ec35c",
		SortBy:             "SortName",
		RememberIndexing:   false,
		PrimaryImageHeight: 250,
		PrimaryImageWidth:  250,
		CustomPrefs: DisplayPreferencesCustomPrefs{
			ChromecastVersion:          "stable",
			SkipForwardLength:          "30000",
			SkipBackLength:             "10000",
			EnableNextVideoInfoOverlay: "False",
			Tvhome:                     "null",
			DashboardTheme:             "null",
		},
		ScrollDirection: "Horizontal",
		ShowBackdrop:    true,
		RememberSorting: false,
		SortOrder:       "Ascending",
		ShowSidebar:     false,
		Client:          "emby",
	}, w)
}

// curl -v http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5

func usersHandler(w http.ResponseWriter, r *http.Request) {
	serveJSON(loggedInUser, w)
}

// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Views?IncludeExternalContent=false'

func userViewsHandler(w http.ResponseWriter, r *http.Request) {
	items := []JFItem{}
	var itemcount int

	for _, c := range config.Collections {
		itemID := genCollectionID(c.SourceId)

		// Root item
		item := JFItem{
			ServerID:                 serverID,
			ParentID:                 collectionRootID,
			Type:                     "CollectionFolder",
			IsFolder:                 true,
			DateCreated:              "2020-01-01T00:00:00.0000000Z",
			PremiereDate:             "2020-01-01T00:00:00.0000000Z",
			Name:                     c.Name_,
			SortName:                 c.Name_,
			ID:                       itemID,
			Etag:                     idHash(itemID),
			CanDelete:                false,
			CanDownload:              false,
			EnableMediaSourceDisplay: true,
			PlayAccess:               "Full",
			RemoteTrailers:           []JFRemoteTrailers{},
			LocalTrailerCount:        0,
			ChildCount:               len(c.Items),
			SpecialFeatureCount:      0,
			DisplayPreferencesID:     displayPreferencesID,
			LocationType:             "Remote",
			Path:                     "/collection",
			ExternalUrls:             []JFExternalUrls{},
			MediaType:                "Unknown",
			LockData:                 false,
		}

		switch c.Type {
		case "movies":
			item.CollectionType = "movies"
		case "shows":
			item.CollectionType = "tvshows"
		}

		items = append(items, item)
		itemcount++
	}

	response := JFUserViewsResponse{
		Items:            items,
		TotalRecordCount: itemcount,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/GroupingOptions

func usersGroupingOptionsHandler(w http.ResponseWriter, r *http.Request) {
	collections := []JFCollection{}
	for _, c := range config.Collections {
		collection := JFCollection{
			Name: c.Name_,
			ID:   genCollectionID(c.SourceId),
		}
		collections = append(collections, collection)
	}
	serveJSON(collections, w)
}

// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/Resume?Limit=12&MediaTypes=Video&Recursive=true&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount'

func usersItemsResumeHandler(w http.ResponseWriter, r *http.Request) {
	response := JFUsersItemsResumeResponse{
		Items:            []string{},
		TotalRecordCount: 0,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/f137a2dd21bbc1b99aa5c0f6bf02a805?Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount'
// handle individual item: a movie/tv collection or individual file
func usersItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId := vars["item"]

	// Is collection?
	if strings.HasPrefix(itemId, itemprefix_collection) {
		collectionItem, err := buildJFItemCollection(itemId)
		if err != nil {
			http.Error(w, "Could not find collection", http.StatusNotFound)
			return

		}
		serveJSON(collectionItem, w)
		return
	}

	// Is episode?
	if strings.HasPrefix(itemId, itemprefix_episode) {
		episodeItem, err := buildJFItemEpisode(itemId)
		if err != nil {
			http.Error(w, "Could not find episode", http.StatusNotFound)
			return
		}
		serveJSON(episodeItem, w)
		return
	}

	if strings.Contains(itemId, "_") {
		log.Print("Item request for unknown prefix!")
		http.Error(w, "Unknown item type", http.StatusInternalServerError)
		return
	}

	// Try to find individual item
	c, i := getItemByID(itemId)
	if i == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	serveJSON(buildJFItem(c, i), w)
}

func buildJFItemCollection(itemid string) (response JFItem, e error) {
	if !strings.HasPrefix(itemid, itemprefix_collection) {
		e = errors.New("malformed collection id")
		return
	}

	collectionid := strings.TrimPrefix(itemid, itemprefix_collection)
	c := getCollection(collectionid)
	if c == nil {
		e = errors.New("collection not found")
		return
	}

	itemID := genCollectionID(c.SourceId)
	response = JFItem{
		Name:                     c.Name_,
		ServerID:                 serverID,
		ID:                       itemID,
		Etag:                     idHash(itemID),
		DateCreated:              "2020-01-01T00:00:00.0000000Z",
		Type:                     "CollectionFolder",
		IsFolder:                 true,
		EnableMediaSourceDisplay: true,
		ChildCount:               len(c.Items),
		DisplayPreferencesID:     displayPreferencesID,
		ExternalUrls:             []JFExternalUrls{},
		PlayAccess:               "Full",
		PrimaryImageAspectRatio:  1.7777777777777777,
		RemoteTrailers:           []JFRemoteTrailers{},
		Path:                     "/collection",
		LocationType:             "FileSystem",
		LockData:                 false,
		MediaType:                "Unknown",
		ParentID:                 "e9d5075a555c1cbc394eec4cef295274",
		CanDelete:                false,
		CanDownload:              false,
		SpecialFeatureCount:      0,
	}
	switch c.Type {
	case "movies":
		response.CollectionType = "movies"
	case "shows":
		response.CollectionType = "tvshows"
	}
	response.SortName = response.CollectionType
	return
}

// buildJFItem builds movie or show (from local file)
func buildJFItem(c *Collection, i *Item) (response JFItem) {
	filename := c.Directory + "/" + i.Name + "/" + i.Video
	return buildJFItemFile(c, i, filename)
}

// builds JFItem of local file
func buildJFItemFile(c *Collection, i *Item, videoFilename string) (response JFItem) {
	file, err := os.Open(videoFilename)
	if err != nil {
		return
	}
	defer file.Close()
	fileStat, err := file.Stat()
	if err != nil {
		return
	}
	file, err = os.Open(i.NfoPath)
	if err == nil {
		i.Nfo = decodeNfo(file)
		file.Close()
	}

	response = JFItem{
		Name:                    i.Name,
		OriginalTitle:           i.Name,
		SortName:                i.Name,
		ForcedSortName:          i.Name,
		ServerID:                serverID,
		ParentID:                idHash(c.Name_),
		ID:                      i.Id,
		Etag:                    idHash(i.Id),
		DateCreated:             fileStat.ModTime().UTC().Format("2001-01-01T00:00:00.0000000Z"),
		PrimaryImageAspectRatio: 0.6666666666666666,
		MediaStreams: []JFMediaStreams{
			{
				Codec:              "h264",
				CodecTag:           "avc1",
				Language:           "eng",
				TimeBase:           "1/16000",
				VideoRange:         "SDR",
				VideoRangeType:     "SDR",
				AudioSpatialFormat: "None",
				DisplayTitle:       "720p H264 SDR",
				IsInterlaced:       false,
				Height:             546,
				Width:              1280,
			},
		},
		ImageTags: JFImageTags{
			Primary: "primary_" + i.Id,
		},
	}

	if c.Type == "movies" {
		response.Type = "Movie"
		response.IsFolder = false
		response.LocationType = "FileSystem"
		response.VideoType = "VideoFile"
		response.Path = "file.mp4"
		response.Container = "mov,mp4,m4a"
		response.MediaSources = buildMediaSource(videoFilename, i.Nfo)

	}

	// Is this a Season item?
	if c.Type == "shows" && i.Seasons != nil {
		response.Type = "Series"
		response.IsFolder = true
		response.ChildCount = len(i.Seasons)
		response.MediaSources = nil
		response.MediaStreams = nil
		// Required to have Infuse load backdrop of episode
		response.BackdropImageTags = []string{
			response.ID,
		}
	}

	enrichResponseWithNFO(&response, i.Nfo)

	return response
}

// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items?ExcludeLocationTypes=Virtual&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&ParentId=f137a2dd21bbc1b99aa5c0f6bf02a805&SortBy=SortName,ProductionYear&SortOrder=Ascending&IncludeItemTypes=Movie&Recursive=true&StartIndex=0&Limit=50'
// find based upon title
// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items?ExcludeLocationTypes=Virtual&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&SearchTerm=p&Recursive=true&Limit=24

// generate list of items based upon provided ParentId or a text searchTerm
func usersItemsHandler(w http.ResponseWriter, r *http.Request) {
	queryparams := r.URL.Query()

	// collection id provided?
	var c *Collection
	collectionid, err := getCollectionID(queryparams.Get("ParentId"))

	// FIXME: if searchTerm provided search in collection "2" (TV)
	searchTerm := queryparams.Get("SearchTerm")
	if searchTerm != "" {
		collectionid = "2"
		err = nil
	}

	if collectionid == "" {
		// todo: this could be a search by person :)
		http.Error(w, "Collection not found", http.StatusNotFound)
		return
	}

	if err == nil {
		c = getCollection(collectionid)
		if c == nil {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
	}
	var items []*Item
	for _, i := range c.Items {
		// Was a collectionId provided?
		if collectionid != "" {
			// log.Printf("provided collection: %s, searching in collection: %+v\n", collectionid, c)
			if c.SourceId == 0 {
				break
			}
		}
		if searchTerm == "" || strings.Contains(strings.ToLower(i.Name), strings.ToLower(searchTerm)) {
			// fix: sortname should be set at the source in Item
			if i.SortName == "" {
				i.SortName = i.Name
			}
			items = append(items, i)
		}
	}

	// Apply sorting if SortBy is provided
	sortBy := queryparams.Get("SortBy")
	if sortBy != "" {
		sortFields := strings.Split(sortBy, ",")
		sort.SliceStable(items, func(i, j int) bool {
			sortOrder := queryparams.Get("SortOrder")
			for _, field := range sortFields {
				switch strings.ToLower(field) {
				case "sortname":
					if items[i].SortName != items[j].SortName {
						if sortOrder == "Descending" {
							return items[i].SortName > items[j].SortName
						}
						return items[i].SortName < items[j].SortName
					}
				case "productionyear":
					if items[i].Year != items[j].Year {
						if sortOrder == "Descending" {
							return items[i].Year > items[j].Year
						}
						return items[i].Year < items[j].Year
					}
				case "criticrating":
					if items[i].Rating != items[j].Rating {
						if sortOrder == "Descending" {
							return items[i].Rating > items[j].Rating
						}
						return items[i].Rating < items[j].Rating
					}
				}
			}
			return false
		})
	}

	// Apply pagination (startIndex and limit)
	startIndex, _ := strconv.Atoi(queryparams.Get("StartIndex"))
	limit, _ := strconv.Atoi(queryparams.Get("Limit"))
	if startIndex >= 0 && startIndex < len(items) {
		items = items[startIndex:]
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}

	// Create API response
	responseItems := []JFItem{}
	for _, i := range items {
		responseItems = append(responseItems, buildJFItem(c, i))
	}
	response := UserItemsResponse{
		Items: responseItems,
		// total count in collection, not count in returned page
		TotalRecordCount: len(c.Items),
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/Users/2b1ec0a52b09456c9823a367d84ac9e5/Items/Latest?Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&ParentId=f137a2dd21bbc1b99aa5c0f6bf02a805&StartIndex=0&Limit=20'

func usersItemsLatestHandler(w http.ResponseWriter, r *http.Request) {
	c1, i1 := getItemByID("rVFG3EzPthk2wowNkqUl")
	c2, i2 := getItemByID("q2e2UzCOd9zkmJenIOph")
	items := []JFItem{
		buildJFItem(c1, i1),
		buildJFItem(c2, i2),
	}
	serveJSON(items, w)
}

// curl -v http://127.0.0.1:9090/Library/VirtualFolders
func libraryVirtualFoldersHandler(w http.ResponseWriter, r *http.Request) {
	libraries := []JFMediaLibrary{}
	for _, c := range config.Collections {
		itemId := genCollectionID(c.SourceId)
		l := JFMediaLibrary{
			Name:               c.Name_,
			ItemId:             itemId,
			PrimaryImageItemId: itemId,
			Locations:          []string{"/"},
		}
		switch c.Type {
		case "movies":
			l.CollectionType = "movies"
		case "shows":
			l.CollectionType = "tvshows"
		}
		libraries = append(libraries, l)
	}
	serveJSON(libraries, w)
}

// curl -v 'http://127.0.0.1:9090/Shows/4QBdg3S803G190AgFrBf/Seasons?UserId=2b1ec0a52b09456c9823a367d84ac9e5&ExcludeLocationTypes=Virtual&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount'
// generate season overview
func showsSeasonsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	showId := vars["show"]
	c, i := getItemByID(showId)
	if i == nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}
	showItem := buildJFItem(c, i)

	// Create API response
	seasons := []JFItem{}
	for _, s := range i.Seasons {
		seasonId := showItem.ID + "/" + fmt.Sprintf("%d", s.SeasonNo)
		season := JFItem{
			Type:               "Season",
			ServerID:           serverID,
			ParentID:           showId,
			SeriesID:           showId,
			ID:                 seasonId,
			Etag:               idHash(seasonId),
			SeriesName:         showItem.Name,
			IndexNumber:        s.SeasonNo,
			Name:               fmt.Sprintf("Season %d", s.SeasonNo),
			SortName:           fmt.Sprintf("%04d", s.SeasonNo),
			IsFolder:           true,
			LocationType:       "FileSystem",
			MediaType:          "Unknown",
			ChildCount:         len(s.Episodes),
			RecursiveItemCount: len(s.Episodes),
			ImageTags: JFImageTags{
				Primary: "season",
				// Backdrop: "season2",
			},
			DateCreated:    "2022-01-01T00:00:00.0000000Z",
			PremiereDate:   "2022-01-01T00:00:00.0000000Z",
			ProductionYear: 2022,
		}
		// season.UserData.LastPlayedDate = time.Now().UTC()

		seasons = append(seasons, season)
	}
	response := UserItemsResponse{
		Items:            seasons,
		TotalRecordCount: len(seasons),
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/Shows/rXlq4EHNxq4HIVQzw3o2/Episodes?UserId=2b1ec0a52b09456c9823a367d84ac9e5&ExcludeLocationTypes=Virtual&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&SeasonId=rXlq4EHNxq4HIVQzw3o2/1'
// generate show overview for one season
func showsEpisodesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	showId := vars["show"]
	c, i := getItemByID(showId)
	if i == nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}
	showItem := buildJFItem(c, i)

	// For which season do we need to produce list of shows?
	queryparams := r.URL.Query()
	seasonId := queryparams.Get("SeasonId")
	if seasonId == "" {
		http.Error(w, "SeasonID not provided", http.StatusNotFound)
		return
	}
	var requestedSeasonId string
	var requestedSeason int
	seasonIdParts := strings.Split(seasonId, "/")
	if len(seasonIdParts) == 2 {
		requestedSeasonId = seasonIdParts[0]
		requestedSeason, _ = strconv.Atoi(seasonIdParts[1])

		if requestedSeasonId != showItem.ID {
			http.Error(w, "Requested session does not belong to series", http.StatusNotFound)
			return
		}
	}

	// Create API response for requested season
	episodes := []JFItem{}
	for _, s := range i.Seasons {
		// log.Printf("Found season: %d, requested: %d\n", s.SeasonNo, requestedSeason)
		if s.SeasonNo != requestedSeason {
			continue
		}
		// log.Printf("Building season %d overview\n", s.SeasonNo)
		for _, e := range s.Episodes {
			episodeId := itemprefix_episode + url.QueryEscape(fmt.Sprintf("%s_%s/S%02d/%s", showId, i.Name, s.SeasonNo, e.BaseName))
			episode, err := buildJFItemEpisode(episodeId)
			if err != nil {
				log.Printf("buildJFItemEpisode returned error %s", err)
				continue
			}
			episodes = append(episodes, episode)
		}
	}
	response := UserItemsResponse{
		Items:            episodes,
		TotalRecordCount: len(episodes),
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// buildJFItemEpisode builds tv episode
func buildJFItemEpisode(episodeid string) (response JFItem, e error) {
	showId, episodebasepath, err := getEpisodeIDDetails(episodeid)
	if err != nil {
		e = errors.New("could not parse episodeid")
		return
	}

	c, showItem := getItemByID(showId)
	if showItem == nil {
		e = errors.New("could not find showid")
		return
	}

	var episodeNfo *Nfo
	nfofile := c.Directory + "/" + episodebasepath + ".nfo"
	file, err := os.Open(nfofile)
	if err == nil {
		episodeNfo = decodeNfo(file)
		file.Close()
	}

	filename := c.Directory + "/" + episodebasepath + ".mp4"
	response = JFItem{
		Type:       "Episode",
		IsFolder:   false,
		ServerID:   serverID,
		ID:         episodeid,
		Etag:       idHash(episodeid),
		Path:       "episode.mp4",
		SeriesName: showItem.Name,
		SeriesID:   idHash(showItem.Name),
		SeasonName: "Season " + episodeNfo.Season,
		Name:       episodeNfo.Title,
		// fixme:
		SortName:     fmt.Sprintf("%03s - %04s - %s", episodeNfo.Season, episodeNfo.Episode, episodeNfo.Title),
		Overview:     episodeNfo.Plot,
		LocationType: "FileSystem",
		MediaType:    "Video",
		VideoType:    "VideoFile",
		Container:    "mov,mp4,m4a",
		HasSubtitles: true,
		MediaSources: buildMediaSource(filename, episodeNfo),
		ImageTags: JFImageTags{
			Primary: "episode",
		},
		DateCreated: "2023-01-01T00:00:00.0000000Z",
	}
	// Get a bunch of metadata from series-level nfo
	if showItem.Nfo != nil {
		enrichResponseWithNFO(&response, showItem.Nfo)
	}

	// Remove ratings as we do not want ratings from series apply to an episode
	response.OfficialRating = ""
	response.CommunityRating = 0

	// Enrich and override metadata from episode nfo (more specific)
	enrichResponseWithNFO(&response, episodeNfo)
	return response, nil
}

func enrichResponseWithNFO(response *JFItem, n *Nfo) {
	if n.Season != "" {
		response.ParentIndexNumber, _ = strconv.Atoi(n.Season)
	}
	if n.Episode != "" {
		response.IndexNumber, _ = strconv.Atoi(n.Episode)
	}

	if n.Plot != "" {
		response.Overview = n.Plot
	}

	if n.Tagline != "" {
		response.Taglines = []string{n.Tagline}
	}

	// TV-14
	if n.Mpaa != "" {
		response.OfficialRating = n.Mpaa
	}

	// ProviderIds: JFProviderIds{
	// 	Tmdb:           "9659",
	// 	Imdb:           "tt0079501",
	// 	TmdbCollection: "8945",
	// },

	if n.Rating != 0 {
		response.CommunityRating = math.Round(float64(n.Rating)*10) / 10
	}

	if len(n.Genre) != 0 {
		// fixme: why do we duplicate both fields?
		response.Genres = n.Genre
		for _, genre := range n.Genre {
			g := JFGenreItems{
				Name: genre,
				ID:   idHash(genre),
			}
			response.GenreItems = append(response.GenreItems, g)
		}
	}

	if n.Studio != "" {
		response.Studios = []JFStudios{
			{
				Name: n.Studio,
				ID:   idHash(n.Studio),
			},
		}
	}

	// if n.Actor != nil {
	// 	for _, actor := range n.Actor {
	// 		p := JFPeople{
	// 			Type: "Actor",
	// 			Name: actor.Name,
	// 			ID:   idHash(actor.Name),
	// 		}
	// 		if actor.Thumb != "" {
	// 			p.PrimaryImageTag = tagprefix_redirect + actor.Thumb
	// 		}
	// 		response.People = append(response.People, p)
	// 	}
	// }

	if n.Year != 0 {
		response.ProductionYear = n.Year
	}

	switch len(n.Premiered) {
	case 0:
		break
	case 10:
		response.PremiereDate = n.Premiered + "T00:00:00.0000000Z"
	default:
		log.Printf("unknown date format info %s", n.Premiered)
	}
}

func buildMediaSource(filename string, episodeNfo *Nfo) (mediasources []JFMediaSources) {
	basename := filepath.Base(filename)

	mediasources = []JFMediaSources{
		{
			ID:                    idHash(filename),
			ETag:                  idHash(filename),
			Name:                  basename,
			Path:                  basename,
			Type:                  "Default",
			Container:             "mp4",
			Protocol:              "File",
			VideoType:             "VideoFile",
			Size:                  4264940672,
			IsRemote:              false,
			ReadAtNativeFramerate: false,
			IgnoreDts:             false,
			IgnoreIndex:           false,
			GenPtsInput:           false,
			SupportsTranscoding:   true,
			SupportsDirectStream:  true,
			SupportsDirectPlay:    true,
			IsInfiniteStream:      false,
			RequiresOpening:       false,
			RequiresClosing:       false,
			RequiresLooping:       false,
			SupportsProbing:       true,
			Formats:               []string{},
			MediaStreams: []JFMediaStreams{
				{
					Codec:              "h264",
					CodecTag:           "avc1",
					Language:           "eng",
					TimeBase:           "1/16000",
					VideoRange:         "SDR",
					VideoRangeType:     "SDR",
					AudioSpatialFormat: "None",
					DisplayTitle:       "720p H264 SDR",
					NalLengthSize:      "4",
					IsInterlaced:       false,
					IsAVC:              true,
					BitDepth:           8,
					RefFrames:          1,
					IsDefault:          true,
					IsForced:           false,
					IsHearingImpaired:  false,
					Height:             546,
					Width:              1280,
					// AverageFrameRate:       23.81,
					// RealFrameRate:          23.81,
					Profile:                "High",
					Type:                   "Video",
					AspectRatio:            "2.35:1",
					Index:                  0,
					IsExternal:             false,
					IsTextSubtitleStream:   false,
					SupportsExternalStream: false,
					PixelFormat:            "yuv420p",
					Level:                  41,
					IsAnamorphic:           false,
				},
				{
					Codec:                  "aac",
					CodecTag:               "mp4a",
					Language:               "eng",
					TimeBase:               "1/48000",
					VideoRange:             "Unknown",
					VideoRangeType:         "Unknown",
					AudioSpatialFormat:     "None",
					LocalizedDefault:       "Default",
					LocalizedExternal:      "External",
					DisplayTitle:           "English - AAC - Stereo - Default",
					IsInterlaced:           false,
					IsAVC:                  false,
					ChannelLayout:          "stereo",
					BitRate:                255577,
					Channels:               2,
					SampleRate:             48000,
					IsDefault:              true,
					IsForced:               false,
					IsHearingImpaired:      false,
					Profile:                "LC",
					Type:                   "Audio",
					Index:                  1,
					IsExternal:             false,
					IsTextSubtitleStream:   false,
					SupportsExternalStream: false,
					Level:                  0,
				},
			},
			MediaAttachments:        []JFMediaAttachments{},
			RequiredHTTPHeaders:     JFRequiredHTTPHeaders{},
			TranscodingSubProtocol:  "http",
			DefaultAudioStreamIndex: 1,
		},
	}

	// Populate data from nfo: duration, bitrate, framerate if available
	if episodeNfo != nil && episodeNfo.FileInfo != nil &&
		episodeNfo.FileInfo.StreamDetails != nil &&
		episodeNfo.FileInfo.StreamDetails.Video != nil {

		mediasources[0].RunTimeTicks = int64(episodeNfo.FileInfo.StreamDetails.Video.DurationInSeconds * 10000000)

		mediasources[0].Bitrate = episodeNfo.FileInfo.StreamDetails.Video.Bitrate
		mediasources[0].MediaStreams[0].AverageFrameRate = float64(episodeNfo.FileInfo.StreamDetails.Video.FrameRate)
		mediasources[0].MediaStreams[0].RealFrameRate = float64(episodeNfo.FileInfo.StreamDetails.Video.FrameRate)
	}

	return
}

// curl -v 'http://127.0.0.1:9090/Shows/NextUp?UserId=2b1ec0a52b09456c9823a367d84ac9e5&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&StartIndex=0&Limit=20'

func showsNextUpHandler(w http.ResponseWriter, r *http.Request) {
	c, i := getItemByID("rVFG3EzPthk2wowNkqUl")
	response := JFShowsNextUpResponse{
		Items: []JFItem{
			buildJFItem(c, i),
		},
		TotalRecordCount: 1,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/Items/rVFG3EzPthk2wowNkqUl/Images/Backdrop?tag=7cec54f0c8f362c75588e83d76fefa75'
// curl -v 'http://127.0.0.1:9090/Items/rVFG3EzPthk2wowNkqUl/Images/Logo?tag=e28fbe648d2dbb76b65c14f14e6b1d72'
// curl -v 'http://127.0.0.1:9090/Items/q2e2UzCOd9zkmJenIOph/Images/Primary?tag=70931a7d8c147c9e2c0aafbad99e03e5'
// curl -v 'http://127.0.0.1:9090/Items/rVFG3EzPthk2wowNkqUl/Images/Primary?tag=268b80952354f01d5a184ed64b36dd52'
// curl -v 'http://127.0.0.1:9090/Items/2vx0ZYKeHxbh5iWhloIB/Images/Primary?tag=redirect_https://image.tmdb.org/t/p/original/3E4x5doNuuu6i9Mef6HPrlZjNb1.jpg'

func itemsImagesHandler(w http.ResponseWriter, r *http.Request) {
	// handle tag-based redirects for item imagery that is external (e.g. external images of actors)
	// for these we do not care about the provided item id
	queryparams := r.URL.Query()
	tag := queryparams.Get("tag")
	if strings.HasPrefix(tag, tagprefix_redirect) {
		http.Redirect(w, r, strings.TrimPrefix(tag, tagprefix_redirect), http.StatusFound)
		return
	}
	if strings.HasPrefix(tag, tagprefix_file) {
		serveFile(w, r, strings.TrimPrefix(tag, tagprefix_file))
		return
	}

	vars := mux.Vars(r)
	itemId := vars["item"]
	if strings.HasPrefix(itemId, itemprefix_episode) {
		showId, episodebasepath, err := getEpisodeIDDetails(itemId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		// log.Printf("\n\n0: %s\n1: %s\n2: %s\n\n", tag, showId, episodebasepath)
		c, showItem := getItemByID(showId)
		if showItem == nil {
			http.Error(w, "Item not found (could not find show)", http.StatusNotFound)
			return
		}
		filename := c.Directory + "/" + episodebasepath + "-thumb.jpg"

		// log.Printf("FILENAME %s\n", filename)
		serveFile(w, r, filename)
		return
	}

	c, i := getItemByID(itemId)
	if i == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	switch vars["type"] {
	case "Primary":
		serveFile(w, r, c.Directory+"/"+i.Name+"/"+"poster.jpg")
		return
	case "Backdrop":
		serveFile(w, r, c.Directory+"/"+i.Name+"/"+"fanart.jpg")
		return
		// We do not have artwork on disk for logo requests
		// case "Logo":
		// return
	}
	log.Printf("Unknown image type requested: %s\n", vars["type"])
	http.Error(w, "Item image not found", http.StatusNotFound)
}

// curl -v 'http://127.0.0.1:9090/Items/68d73f6f48efedb7db697bf9fee580cb/PlaybackInfo?UserId=2b1ec0a52b09456c9823a367d84ac9e5'

func itemsPlaybackInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId := vars["item"]

	c, i := getItemByID(itemId)
	if i == nil || i.Video == "" {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	item := buildJFItem(c, i)

	response := JFUsersPlaybackInfoResponse{
		MediaSources:  item.MediaSources,
		PlaySessionID: "fc3b27127bf84ed89a300c6285d697e2",
	}
	serveJSON(response, w)
}

// return commercial, preview, recap, outro, intro segments of an item
func itemsMediaSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	response := UserItemsResponse{
		Items:            []JFItem{},
		TotalRecordCount: 0,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v -I 'http://127.0.0.1:9090/Videos/NrXTYiS6xAxFj4QAiJoT/stream'

func videoStreamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId := vars["item"]

	// Is episode?
	if strings.HasPrefix(itemId, itemprefix_episode) {
		showId, episodebasepath, err := getEpisodeIDDetails(itemId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		c, showItem := getItemByID(showId)
		if showItem == nil {
			http.Error(w, "Could not find show", http.StatusNotFound)
			return
		}

		filename := c.Directory + "/" + episodebasepath + ".mp4"
		serveFile(w, r, filename)
		return
	}

	c, i := getItemByID(vars["item"])
	if i == nil || i.Video == "" {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	serveFile(w, r, c.Directory+"/"+i.Name+"/"+i.Video)
}

// return list of actors (used by Infuse's search)
func personsHandler(w http.ResponseWriter, r *http.Request) {
	response := UserItemsResponse{
		Items:            []JFItem{},
		TotalRecordCount: 0,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// session handling
func sessionsPlayingHandler(w http.ResponseWriter, r *http.Request) {
	_, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	w.WriteHeader(http.StatusNoContent)
}

// misc stuff
func serveFile(w http.ResponseWriter, r *http.Request, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		http.Error(w, "Could not retrieve file info", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, fileStat.Name(), fileStat.ModTime(), file)
}

func genCollectionID(id int) (collectionID string) {
	collectionID = itemprefix_collection + fmt.Sprintf("%d", id)
	return
}

func getCollectionID(input string) (id string, err error) {
	if !strings.HasPrefix(input, itemprefix_collection) {
		err = errors.New("not a collectionid")
		return
	}
	id = strings.TrimPrefix(input, itemprefix_collection)
	return
}

func getEpisodeIDDetails(episodeid string) (showid, episodebasepath string, err error) {
	if !strings.HasPrefix(episodeid, itemprefix_episode) {
		err = errors.New("not an episodeid")
		return
	}
	episode_details, _ := url.QueryUnescape(strings.TrimPrefix(episodeid, itemprefix_episode))
	re := regexp.MustCompile(`([0-9A-Za-z]+)_(.+)`)
	matches := re.FindStringSubmatch(episode_details)
	if len(matches) != 3 {
		err = errors.New("Item not found (could not find episode)")
		return
	}
	showid = matches[1]
	episodebasepath = matches[2]
	return
}

func getItemByID(showId string) (c *Collection, i *Item) {
	for _, c := range config.Collections {
		if i = getItem(c.Name_, showId); i != nil {
			return &c, i
		}
	}
	return nil, nil
}
