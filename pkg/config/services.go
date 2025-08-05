package config

import (
	"fmt"
	"time"
)

// LibraryConfig extends BaseConfig with library-specific settings
type LibraryConfig struct {
	BaseConfig `koanf:",squash"`
	Library    LibrarySettings `koanf:"library"`
}

// LibrarySettings contains library service specific settings
type LibrarySettings struct {
	ScanInterval      time.Duration `koanf:"scan_interval"`
	MaxConcurrentScan int           `koanf:"max_concurrent_scan"`
	FileExtensions    []string      `koanf:"file_extensions"`
	IgnorePatterns    []string      `koanf:"ignore_patterns"`
	ThumbnailSize     int           `koanf:"thumbnail_size"`
	EnableAutoScan    bool          `koanf:"enable_auto_scan"`
}

// Validate validates the library configuration
func (c *LibraryConfig) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	if c.Library.ScanInterval < time.Minute {
		return fmt.Errorf("scan interval must be at least 1 minute")
	}
	if c.Library.MaxConcurrentScan < 1 {
		return fmt.Errorf("max concurrent scan must be at least 1")
	}
	return nil
}

// UserConfig extends BaseConfig with user/auth-specific settings
type UserConfig struct {
	BaseConfig `koanf:",squash"`
	Auth       AuthSettings `koanf:"auth"`
}

// AuthSettings contains authentication specific settings
type AuthSettings struct {
	JWTSecret           string        `koanf:"jwt_secret"`
	JWTAccessExpiry     time.Duration `koanf:"jwt_access_expiry"`
	JWTRefreshExpiry    time.Duration `koanf:"jwt_refresh_expiry"`
	BCryptCost          int           `koanf:"bcrypt_cost"`
	SessionTimeout      time.Duration `koanf:"session_timeout"`
	MaxLoginAttempts    int           `koanf:"max_login_attempts"`
	LockoutDuration     time.Duration `koanf:"lockout_duration"`
	PasswordMinLength   int           `koanf:"password_min_length"`
	RequireEmailVerify  bool          `koanf:"require_email_verify"`
	EnableOAuth         bool          `koanf:"enable_oauth"`
	OAuthProviders      []string      `koanf:"oauth_providers"`
}

// Validate validates the user configuration
func (c *UserConfig) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.Auth.JWTAccessExpiry < time.Minute {
		return fmt.Errorf("JWT access expiry must be at least 1 minute")
	}
	if c.Auth.BCryptCost < 10 || c.Auth.BCryptCost > 31 {
		return fmt.Errorf("bcrypt cost must be between 10 and 31")
	}
	return nil
}

// StreamingConfig extends BaseConfig with streaming-specific settings
type StreamingConfig struct {
	BaseConfig `koanf:",squash"`
	Streaming  StreamingSettings `koanf:"streaming"`
}

// StreamingSettings contains streaming service specific settings
type StreamingSettings struct {
	TranscodingEnabled  bool              `koanf:"transcoding_enabled"`
	TranscodingProfiles []TranscodeProfile `koanf:"transcoding_profiles"`
	SegmentDuration     time.Duration     `koanf:"segment_duration"`
	BufferSize          int               `koanf:"buffer_size"`
	MaxConcurrentStreams int              `koanf:"max_concurrent_streams"`
	CachePath           string            `koanf:"cache_path"`
	CacheSize           int64             `koanf:"cache_size"` // in bytes
	EnableHLS           bool              `koanf:"enable_hls"`
	EnableDASH          bool              `koanf:"enable_dash"`
	HardwareAccel       string            `koanf:"hardware_accel"` // none, nvidia, intel, amd
}

// TranscodeProfile defines a transcoding profile
type TranscodeProfile struct {
	Name      string `koanf:"name"`
	VideoCodec string `koanf:"video_codec"`
	AudioCodec string `koanf:"audio_codec"`
	Bitrate   string `koanf:"bitrate"`
	Resolution string `koanf:"resolution"`
	Preset    string `koanf:"preset"`
}

// Validate validates the streaming configuration
func (c *StreamingConfig) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	if c.Streaming.SegmentDuration < time.Second {
		return fmt.Errorf("segment duration must be at least 1 second")
	}
	if c.Streaming.MaxConcurrentStreams < 1 {
		return fmt.Errorf("max concurrent streams must be at least 1")
	}
	return nil
}

// AcquisitionConfig extends BaseConfig with acquisition-specific settings
type AcquisitionConfig struct {
	BaseConfig  `koanf:",squash"`
	Acquisition AcquisitionSettings `koanf:"acquisition"`
}

// AcquisitionSettings contains acquisition service specific settings
type AcquisitionSettings struct {
	Indexers            []IndexerConfig `koanf:"indexers"`
	DownloadPath        string          `koanf:"download_path"`
	CompletedPath       string          `koanf:"completed_path"`
	MaxActiveDownloads  int             `koanf:"max_active_downloads"`
	DownloadTimeout     time.Duration   `koanf:"download_timeout"`
	RetryAttempts       int             `koanf:"retry_attempts"`
	RetryDelay          time.Duration   `koanf:"retry_delay"`
	MinFreeDiskSpace    int64           `koanf:"min_free_disk_space"` // in bytes
	PreferredQuality    []string        `koanf:"preferred_quality"`
	ExcludedKeywords    []string        `koanf:"excluded_keywords"`
	RequiredKeywords    []string        `koanf:"required_keywords"`
}

// IndexerConfig contains indexer configuration
type IndexerConfig struct {
	Name     string `koanf:"name"`
	Type     string `koanf:"type"` // torrent, usenet, etc
	URL      string `koanf:"url"`
	APIKey   string `koanf:"api_key"`
	Enabled  bool   `koanf:"enabled"`
	Priority int    `koanf:"priority"`
	RateLimit int   `koanf:"rate_limit"` // requests per minute
}

// Validate validates the acquisition configuration
func (c *AcquisitionConfig) Validate() error {
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}
	if c.Acquisition.DownloadPath == "" {
		return fmt.Errorf("download path is required")
	}
	if c.Acquisition.MaxActiveDownloads < 1 {
		return fmt.Errorf("max active downloads must be at least 1")
	}
	return nil
}

// GetDefaultLibraryConfig returns default library configuration
func GetDefaultLibraryConfig() *LibraryConfig {
	base := GetDefaults()
	base.Service.Name = "library"
	base.Service.Port = 8081
	base.Service.GRPCPort = 9091
	
	return &LibraryConfig{
		BaseConfig: *base,
		Library: LibrarySettings{
			ScanInterval:      30 * time.Minute,
			MaxConcurrentScan: 2,
			FileExtensions:    []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg"},
			IgnorePatterns:    []string{"sample", "trailer", "extra"},
			ThumbnailSize:     320,
			EnableAutoScan:    true,
		},
	}
}

// GetDefaultUserConfig returns default user configuration
func GetDefaultUserConfig() *UserConfig {
	base := GetDefaults()
	base.Service.Name = "user"
	base.Service.Port = 8082
	base.Service.GRPCPort = 9092
	
	return &UserConfig{
		BaseConfig: *base,
		Auth: AuthSettings{
			JWTSecret:          "", // Must be set via env or config
			JWTAccessExpiry:    15 * time.Minute,
			JWTRefreshExpiry:   7 * 24 * time.Hour,
			BCryptCost:         12,
			SessionTimeout:     30 * time.Minute,
			MaxLoginAttempts:   5,
			LockoutDuration:    15 * time.Minute,
			PasswordMinLength:  8,
			RequireEmailVerify: false,
			EnableOAuth:        false,
			OAuthProviders:     []string{},
		},
	}
}

// GetDefaultStreamingConfig returns default streaming configuration
func GetDefaultStreamingConfig() *StreamingConfig {
	base := GetDefaults()
	base.Service.Name = "streaming"
	base.Service.Port = 8083
	base.Service.GRPCPort = 9093
	
	return &StreamingConfig{
		BaseConfig: *base,
		Streaming: StreamingSettings{
			TranscodingEnabled: true,
			TranscodingProfiles: []TranscodeProfile{
				{
					Name:       "1080p",
					VideoCodec: "h264",
					AudioCodec: "aac",
					Bitrate:    "5000k",
					Resolution: "1920x1080",
					Preset:     "medium",
				},
				{
					Name:       "720p",
					VideoCodec: "h264",
					AudioCodec: "aac",
					Bitrate:    "2500k",
					Resolution: "1280x720",
					Preset:     "medium",
				},
			},
			SegmentDuration:      10 * time.Second,
			BufferSize:           1024 * 1024 * 10, // 10MB
			MaxConcurrentStreams: 10,
			CachePath:            "/tmp/narwhal/streaming",
			CacheSize:            1024 * 1024 * 1024 * 10, // 10GB
			EnableHLS:            true,
			EnableDASH:           false,
			HardwareAccel:        "none",
		},
	}
}

// GetDefaultAcquisitionConfig returns default acquisition configuration
func GetDefaultAcquisitionConfig() *AcquisitionConfig {
	base := GetDefaults()
	base.Service.Name = "acquisition"
	base.Service.Port = 8084
	base.Service.GRPCPort = 9094
	
	return &AcquisitionConfig{
		BaseConfig: *base,
		Acquisition: AcquisitionSettings{
			Indexers:           []IndexerConfig{},
			DownloadPath:       "/tmp/narwhal/downloads",
			CompletedPath:      "/tmp/narwhal/completed",
			MaxActiveDownloads: 3,
			DownloadTimeout:    2 * time.Hour,
			RetryAttempts:      3,
			RetryDelay:         5 * time.Minute,
			MinFreeDiskSpace:   1024 * 1024 * 1024, // 1GB
			PreferredQuality:   []string{"1080p", "720p"},
			ExcludedKeywords:   []string{"cam", "ts", "screener"},
			RequiredKeywords:   []string{},
		},
	}
}