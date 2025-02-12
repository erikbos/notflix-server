package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// API definitions: https://swagger.emby.media/ & https://api.jellyfin.org/
// Docs: https://github.com/mediabrowser/emby/wiki

func registerJellyfinHandlers(s *mux.Router) {
	r := s.UseEncodedPath()

	gzip := func(handler http.HandlerFunc) http.Handler {
		return handlers.CompressHandler(http.HandlerFunc(handler))
	}

	r.Handle("/System/Info/Public", gzip(systemInfoHandler))
	r.Handle("/DisplayPreferences/usersettings", gzip(displayPreferencesHandler))

	r.Handle("/Users/AuthenticateByName", gzip(usersAuthenticateByNameHandler)).Methods("POST")
	r.Handle("/Users/{user}", gzip(usersHandler))
	r.Handle("/Users/{user}/Views", gzip(usersViewsHandler))
	r.Handle("/Users/{user}/GroupingOptions", gzip(usersGroupingOptionsHandler))
	r.Handle("/Users/{user}/Items", gzip(usersItemsHandler))
	r.Handle("/Users/{user}/Items/Latest", gzip(usersItemsLatestHandler))
	r.Handle("/Users/{user}/Items/{item}", gzip(usersItemHandler))
	r.Handle("/Users/{user}/Items/Resume", gzip(usersItemsResumeHandler))

	r.Handle("/Library/VirtualFolders", gzip(libraryVirtualFoldersHandler))
	r.Handle("/Shows/NextUp", gzip(showsNextUpHandler))
	r.Handle("/Shows/{show}/Seasons", gzip(showsSeasonsHandler))
	r.Handle("/Shows/{show}/Episodes", gzip(showsEpisodesHandler))

	r.Handle("/Items/{item}/Images/{type}", gzip(itemsImagesHandler))
	r.Handle("/Items/{item}/PlaybackInfo", gzip(itemsPlaybackInfoHandler))
	r.Handle("/MediaSegments/{item}", gzip(mediaSegmentsHandler))
	r.Handle("/Videos/{item}/stream", gzip(videoStreamHandler))

	r.Handle("/Persons", gzip(personsHandler))

	r.Handle("/Sessions/Playing", gzip(sessionsPlayingHandler)).Methods("POST")
	r.Handle("/Sessions/Playing/Progress", gzip(sessionsPlayingProgressHandler)).Methods("POST")
	r.Handle("/Sessions/Playing/Stopped", gzip(sessionsPlayingStoppedHandler)).Methods("POST")
}

const (
	// Misc IDs for api responses
	serverID              = "2b11644442754f02a0c1e45d2a9f5c71"
	userID                = "2b1ec0a52b09456c9823a367d84ac9e5"
	collectionRootID      = "e9d5075a555c1cbc394eec4cef295274"
	displayPreferencesID  = "f137a2dd21bbc1b99aa5c0f6bf02a805"
	collectionTypeMovies  = "movies"
	collectionTypeTVShows = "tvshows"

	// itemid prefixes
	itemprefix_collection = "collection_"
	itemprefix_show       = "show_"
	itemprefix_season     = "season_"
	itemprefix_episode    = "episode_"

	// imagetag prefix will get HTTP-redirected
	tagprefix_redirect = "redirect_"
	// imagetag prefix means we will serve the filename from local disk
	tagprefix_file = "file_"
)

var loggedInUser = JFUser{
	Id:                        userID,
	ServerId:                  serverID,
	Name:                      "user",
	HasPassword:               true,
	HasConfiguredPassword:     true,
	HasConfiguredEasyPassword: false,
	EnableAutoLogin:           false,
	LastLoginDate:             time.Now(),
	LastActivityDate:          time.Now(),
}

// curl -v http://127.0.0.1:9090/System/Info/Public
func systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	response := JFSystemInfoResponse{
		Id:           serverID,
		LocalAddress: "http://" + hostname + ":9090/",
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
func usersAuthenticateByNameHandler(w http.ResponseWriter, r *http.Request) {
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
func usersViewsHandler(w http.ResponseWriter, r *http.Request) {
	items := []JFItem{}

	for _, c := range config.Collections {
		itemID := genCollectionID(c.SourceId)

		// Root item
		item := JFItem{
			ServerID:                 serverID,
			ParentID:                 collectionRootID,
			Type:                     "CollectionFolder",
			IsFolder:                 true,
			DateCreated:              time.Now().UTC(),
			PremiereDate:             time.Now().UTC(),
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
			MediaType:                "Unknown",
			LockData:                 false,
		}

		switch c.Type {
		case "movies":
			item.CollectionType = collectionTypeMovies
		case "shows":
			item.CollectionType = collectionTypeTVShows
		}

		items = append(items, item)
	}

	response := JFUserViewsResponse{
		Items:            items,
		TotalRecordCount: len(config.Collections),
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
// handle individual item: any type: collection, a movie/show or individual file
func usersItemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemId := vars["item"]

	splitted := strings.Split(itemId, "_")
	if len(splitted) == 2 {
		switch splitted[0] {
		case "collection":
			collectionItem, err := buildJFItemCollection(itemId)
			if err != nil {
				http.Error(w, "Could not find collection", http.StatusNotFound)
				return

			}
			serveJSON(collectionItem, w)
			return
		case "season":
			seasonItem, err := buildJFItemSeason(itemId)
			if err != nil {
				http.Error(w, "Could not find season", http.StatusNotFound)
				return
			}
			serveJSON(seasonItem, w)
			return
		case "episode":
			episodeItem, err := buildJFItemEpisode(itemId)
			if err != nil {
				http.Error(w, "Could not find episode", http.StatusNotFound)
				return
			}
			serveJSON(episodeItem, w)
			return
		default:
			log.Print("Item request for unknown prefix!")
			http.Error(w, "Unknown item prefix", http.StatusInternalServerError)
			return
		}
	}

	// Try to find individual item
	c, i := getItemByID(itemId)
	if i == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	serveJSON(buildJFItem(c, i, false), w)
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
		// TODO: this could be a search by person :)
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

	// Apply pagination
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
		responseItems = append(responseItems, buildJFItem(c, i, true))
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
		buildJFItem(c1, i1, true),
		buildJFItem(c2, i2, true),
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
			l.CollectionType = collectionTypeMovies
		case "shows":
			l.CollectionType = collectionTypeTVShows
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
	_, i := getItemByID(showId)
	if i == nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}
	// Create API response
	seasons := []JFItem{}
	for _, s := range i.Seasons {
		season, err := buildJFItemSeason(s.Id)
		if err != nil {
			log.Printf("buildJFItemSeason returned error %s", err)
			continue
		}
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
// generate episode overview for one season of a show
func showsEpisodesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, i := getItemByID(vars["show"])
	if i == nil {
		http.Error(w, "Show not found", http.StatusNotFound)
		return
	}

	// Do we need to filter down overview by a particular season?
	RequestedSeasonId := r.URL.Query().Get("SeasonId")

	// Create API response for requested season
	episodes := []JFItem{}
	for _, s := range i.Seasons {
		// Limit results to a season if id provided
		if RequestedSeasonId != "" && itemprefix_season+s.Id != RequestedSeasonId {
			continue
		}
		for _, e := range s.Episodes {
			episodeId := itemprefix_episode + e.Id
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
		w.Header().Set("cache-control", "max-age=2592000")
		http.Redirect(w, r, strings.TrimPrefix(tag, tagprefix_redirect), http.StatusFound)
		return
	}
	if strings.HasPrefix(tag, tagprefix_file) {
		w.Header().Set("cache-control", "max-age=2592000")
		serveFile(w, r, strings.TrimPrefix(tag, tagprefix_file))
		return
	}

	vars := mux.Vars(r)
	itemId := vars["item"]
	imageType := vars["type"]

	splitted := strings.Split(itemId, "_")
	if len(splitted) == 2 {
		switch splitted[0] {
		// case "collection":
		// 	collectionItem, err := buildJFItemCollection(itemId)
		// 	if err != nil {
		// 		http.Error(w, "Could not find collection", http.StatusNotFound)
		// 		return

		// 	}
		// 	serveJSON(collectionItem, w)
		// 	return
		case "season":
			c, item, season := getSeasonByID(itemId)
			if season == nil {
				http.Error(w, "Could not find season", http.StatusNotFound)
				return
			}
			switch imageType {
			case "Primary":
				serveFile(w, r, c.Directory+"/"+item.Name+"/"+season.Poster)
				return
			default:
				log.Printf("Image request %s, unknown type %s", itemId, imageType)
				return
			}
		case "episode":
			c, item, _, episode := getEpisodeByID(itemId)
			if episode == nil {
				http.Error(w, "Item not found (could not find episode)", http.StatusNotFound)
				return
			}
			serveFile(w, r, c.Directory+"/"+item.Name+"/"+episode.Thumb)
			return
		default:
			log.Printf("Image request for unknown prefix %s!", itemId)
			http.Error(w, "Unknown image item prefix", http.StatusInternalServerError)
			return
		}
	}

	c, i := getItemByID(itemId)
	if i == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	switch vars["type"] {
	case "Primary":
		w.Header().Set("cache-control", "max-age=2592000")
		serveFile(w, r, c.Directory+"/"+i.Name+"/"+i.Poster)
		return
	case "Backdrop":
		w.Header().Set("cache-control", "max-age=2592000")
		serveFile(w, r, c.Directory+"/"+i.Name+"/"+i.Fanart)
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
	// vars := mux.Vars(r)
	// itemId := vars["item"]

	// c, i := getItemByID(itemId)
	// if i == nil || i.Video == "" {
	// 	http.Error(w, "Item not found", http.StatusNotFound)
	// 	return
	// }
	// item := buildJFItem(c, i, true)

	response := JFUsersPlaybackInfoResponse{
		MediaSources: buildMediaSource("test.mp4", nil),
		// TODO this static id should be generated based upon authenticated user
		// this id is used when submitting playstate via /Sessions/Playing endpoints
		PlaySessionID: "fc3b27127bf84ed89a300c6285d697e2",
	}
	serveJSON(response, w)
}

// return information about intro, commercial, preview, recap, outro segments
// of an item, not supported.
func mediaSegmentsHandler(w http.ResponseWriter, r *http.Request) {
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
		c, item, _, episode := getEpisodeByID(itemId)
		if episode == nil {
			http.Error(w, "Could not find episode", http.StatusNotFound)
			return
		}
		serveFile(w, r, c.Directory+"/"+item.Name+"/"+episode.Video)
		return
	}

	c, i := getItemByID(vars["item"])
	if i == nil || i.Video == "" {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	serveFile(w, r, c.Directory+"/"+i.Name+"/"+i.Video)
}

// return list of actors (hit by Infuse's search)
// not supported
func personsHandler(w http.ResponseWriter, r *http.Request) {
	response := UserItemsResponse{
		Items:            []JFItem{},
		TotalRecordCount: 0,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// curl -v 'http://127.0.0.1:9090/Shows/NextUp?UserId=2b1ec0a52b09456c9823a367d84ac9e5&Fields=DateCreated,Etag,Genres,MediaSources,AlternateMediaSources,Overview,ParentId,Path,People,ProviderIds,SortName,RecursiveItemCount,ChildCount&StartIndex=0&Limit=20'
func showsNextUpHandler(w http.ResponseWriter, r *http.Request) {
	c, i := getItemByID("rVFG3EzPthk2wowNkqUl")
	response := JFShowsNextUpResponse{
		Items: []JFItem{
			buildJFItem(c, i, true),
		},
		TotalRecordCount: 1,
		StartIndex:       0,
	}
	serveJSON(response, w)
}

// PositionTicks are in micro seconds
const TicsToSeconds = 10000000

func sessionsPlayingHandler(w http.ResponseWriter, r *http.Request) {
	var request JFPlayState
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	log.Printf("sessionsPlayingHandler %+v\n", request)
	log.Printf("sessionsPlayingHandler Progress: %d seconds\n", request.PositionTicks/TicsToSeconds)
	w.WriteHeader(http.StatusNoContent)
}

func sessionsPlayingProgressHandler(w http.ResponseWriter, r *http.Request) {
	var request JFPlayState
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	log.Printf("sessionsPlayingProgressHandler %+v\n", request)
	log.Printf("sessionsPlayingProgressHandler Progress: %d seconds\n", request.PositionTicks/TicsToSeconds)
	w.WriteHeader(http.StatusNoContent)
}

func sessionsPlayingStoppedHandler(w http.ResponseWriter, r *http.Request) {
	var request JFPlayState
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	log.Printf("sessionsPlayingStoppedHandler %+v\n", request)
	log.Printf("sessionsPlayingStoppedHandler Progress: %d seconds\n", request.PositionTicks/TicsToSeconds)
	w.WriteHeader(http.StatusNoContent)
}

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
		DateCreated:              time.Now().UTC(),
		Type:                     "CollectionFolder",
		IsFolder:                 true,
		EnableMediaSourceDisplay: true,
		ChildCount:               len(c.Items),
		DisplayPreferencesID:     displayPreferencesID,
		ExternalUrls:             []JFExternalUrls{},
		PlayAccess:               "Full",
		PrimaryImageAspectRatio:  1.7777777777777777,
		RemoteTrailers:           []JFRemoteTrailers{},
		LocationType:             "FileSystem",
		Path:                     "/collection",
		LockData:                 false,
		MediaType:                "Unknown",
		ParentID:                 "e9d5075a555c1cbc394eec4cef295274",
		CanDelete:                false,
		CanDownload:              false,
		SpecialFeatureCount:      0,
	}
	switch c.Type {
	case "movies":
		response.CollectionType = collectionTypeMovies
	case "shows":
		response.CollectionType = collectionTypeTVShows
	}
	response.SortName = response.CollectionType
	return
}

// buildJFItem builds movie or show from db
func buildJFItem(c *Collection, i *Item, listView bool) (response JFItem) {
	response = JFItem{
		ID:                      i.Id,
		ParentID:                idHash(c.Name_),
		ServerID:                serverID,
		Name:                    i.Name,
		OriginalTitle:           i.Name,
		SortName:                i.Name,
		ForcedSortName:          i.Name,
		Etag:                    idHash(i.Id),
		DateCreated:             time.Unix(i.FirstVideo/1000, 0).UTC(),
		PremiereDate:            time.Unix(i.FirstVideo/1000, 0).UTC(),
		PrimaryImageAspectRatio: 0.6666666666666666,
	}

	response.ImageTags = &JFImageTags{
		Primary: "primary_" + i.Id,
	}

	// Required to have Infuse load backdrop of episode
	response.BackdropImageTags = []string{
		response.ID,
	}

	if c.Type == "movies" {
		response.Type = "Movie"
		response.IsFolder = false
		response.LocationType = "FileSystem"
		response.Path = "file.mp4"
		response.MediaType = "Video"
		response.VideoType = "VideoFile"
		response.Container = "mov,mp4,m4a"

		lazyLoadNFO(&i.Nfo, i.NfoPath)
		filename := c.Directory + "/" + i.Name + "/" + i.Video
		response.MediaSources = buildMediaSource(filename, i.Nfo)

		// listview = true, movie carousel return both primary and BackdropImageTags
		// non-listview = false, remove primary (thumbnail) image reference
		if !listView {
			response.ImageTags = nil
		}
	}

	if c.Type == "shows" {
		response.Type = "Series"
		response.IsFolder = true
		response.ChildCount = len(i.Seasons)
	}

	enrichResponseWithNFO(&response, i.Nfo)

	return response
}

// buildJFItemSeason builds season
func buildJFItemSeason(seasonid string) (response JFItem, err error) {
	_, show, season := getSeasonByID(seasonid)
	if season == nil {
		err = errors.New("could not find season")
		return
	}

	response = JFItem{
		Type:               "Season",
		ServerID:           serverID,
		ParentID:           show.Id,
		SeriesID:           show.Id,
		ID:                 itemprefix_season + seasonid,
		Etag:               idHash(seasonid),
		SeriesName:         show.Name,
		IndexNumber:        season.SeasonNo,
		Name:               fmt.Sprintf("Season %d", season.SeasonNo),
		SortName:           fmt.Sprintf("%04d", season.SeasonNo),
		IsFolder:           true,
		LocationType:       "FileSystem",
		MediaType:          "Unknown",
		ChildCount:         len(season.Episodes),
		RecursiveItemCount: len(season.Episodes),
		DateCreated:        time.Now().UTC(),
		PremiereDate:       time.Now().UTC(),
		ImageTags: &JFImageTags{
			Primary: "season",
		},
	}
	return response, nil
}

// buildJFItemEpisode builds episode
func buildJFItemEpisode(episodeid string) (response JFItem, err error) {
	_, show, _, episode := getEpisodeByID(episodeid)
	if episode == nil {
		err = errors.New("could not find episode")
		return
	}

	response = JFItem{
		Type:         "Episode",
		ID:           episodeid,
		Etag:         idHash(episodeid),
		ServerID:     serverID,
		SeriesName:   show.Name,
		SeriesID:     idHash(show.Name),
		LocationType: "FileSystem",
		Path:         "episode.mp4",
		IsFolder:     false,
		MediaType:    "Video",
		VideoType:    "VideoFile",
		Container:    "mov,mp4,m4a",
		HasSubtitles: true,
		DateCreated:  time.Unix(episode.VideoTS/1000, 0).UTC(),
		PremiereDate: time.Unix(episode.VideoTS/1000, 0).UTC(),
		ImageTags: &JFImageTags{
			Primary: "episode",
		},
	}

	// Get a bunch of metadata from show-level nfo
	lazyLoadNFO(&show.Nfo, show.NfoPath)
	if show.Nfo != nil {
		enrichResponseWithNFO(&response, show.Nfo)
	}

	// Remove ratings as we do not want ratings from series apply to an episode
	response.OfficialRating = ""
	response.CommunityRating = 0

	// Enrich and override metadata using episode nfo, if available, as it is more specific than data from show
	lazyLoadNFO(&episode.Nfo, episode.NfoPath)
	if episode.Nfo != nil {
		enrichResponseWithNFO(&response, episode.Nfo)
	}

	// Add some generic mediasource to indicate "720p, stereo"
	response.MediaSources = buildMediaSource(episode.Video, episode.Nfo)

	return response, nil
}

func lazyLoadNFO(n **Nfo, filename string) {
	// NFO already loaded and parsed?
	if *n != nil {
		return
	}
	if file, err := os.Open(filename); err == nil {
		defer file.Close()
		*n = decodeNfo(file)
	}
}

func enrichResponseWithNFO(response *JFItem, n *Nfo) {
	if n == nil {
		return
	}

	response.Name = n.Title
	response.Overview = n.Plot
	if n.Tagline != "" {
		response.Taglines = []string{n.Tagline}
	}

	// Handle episode naming & numbering
	if n.Season != "" {
		response.SeasonName = "Season " + n.Season
		response.ParentIndexNumber, _ = strconv.Atoi(n.Season)
	}
	if n.Episode != "" {
		response.IndexNumber, _ = strconv.Atoi(n.Episode)
	}
	if response.ParentIndexNumber != 0 && response.IndexNumber != 0 {
		response.SortName = fmt.Sprintf("%03s - %04s - %s", n.Season, n.Episode, n.Title)
	}

	// TV-14
	response.OfficialRating = n.Mpaa

	// ProviderIds: JFProviderIds{
	// 	Tmdb:           "9659",
	// 	Imdb:           "tt0079501",
	// 	TmdbCollection: "8945",
	// },

	if n.Rating != 0 {
		response.CommunityRating = math.Round(float64(n.Rating)*10) / 10
	}

	if len(n.Genre) != 0 {
		normalizedGenres := normalizeGenres(n.Genre)
		// Why do we populate two response fields with same data?
		response.Genres = normalizedGenres
		for _, genre := range normalizedGenres {
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

	if n.Premiered != "" {
		if parsedTime, err := parseTime(n.Premiered); err == nil {
			response.PremiereDate = parsedTime
		}
	}
	if n.Aired != "" {
		if parsedTime, err := parseTime(n.Aired); err == nil {
			response.PremiereDate = parsedTime
		}
	}
}

func buildMediaSource(filename string, n *Nfo) (mediasources []JFMediaSources) {
	// todo: this should be replaced with actual mp4 file detail gathering

	basename := filepath.Base(filename)
	mediasource := JFMediaSources{
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
	}

	// log.Printf("buildMediaSource: n: %+v, n2: %+v, n3: %+v\n", n, n.FileInfo, n.FileInfo.StreamDetails)
	if n == nil || n.FileInfo == nil || n.FileInfo.StreamDetails == nil {
		return []JFMediaSources{mediasource}
	}

	NfoVideo := n.FileInfo.StreamDetails.Video
	mediasource.Bitrate = NfoVideo.Bitrate
	mediasource.RunTimeTicks = int64(NfoVideo.DurationInSeconds) * 10000000

	// Take first alpha-3 language, ignore others
	language := n.FileInfo.StreamDetails.Audio.Language[0:3]

	// Create video stream with high-level details based upon NFO
	videostream := JFMediaStreams{
		Index:            0,
		Type:             "Video",
		IsDefault:        true,
		Language:         language,
		AverageFrameRate: math.Round(float64(NfoVideo.FrameRate*100)) / 100,
		RealFrameRate:    math.Round(float64(NfoVideo.FrameRate*100)) / 100,
		TimeBase:         "1/16000",
		Height:           NfoVideo.Height,
		Width:            NfoVideo.Width,
		Codec:            NfoVideo.Codec,
	}
	switch strings.ToLower(NfoVideo.Codec) {
	case "x264":
		fallthrough
	case "h264":
		videostream.Codec = "h264"
		videostream.CodecTag = "avc1"
	case "hevc":
		videostream.Codec = "hevc"
		videostream.CodecTag = "hvc1"
	default:
		log.Printf("Nfo of %s has unknown video codec %s", filename, NfoVideo.Codec)
	}

	mediasource.MediaStreams = append(mediasource.MediaStreams, videostream)

	// Create audio stream with high-level details based upon NFO
	audiostream := JFMediaStreams{
		Index:              1,
		Type:               "Audio",
		Language:           language,
		TimeBase:           "1/48000",
		SampleRate:         48000,
		AudioSpatialFormat: "None",
		LocalizedDefault:   "Default",
		LocalizedExternal:  "External",
		IsInterlaced:       false,
		IsAVC:              false,
		IsDefault:          true,
	}

	NfoAudio := n.FileInfo.StreamDetails.Audio
	audiostream.BitRate = NfoAudio.Bitrate
	audiostream.Channels = NfoAudio.Channels

	switch NfoAudio.Channels {
	case 2:
		audiostream.Title = "Stereo"
		audiostream.ChannelLayout = "stereo"
	case 6:
		audiostream.Title = "5.1 Channel"
		audiostream.ChannelLayout = "5.1"
	default:
		log.Printf("Nfo of %s has unknown audio channel configuration %d", filename, NfoAudio.Channels)
	}

	switch strings.ToLower(NfoAudio.Codec) {
	case "ac3":
		audiostream.Codec = "ac3"
		audiostream.CodecTag = "ac-3"
	case "aac":
		audiostream.Codec = "aac"
		audiostream.CodecTag = "mp4a"
	default:
		log.Printf("Nfo of %s has unknown audio codec %s", filename, NfoAudio.Codec)
	}

	audiostream.DisplayTitle = audiostream.Title + " - " + strings.ToUpper(audiostream.Codec)

	mediasource.MediaStreams = append(mediasource.MediaStreams, audiostream)

	return []JFMediaSources{mediasource}
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

func getItemByID(itemId string) (c *Collection, i *Item) {
	for _, c := range config.Collections {
		if i = getItem(c.Name_, itemId); i != nil {
			return &c, i
		}
	}
	return nil, nil
}

func getSeasonByID(saesonId string) (*Collection, *Item, *Season) {
	saesonId = strings.TrimPrefix(saesonId, itemprefix_season)

	// fixme: wooho O(n^^3) "just temporarily.."
	for _, c := range config.Collections {
		for _, i := range c.Items {
			for _, s := range i.Seasons {
				if s.Id == saesonId {
					return &c, i, &s
				}
			}
		}
	}
	return nil, nil, nil
}

func getEpisodeByID(episodeId string) (*Collection, *Item, *Season, *Episode) {
	episodeId = strings.TrimPrefix(episodeId, itemprefix_episode)

	// fixme: wooho O(n^^4) "just temporarily.."
	for _, c := range config.Collections {
		for _, i := range c.Items {
			for _, s := range i.Seasons {
				for _, e := range s.Episodes {
					if e.Id == episodeId {
						return &c, i, &s, &e
					}

				}
			}
		}
	}
	return nil, nil, nil, nil
}

func searchItemByName(searchkey string) (*Collection, *Item) {
	// fixme: wooho O(n^^4) "just temporarily.."
	for _, c := range config.Collections {
		for _, i := range c.Items {
			if strings.Contains(i.Name, searchkey) {
				return &c, i
			}
		}
	}
	return nil, nil
}

func parseTime(input string) (parsedTime time.Time, err error) {
	timeFormats := []string{
		"15:04:05",
		"2006-01-02",
		"2006/01/02",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"02 Jan 2006",
		"02 Jan 2006 15:04:05",
	}

	// Try each format until one succeeds
	for _, format := range timeFormats {
		if parsedTime, err = time.Parse(format, input); err == nil {
			// log.Printf("Parsed: %s as %v\n", input, parsedTime)
			return
		}
	}
	return
}
