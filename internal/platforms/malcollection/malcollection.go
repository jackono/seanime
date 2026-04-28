package malcollection

import (
	"fmt"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/api/mal"
	"seanime/internal/database/db"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

const (
	TrackerEnvName = "SEANIME_TRACKER"
	TrackerAnilist = "anilist"
	TrackerMAL     = "mal"
)

var trackerModeState = struct {
	sync.RWMutex
	mode string
}{}

func NormalizeTrackerMode(mode string) string {
	if strings.EqualFold(mode, TrackerMAL) {
		return TrackerMAL
	}
	return TrackerAnilist
}

func SetTrackerMode(mode string) {
	trackerModeState.Lock()
	defer trackerModeState.Unlock()
	trackerModeState.mode = NormalizeTrackerMode(mode)
}

func TrackerMode() string {
	trackerModeState.RLock()
	mode := trackerModeState.mode
	trackerModeState.RUnlock()
	if mode != "" {
		return NormalizeTrackerMode(mode)
	}
	return NormalizeTrackerMode(os.Getenv(TrackerEnvName))
}

func Enabled() bool {
	return TrackerMode() == TrackerMAL
}

func GetAnimeCollection(database *db.Database, logger *zerolog.Logger) (*anilist.AnimeCollection, error) {
	if !Enabled() {
		return nil, fmt.Errorf("MAL tracker is disabled")
	}

	malInfo, err := database.GetMalInfo()
	if err != nil {
		return nil, err
	}

	malInfo, err = mal.VerifyMALAuth(malInfo, database, logger)
	if err != nil {
		return nil, err
	}

	entries, err := mal.NewWrapper(malInfo.AccessToken, logger).GetAnimeCollection()
	if err != nil {
		return nil, err
	}

	return ConvertAnimeCollection(entries), nil
}

func GetAnime(database *db.Database, logger *zerolog.Logger, malID int) (*anilist.BaseAnime, error) {
	if !Enabled() {
		return nil, fmt.Errorf("MAL tracker is disabled")
	}

	anime, err := getMALAnime(database, logger, malID)
	if err != nil {
		return nil, err
	}

	return ConvertBasicAnime(anime), nil
}

func GetAnimeDetails(database *db.Database, logger *zerolog.Logger, malID int) (*anilist.AnimeDetailsById_Media, error) {
	if !Enabled() {
		return nil, fmt.Errorf("MAL tracker is disabled")
	}

	anime, err := getMALAnime(database, logger, malID)
	if err != nil {
		return nil, err
	}

	return ConvertAnimeDetails(anime), nil
}

func getMALAnime(database *db.Database, logger *zerolog.Logger, malID int) (*mal.BasicAnime, error) {
	malInfo, err := database.GetMalInfo()
	if err != nil {
		return nil, err
	}

	malInfo, err = mal.VerifyMALAuth(malInfo, database, logger)
	if err != nil {
		return nil, err
	}

	return mal.NewWrapper(malInfo.AccessToken, logger).GetAnimeDetails(malID)
}

func ConvertAnimeCollection(malEntries []*mal.AnimeListEntry) *anilist.AnimeCollection {
	listsByStatus := map[anilist.MediaListStatus]*anilist.AnimeCollection_MediaListCollection_Lists{}
	for _, status := range []anilist.MediaListStatus{
		anilist.MediaListStatusCurrent,
		anilist.MediaListStatusCompleted,
		anilist.MediaListStatusPaused,
		anilist.MediaListStatusDropped,
		anilist.MediaListStatusPlanning,
		anilist.MediaListStatusRepeating,
	} {
		statusCopy := status
		name := animeListStatusName(status)
		isCustom := false
		listsByStatus[status] = &anilist.AnimeCollection_MediaListCollection_Lists{
			Name:         &name,
			Status:       &statusCopy,
			IsCustomList: &isCustom,
			Entries:      make([]*anilist.AnimeCollection_MediaListCollection_Lists_Entries, 0),
		}
	}

	for _, malEntry := range malEntries {
		if malEntry == nil || malEntry.Node.ID == 0 {
			continue
		}
		status := mapAnimeListStatus(malEntry.ListStatus.Status, malEntry.ListStatus.IsRewatching)
		progress := malEntry.ListStatus.NumEpisodesWatched
		score := float64(malEntry.ListStatus.Score)
		repeat := 0
		if malEntry.ListStatus.IsRewatching {
			repeat = 1
		}

		entry := &anilist.AnimeCollection_MediaListCollection_Lists_Entries{
			ID:       malEntry.Node.ID,
			Media:    convertAnime(malEntry),
			Progress: &progress,
			Score:    &score,
			Repeat:   &repeat,
			Status:   &status,
		}
		listsByStatus[status].Entries = append(listsByStatus[status].Entries, entry)
	}

	lists := make([]*anilist.AnimeCollection_MediaListCollection_Lists, 0, len(listsByStatus))
	for _, status := range []anilist.MediaListStatus{
		anilist.MediaListStatusCurrent,
		anilist.MediaListStatusCompleted,
		anilist.MediaListStatusPaused,
		anilist.MediaListStatusDropped,
		anilist.MediaListStatusPlanning,
		anilist.MediaListStatusRepeating,
	} {
		if len(listsByStatus[status].Entries) > 0 {
			lists = append(lists, listsByStatus[status])
		}
	}

	return &anilist.AnimeCollection{
		MediaListCollection: &anilist.AnimeCollection_MediaListCollection{
			Lists: lists,
		},
	}
}

func ConvertBasicAnime(anime *mal.BasicAnime) *anilist.BaseAnime {
	if anime == nil {
		return nil
	}

	id := anime.ID
	siteURL := fmt.Sprintf("https://myanimelist.net/anime/%d", id)
	mediaType := anilist.MediaTypeAnime
	format := mapAnimeFormat(anime.MediaType)
	status := mapAnimeStatus(anime.Status)
	isAdult := anime.NSFW != "" && anime.NSFW != "white"
	episodes := anime.NumEpisodes
	meanScore := int(anime.Mean * 10)
	title := anime.Title
	description := anime.Synopsis

	media := &anilist.BaseAnime{
		ID:          id,
		IDMal:       &id,
		SiteURL:     &siteURL,
		Status:      &status,
		Type:        &mediaType,
		Format:      &format,
		IsAdult:     &isAdult,
		Title:       &anilist.BaseAnime_Title{Romaji: &title, UserPreferred: &title},
		CoverImage:  &anilist.BaseAnime_CoverImage{},
		Description: &description,
	}

	if anime.AlternativeTitles.En != "" {
		media.Title.English = &anime.AlternativeTitles.En
	}
	if anime.AlternativeTitles.Ja != "" {
		media.Title.Native = &anime.AlternativeTitles.Ja
	}
	if anime.MainPicture.Large != "" {
		media.CoverImage.Large = &anime.MainPicture.Large
		media.CoverImage.ExtraLarge = &anime.MainPicture.Large
	}
	if anime.MainPicture.Medium != "" {
		media.CoverImage.Medium = &anime.MainPicture.Medium
	}
	if episodes > 0 {
		media.Episodes = &episodes
	}
	if meanScore > 0 {
		media.MeanScore = &meanScore
	}
	if anime.StartSeason.Year > 0 {
		media.SeasonYear = &anime.StartSeason.Year
		season := mapAnimeSeason(anime.StartSeason.Season)
		media.Season = &season
	}
	if anime.StartDate != "" {
		media.StartDate = parseAnimeDate(anime.StartDate)
	}
	if anime.EndDate != "" {
		media.EndDate = parseAnimeEndDate(anime.EndDate)
	}
	for _, synonym := range anime.AlternativeTitles.Synonyms {
		s := synonym
		media.Synonyms = append(media.Synonyms, &s)
	}

	return media
}

func ConvertAnimeDetails(anime *mal.BasicAnime) *anilist.AnimeDetailsById_Media {
	if anime == nil {
		return nil
	}

	siteURL := fmt.Sprintf("https://myanimelist.net/anime/%d", anime.ID)
	meanScore := int(anime.Mean * 10)
	popularity := anime.Popularity
	description := anime.Synopsis
	details := &anilist.AnimeDetailsById_Media{
		ID:          anime.ID,
		SiteURL:     &siteURL,
		Description: &description,
		MeanScore:   &meanScore,
		Popularity:  &popularity,
	}

	if meanScore > 0 {
		details.AverageScore = &meanScore
	}
	if anime.StartDate != "" {
		details.StartDate = parseAnimeDetailsDate(anime.StartDate)
	}
	if anime.EndDate != "" {
		details.EndDate = parseAnimeDetailsEndDate(anime.EndDate)
	}
	if anime.Rank > 0 {
		allTime := true
		rankContext := "highest rated all time"
		format := mapAnimeFormat(anime.MediaType)
		details.Rankings = append(details.Rankings, &anilist.AnimeDetailsById_Media_Rankings{
			AllTime: &allTime,
			Context: rankContext,
			Format:  format,
			Rank:    anime.Rank,
			Type:    anilist.MediaRankTypeRated,
		})
	}

	return details
}

func convertAnime(malEntry *mal.AnimeListEntry) *anilist.BaseAnime {
	id := malEntry.Node.ID
	siteURL := fmt.Sprintf("https://myanimelist.net/anime/%d", id)
	mediaType := anilist.MediaTypeAnime
	format := mapAnimeFormat(malEntry.Node.MediaType)
	status := mapAnimeStatus(malEntry.Node.Status)
	isAdult := malEntry.Node.NSFW != "" && malEntry.Node.NSFW != "white"
	episodes := malEntry.Node.NumEpisodes
	meanScore := int(malEntry.Node.Mean * 10)
	title := malEntry.Node.Title
	englishTitle := malEntry.Node.AlternativeTitles.En
	nativeTitle := malEntry.Node.AlternativeTitles.Ja
	description := malEntry.Node.Synopsis

	media := &anilist.BaseAnime{
		ID:          id,
		IDMal:       &id,
		SiteURL:     &siteURL,
		Status:      &status,
		Type:        &mediaType,
		Format:      &format,
		IsAdult:     &isAdult,
		Title:       &anilist.BaseAnime_Title{Romaji: &title, UserPreferred: &title},
		CoverImage:  &anilist.BaseAnime_CoverImage{},
		Description: &description,
	}

	if englishTitle != "" {
		media.Title.English = &englishTitle
	}
	if nativeTitle != "" {
		media.Title.Native = &nativeTitle
	}
	if malEntry.Node.MainPicture.Large != "" {
		media.CoverImage.Large = &malEntry.Node.MainPicture.Large
		media.CoverImage.ExtraLarge = &malEntry.Node.MainPicture.Large
	}
	if malEntry.Node.MainPicture.Medium != "" {
		media.CoverImage.Medium = &malEntry.Node.MainPicture.Medium
	}
	if episodes > 0 {
		media.Episodes = &episodes
	}
	if meanScore > 0 {
		media.MeanScore = &meanScore
	}
	if malEntry.Node.StartSeason.Year > 0 {
		media.SeasonYear = &malEntry.Node.StartSeason.Year
		season := mapAnimeSeason(malEntry.Node.StartSeason.Season)
		media.Season = &season
	}
	if malEntry.Node.StartDate != "" {
		media.StartDate = parseAnimeDate(malEntry.Node.StartDate)
	}
	if malEntry.Node.EndDate != "" {
		media.EndDate = parseAnimeEndDate(malEntry.Node.EndDate)
	}
	if len(malEntry.Node.AlternativeTitles.Synonyms) > 0 {
		for _, synonym := range malEntry.Node.AlternativeTitles.Synonyms {
			s := synonym
			media.Synonyms = append(media.Synonyms, &s)
		}
	}

	return media
}

func animeListStatusName(status anilist.MediaListStatus) string {
	switch status {
	case anilist.MediaListStatusCurrent:
		return "Watching"
	case anilist.MediaListStatusCompleted:
		return "Completed"
	case anilist.MediaListStatusPaused:
		return "Paused"
	case anilist.MediaListStatusDropped:
		return "Dropped"
	case anilist.MediaListStatusRepeating:
		return "Repeating"
	default:
		return "Planning"
	}
}

func mapAnimeListStatus(status mal.MediaListStatus, isRewatching bool) anilist.MediaListStatus {
	if isRewatching {
		return anilist.MediaListStatusRepeating
	}
	switch status {
	case mal.MediaListStatusWatching:
		return anilist.MediaListStatusCurrent
	case mal.MediaListStatusCompleted:
		return anilist.MediaListStatusCompleted
	case mal.MediaListStatusOnHold:
		return anilist.MediaListStatusPaused
	case mal.MediaListStatusDropped:
		return anilist.MediaListStatusDropped
	default:
		return anilist.MediaListStatusPlanning
	}
}

func mapAnimeFormat(mediaType mal.MediaType) anilist.MediaFormat {
	switch mediaType {
	case mal.MediaTypeMovie:
		return anilist.MediaFormatMovie
	case mal.MediaTypeOVA:
		return anilist.MediaFormatOva
	case mal.MediaTypeONA:
		return anilist.MediaFormatOna
	case mal.MediaTypeSpecial:
		return anilist.MediaFormatSpecial
	case mal.MediaTypeMusic:
		return anilist.MediaFormatMusic
	default:
		return anilist.MediaFormatTv
	}
}

func mapAnimeStatus(status mal.MediaStatus) anilist.MediaStatus {
	switch status {
	case mal.MediaStatusCurrentlyAiring:
		return anilist.MediaStatusReleasing
	case mal.MediaStatusNotYetAired:
		return anilist.MediaStatusNotYetReleased
	default:
		return anilist.MediaStatusFinished
	}
}

func mapAnimeSeason(season string) anilist.MediaSeason {
	switch strings.ToLower(season) {
	case "winter":
		return anilist.MediaSeasonWinter
	case "spring":
		return anilist.MediaSeasonSpring
	case "summer":
		return anilist.MediaSeasonSummer
	case "fall":
		return anilist.MediaSeasonFall
	default:
		return anilist.MediaSeasonWinter
	}
}

func parseAnimeDate(value string) *anilist.BaseAnime_StartDate {
	parts := strings.Split(value, "-")
	date := &anilist.BaseAnime_StartDate{}
	if len(parts) > 0 {
		date.Year = parseIntPtr(parts[0])
	}
	if len(parts) > 1 {
		date.Month = parseIntPtr(parts[1])
	}
	if len(parts) > 2 {
		date.Day = parseIntPtr(parts[2])
	}
	return date
}

func parseAnimeEndDate(value string) *anilist.BaseAnime_EndDate {
	parts := strings.Split(value, "-")
	date := &anilist.BaseAnime_EndDate{}
	if len(parts) > 0 {
		date.Year = parseIntPtr(parts[0])
	}
	if len(parts) > 1 {
		date.Month = parseIntPtr(parts[1])
	}
	if len(parts) > 2 {
		date.Day = parseIntPtr(parts[2])
	}
	return date
}

func parseAnimeDetailsDate(value string) *anilist.AnimeDetailsById_Media_StartDate {
	parts := strings.Split(value, "-")
	date := &anilist.AnimeDetailsById_Media_StartDate{}
	if len(parts) > 0 {
		date.Year = parseIntPtr(parts[0])
	}
	if len(parts) > 1 {
		date.Month = parseIntPtr(parts[1])
	}
	if len(parts) > 2 {
		date.Day = parseIntPtr(parts[2])
	}
	return date
}

func parseAnimeDetailsEndDate(value string) *anilist.AnimeDetailsById_Media_EndDate {
	parts := strings.Split(value, "-")
	date := &anilist.AnimeDetailsById_Media_EndDate{}
	if len(parts) > 0 {
		date.Year = parseIntPtr(parts[0])
	}
	if len(parts) > 1 {
		date.Month = parseIntPtr(parts[1])
	}
	if len(parts) > 2 {
		date.Day = parseIntPtr(parts[2])
	}
	return date
}

func parseIntPtr(value string) *int {
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return nil
	}
	return &parsed
}
