package handler

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/narwhalmedia/narwhal/internal/library/constants"
	commonpb "github.com/narwhalmedia/narwhal/pkg/common/v1"
	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

// convertMediaType converts proto media type to string.
func convertMediaType(t commonpb.MediaType) string {
	switch t {
	case commonpb.MediaType_MEDIA_TYPE_MOVIE:
		return "movie"
	case commonpb.MediaType_MEDIA_TYPE_SERIES:
		return "tv_show"
	case commonpb.MediaType_MEDIA_TYPE_MUSIC:
		return "music"
	default:
		return "movie"
	}
}

// convertMediaTypeToProto converts string media type to proto media type.
func convertMediaTypeToProto(t string) commonpb.MediaType {
	switch t {
	case "movie":
		return commonpb.MediaType_MEDIA_TYPE_MOVIE
	case "tv_show", "series":
		return commonpb.MediaType_MEDIA_TYPE_SERIES
	case "music":
		return commonpb.MediaType_MEDIA_TYPE_MUSIC
	default:
		return commonpb.MediaType_MEDIA_TYPE_UNSPECIFIED
	}
}

// convertDomainLibraryToProto converts domain library to proto library.
func convertLibraryToProto(lib *models.Library) *librarypb.Library {
	proto := &librarypb.Library{
		Id:                  lib.ID.String(),
		Name:                lib.Name,
		Path:                lib.Path,
		Type:                convertMediaTypeToProto(string(lib.Type)),
		AutoScan:            lib.Enabled,
		ScanIntervalMinutes: int32(lib.ScanInterval / constants.SecondsToMinutes), // Convert from seconds to minutes
		Created:             timestamppb.New(lib.CreatedAt),
		Updated:             timestamppb.New(lib.UpdatedAt),
	}

	if lib.LastScanAt != nil {
		proto.LastScanned = timestamppb.New(*lib.LastScanAt)
	}

	return proto
}

// convertMediaToProto converts domain media to proto media.
func convertMediaToProto(media *models.Media, includeMetadata, includeEpisodes bool) *librarypb.Media {
	protoMedia := &librarypb.Media{
		Id:              media.ID.String(),
		Title:           media.Title,
		Type:            convertMediaTypeToProtoFromMediaType(media.Type),
		Path:            media.FilePath,
		SizeBytes:       media.FileSize,
		DurationSeconds: int32(media.Runtime * 60),
		Resolution:      media.Resolution,
		Codec:           media.VideoCodec,
		Bitrate:         int32(media.Bitrate),
		Added:           timestamppb.New(media.CreatedAt),
		Modified:        timestamppb.New(media.UpdatedAt),
		LastScanned:     timestamppb.New(media.UpdatedAt),
	}

	if includeEpisodes && len(media.Episodes) > 0 {
		protoMedia.Episodes = make([]*librarypb.Episode, len(media.Episodes))
		for i, ep := range media.Episodes {
			protoMedia.Episodes[i] = convertEpisodeToProto(ep)
		}
	}

	return protoMedia
}

// convertMediaTypeToProtoFromMediaType converts models.MediaType to proto.
func convertMediaTypeToProtoFromMediaType(t models.MediaType) commonpb.MediaType {
	switch t {
	case models.MediaTypeMovie:
		return commonpb.MediaType_MEDIA_TYPE_MOVIE
	case models.MediaTypeSeries:
		return commonpb.MediaType_MEDIA_TYPE_SERIES
	case models.MediaTypeMusic:
		return commonpb.MediaType_MEDIA_TYPE_MUSIC
	default:
		return commonpb.MediaType_MEDIA_TYPE_UNSPECIFIED
	}
}

// convertMetadataToProto converts domain metadata to proto metadata.
func convertMetadataToProto(metadata *models.Metadata) *librarypb.Metadata {
	proto := &librarypb.Metadata{
		Id:          metadata.ID.String(),
		MediaId:     metadata.MediaID.String(),
		ImdbId:      metadata.IMDBID,
		TmdbId:      metadata.TMDBID,
		TvdbId:      metadata.TVDBID,
		Description: metadata.Description,
		Rating:      metadata.Rating,
		Genres:      metadata.Genres,
		Cast:        metadata.Cast,
		Directors:   metadata.Directors,
		PosterUrl:   metadata.PosterURL,
		BackdropUrl: metadata.BackdropURL,
		TrailerUrl:  metadata.TrailerURL,
	}

	// Parse ReleaseDate string to time.Time if not empty
	if metadata.ReleaseDate != "" {
		// Try common date formats
		for _, format := range []string{"2006-01-02", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00"} {
			if t, err := time.Parse(format, metadata.ReleaseDate); err == nil {
				proto.ReleaseDate = timestamppb.New(t)
				break
			}
		}
	}

	return proto
}

// convertEpisodeToProto converts domain episode to proto episode.
func convertEpisodeToProto(episode *models.Episode) *librarypb.Episode {
	proto := &librarypb.Episode{
		Id:              episode.ID.String(),
		MediaId:         episode.MediaID.String(),
		SeasonNumber:    int32(episode.SeasonNumber),
		EpisodeNumber:   int32(episode.EpisodeNumber),
		Title:           episode.Title,
		Path:            episode.FilePath,
		DurationSeconds: int32(episode.Runtime * 60),
		Added:           timestamppb.New(episode.CreatedAt),
	}

	if episode.AirDate != nil && !episode.AirDate.IsZero() {
		proto.AirDate = timestamppb.New(*episode.AirDate)
	}

	return proto
}
