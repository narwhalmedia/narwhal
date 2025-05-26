package download

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/download"
)

// TorrentDownloader implements torrent download functionality
type TorrentDownloader struct {
	client   *torrent.Client
	logger   *zap.Logger
	dataDir  string
}

// NewTorrentDownloader creates a new torrent downloader
func NewTorrentDownloader(dataDir string, logger *zap.Logger) (*TorrentDownloader, error) {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dataDir
	cfg.Seed = false // Don't seed after download
	cfg.Debug = false
	
	// Configure DHT
	cfg.DHTConfig.StartingNodes = []string{
		"router.utorrent.com:6881",
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
	}

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	return &TorrentDownloader{
		client:  client,
		logger:  logger.Named("torrent-downloader"),
		dataDir: dataDir,
	}, nil
}

// Download starts downloading from the source (magnet link or torrent file)
func (d *TorrentDownloader) Download(ctx context.Context, source string, destination io.Writer, progress chan<- download.Progress) error {
	// Add torrent
	t, err := d.addTorrent(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to add torrent: %w", err)
	}

	// Wait for torrent info to be available
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		t.Drop()
		return ctx.Err()
	case <-time.After(30 * time.Second):
		t.Drop()
		return fmt.Errorf("timeout waiting for torrent metadata")
	}

	// Start download
	t.DownloadAll()

	// Monitor progress
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Drop()
			return ctx.Err()
		case <-ticker.C:
			stats := t.Stats()
			if progress != nil {
				prog := download.Progress{
					BytesDownloaded: stats.BytesReadData.Int64(),
					TotalBytes:      t.Length(),
					Speed:           int64(stats.ActivePeers) * 1024, // Approximate
					Seeders:         stats.ConnectedSeeders,
					Leechers:        stats.ActivePeers - stats.ConnectedSeeders,
				}
				
				// Calculate ETA
				if prog.Speed > 0 && t.Length() > 0 {
					remaining := t.Length() - stats.BytesReadData.Int64()
					prog.ETA = time.Duration(remaining/prog.Speed) * time.Second
				}

				select {
				case progress <- prog:
				default:
				}
			}

			// Check if download is complete
			if t.Complete.Bool() {
				// Copy to destination if needed
				if destination != nil {
					if err := d.copyTorrentFiles(t, destination); err != nil {
						t.Drop()
						return fmt.Errorf("failed to copy torrent files: %w", err)
					}
				}
				t.Drop()
				return nil
			}
		}
	}
}

// Resume resumes a download from the given offset
func (d *TorrentDownloader) Resume(ctx context.Context, source string, destination io.WriteSeeker, offset int64, progress chan<- download.Progress) error {
	// Torrents automatically resume from where they left off
	// The offset parameter is ignored as torrent protocol handles this internally
	return d.Download(ctx, source, destination, progress)
}

// GetMetadata fetches metadata about the download
func (d *TorrentDownloader) GetMetadata(ctx context.Context, source string) (*download.Metadata, error) {
	t, err := d.addTorrent(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}
	defer t.Drop()

	// Wait for info
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for torrent metadata")
	}

	info := t.Info()
	metadata := &download.Metadata{
		FileName:    info.Name,
		ContentType: "application/x-bittorrent",
		InfoHash:    t.InfoHash().String(),
		Headers:     make(map[string]string),
	}

	// Add torrent-specific metadata
	metadata.Headers["pieces"] = fmt.Sprintf("%d", info.NumPieces())
	metadata.Headers["piece_length"] = fmt.Sprintf("%d", info.PieceLength)
	metadata.Headers["total_length"] = fmt.Sprintf("%d", t.Length())

	return metadata, nil
}

// Close closes the torrent client
func (d *TorrentDownloader) Close() error {
	return d.client.Close()
}

// addTorrent adds a torrent from magnet link or file
func (d *TorrentDownloader) addTorrent(ctx context.Context, source string) (*torrent.Torrent, error) {
	// Check if source is a magnet link
	if isMagnetLink(source) {
		return d.client.AddMagnet(source)
	}

	// Try as torrent file path
	metaInfo, err := metainfo.LoadFromFile(source)
	if err != nil {
		// Try as torrent file URL
		// TODO: Download torrent file from URL
		return nil, fmt.Errorf("failed to load torrent: %w", err)
	}

	return d.client.AddTorrent(metaInfo)
}

// copyTorrentFiles copies torrent files to destination
func (d *TorrentDownloader) copyTorrentFiles(t *torrent.Torrent, destination io.Writer) error {
	// For single file torrents
	if len(t.Files()) == 1 {
		file := t.Files()[0]
		reader := file.NewReader()
		reader.SetResponsive()
		defer reader.Close()

		_, err := io.Copy(destination, reader)
		return err
	}

	// For multi-file torrents, we might need to handle differently
	// This is a simplified implementation
	for _, file := range t.Files() {
		reader := file.NewReader()
		reader.SetResponsive()
		
		if _, err := io.Copy(destination, reader); err != nil {
			reader.Close()
			return err
		}
		reader.Close()
	}

	return nil
}

// isMagnetLink checks if the source is a magnet link
func isMagnetLink(source string) bool {
	return len(source) >= 8 && source[:8] == "magnet:?"
}

// TorrentManager manages multiple torrent downloads
type TorrentManager struct {
	downloader *TorrentDownloader
	downloads  map[string]*torrent.Torrent
	logger     *zap.Logger
}

// NewTorrentManager creates a new torrent manager
func NewTorrentManager(dataDir string, logger *zap.Logger) (*TorrentManager, error) {
	downloader, err := NewTorrentDownloader(dataDir, logger)
	if err != nil {
		return nil, err
	}

	return &TorrentManager{
		downloader: downloader,
		downloads:  make(map[string]*torrent.Torrent),
		logger:     logger.Named("torrent-manager"),
	}, nil
}

// StartDownload starts a new torrent download
func (tm *TorrentManager) StartDownload(ctx context.Context, id, source string, progress chan<- download.Progress) error {
	t, err := tm.downloader.addTorrent(ctx, source)
	if err != nil {
		return err
	}

	tm.downloads[id] = t
	
	// Start download in background
	go func() {
		<-t.GotInfo()
		t.DownloadAll()
		
		// Monitor progress
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if t.Complete.Bool() {
				delete(tm.downloads, id)
				return
			}

			if progress != nil {
				stats := t.Stats()
				prog := download.Progress{
					BytesDownloaded: stats.BytesReadData.Int64(),
					TotalBytes:      t.Length(),
					Speed:           int64(stats.ActivePeers) * 1024,
					Seeders:         stats.ConnectedSeeders,
					Leechers:        stats.ActivePeers - stats.ConnectedSeeders,
				}
				
				select {
				case progress <- prog:
				default:
				}
			}
		}
	}()

	return nil
}

// PauseDownload pauses a torrent download
func (tm *TorrentManager) PauseDownload(id string) error {
	t, ok := tm.downloads[id]
	if !ok {
		return fmt.Errorf("torrent not found: %s", id)
	}

	t.DisallowDataDownload()
	return nil
}

// ResumeDownload resumes a torrent download
func (tm *TorrentManager) ResumeDownload(id string) error {
	t, ok := tm.downloads[id]
	if !ok {
		return fmt.Errorf("torrent not found: %s", id)
	}

	t.AllowDataDownload()
	return nil
}

// CancelDownload cancels a torrent download
func (tm *TorrentManager) CancelDownload(id string) error {
	t, ok := tm.downloads[id]
	if !ok {
		return fmt.Errorf("torrent not found: %s", id)
	}

	t.Drop()
	delete(tm.downloads, id)
	return nil
}

// Close closes the torrent manager
func (tm *TorrentManager) Close() error {
	// Drop all torrents
	for id, t := range tm.downloads {
		t.Drop()
		delete(tm.downloads, id)
	}
	
	return tm.downloader.Close()
}