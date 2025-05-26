package transcode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/narwhalmedia/narwhal/internal/domain/transcode"
)

var (
	progressRegex = regexp.MustCompile(`time=(\d+:\d+:\d+\.\d+)`)
	durationRegex = regexp.MustCompile(`Duration: (\d+:\d+:\d+\.\d+)`)
)

type FFmpegTranscoder struct {
	logger   *zap.Logger
	ffmpeg   string
	ffprobe  string
	jobs     map[uuid.UUID]*jobContext
	mu       sync.RWMutex
}

type jobContext struct {
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	progress chan transcode.Progress
}

func NewFFmpegTranscoder(logger *zap.Logger) (*FFmpegTranscoder, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	return &FFmpegTranscoder{
		logger:  logger,
		ffmpeg:  ffmpeg,
		ffprobe: ffprobe,
		jobs:    make(map[uuid.UUID]*jobContext),
	}, nil
}

func (t *FFmpegTranscoder) Transcode(ctx context.Context, job *transcode.Job, progress chan<- transcode.Progress) error {
	// Store job context
	jobCtx, cancel := context.WithCancel(ctx)
	t.mu.Lock()
	t.jobs[job.ID()] = &jobContext{
		cancel:   cancel,
		progress: make(chan transcode.Progress, 1),
	}
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.jobs, job.ID())
		t.mu.Unlock()
		close(progress)
	}()

	// Get input metadata
	metadata, err := t.getMetadata(job.InputPath())
	if err != nil {
		return fmt.Errorf("failed to get input metadata: %w", err)
	}

	// Create output directory
	outputDir := filepath.Dir(job.OutputPath())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Transcode based on profile
	switch job.Profile() {
	case transcode.ProfileHLS:
		return t.transcodeHLS(jobCtx, job, metadata, progress)
	case transcode.ProfileMP4:
		return t.transcodeMP4(jobCtx, job, metadata, progress)
	case transcode.ProfileWebM:
		return t.transcodeWebM(jobCtx, job, metadata, progress)
	default:
		return fmt.Errorf("unsupported profile: %s", job.Profile())
	}
}

func (t *FFmpegTranscoder) transcodeHLS(ctx context.Context, job *transcode.Job, metadata *videoMetadata, progress chan<- transcode.Progress) error {
	outputDir := strings.TrimSuffix(job.OutputPath(), ".m3u8")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create HLS output directory: %w", err)
	}

	// Get HLS variants from options
	variants := transcode.DefaultHLSVariants

	// Filter variants based on input resolution
	filteredVariants := t.filterVariants(variants, metadata.width, metadata.height)

	// Create master playlist
	masterPlaylist := filepath.Join(outputDir, "master.m3u8")
	
	// Build FFmpeg command for all variants
	args := []string{
		"-i", job.InputPath(),
		"-hide_banner",
		"-progress", "pipe:1",
	}

	// Add variant outputs
	for i, variant := range filteredVariants {
		variantDir := filepath.Join(outputDir, fmt.Sprintf("variant_%d", i))
		if err := os.MkdirAll(variantDir, 0755); err != nil {
			return fmt.Errorf("failed to create variant directory: %w", err)
		}

		// Video encoding
		args = append(args,
			"-map", "0:v",
			"-c:v", "libx264",
			"-preset", t.getPreset(job.Options()),
			"-crf", fmt.Sprintf("%d", variant.CRF),
			"-g", "48",
			"-keyint_min", "48",
			"-sc_threshold", "0",
			"-b:v", fmt.Sprintf("%dk", variant.Bitrate),
			"-maxrate", fmt.Sprintf("%dk", int(float64(variant.Bitrate)*1.5)),
			"-bufsize", fmt.Sprintf("%dk", variant.Bitrate*2),
			"-vf", fmt.Sprintf("scale=%d:%d", variant.Width, variant.Height),
		)

		// Audio encoding
		args = append(args,
			"-map", "0:a",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ac", "2",
		)

		// HLS options
		args = append(args,
			"-f", "hls",
			"-hls_time", "6",
			"-hls_list_size", "0",
			"-hls_segment_filename", filepath.Join(variantDir, "segment_%03d.ts"),
			filepath.Join(variantDir, "playlist.m3u8"),
		)
	}

	// Execute FFmpeg
	cmd := exec.CommandContext(ctx, t.ffmpeg, args...)
	
	t.mu.Lock()
	if jobCtx, ok := t.jobs[job.ID()]; ok {
		jobCtx.cmd = cmd
	}
	t.mu.Unlock()

	// Capture progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Monitor progress
	go t.monitorProgress(stdout, metadata.duration, progress)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return transcode.ErrJobCancelled
		}
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	// Create master playlist
	if err := t.createMasterPlaylist(masterPlaylist, outputDir, filteredVariants); err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}

	return nil
}

func (t *FFmpegTranscoder) transcodeMP4(ctx context.Context, job *transcode.Job, metadata *videoMetadata, progress chan<- transcode.Progress) error {
	args := []string{
		"-i", job.InputPath(),
		"-hide_banner",
		"-progress", "pipe:1",
		"-c:v", "libx264",
		"-preset", t.getPreset(job.Options()),
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		job.OutputPath(),
	}

	cmd := exec.CommandContext(ctx, t.ffmpeg, args...)
	
	t.mu.Lock()
	if jobCtx, ok := t.jobs[job.ID()]; ok {
		jobCtx.cmd = cmd
	}
	t.mu.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	go t.monitorProgress(stdout, metadata.duration, progress)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return transcode.ErrJobCancelled
		}
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

func (t *FFmpegTranscoder) transcodeWebM(ctx context.Context, job *transcode.Job, metadata *videoMetadata, progress chan<- transcode.Progress) error {
	args := []string{
		"-i", job.InputPath(),
		"-hide_banner",
		"-progress", "pipe:1",
		"-c:v", "libvpx-vp9",
		"-crf", "30",
		"-b:v", "0",
		"-c:a", "libopus",
		"-b:a", "128k",
		"-y",
		job.OutputPath(),
	}

	cmd := exec.CommandContext(ctx, t.ffmpeg, args...)
	
	t.mu.Lock()
	if jobCtx, ok := t.jobs[job.ID()]; ok {
		jobCtx.cmd = cmd
	}
	t.mu.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	go t.monitorProgress(stdout, metadata.duration, progress)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return transcode.ErrJobCancelled
		}
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

func (t *FFmpegTranscoder) Cancel(ctx context.Context, jobID uuid.UUID) error {
	t.mu.RLock()
	jobCtx, ok := t.jobs[jobID]
	t.mu.RUnlock()

	if !ok {
		return transcode.ErrJobNotFound
	}

	jobCtx.cancel()
	return nil
}

func (t *FFmpegTranscoder) GetCapabilities() transcode.Capabilities {
	return transcode.Capabilities{
		SupportedProfiles: []transcode.Profile{
			transcode.ProfileHLS,
			transcode.ProfileMP4,
			transcode.ProfileWebM,
		},
		SupportedCodecs: []string{
			"h264", "h265", "vp8", "vp9", "av1",
		},
		MaxResolution: transcode.Resolution{
			Width:  7680, // 8K
			Height: 4320,
		},
		HardwareAcceleration: false, // Can be extended to support NVENC, QSV, etc.
	}
}

func (t *FFmpegTranscoder) monitorProgress(stdout io.ReadCloser, totalDuration time.Duration, progress chan<- transcode.Progress) {
	defer stdout.Close()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse progress from FFmpeg output
		if strings.Contains(line, "time=") {
			matches := progressRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				current := parseDuration(matches[1])
				if totalDuration > 0 {
					percentage := float64(current) / float64(totalDuration) * 100
					progress <- transcode.Progress{
						Percent:       percentage,
						CurrentTime:   current,
						TotalDuration: totalDuration,
						Speed:         1.0, // TODO: Parse actual speed
					}
				}
			}
		}
	}
}

func (t *FFmpegTranscoder) getMetadata(inputPath string) (*videoMetadata, error) {
	cmd := exec.Command(t.ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Find video stream
	var videoStream *ffprobeStream
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			videoStream = &stream
			break
		}
	}

	if videoStream == nil {
		return nil, fmt.Errorf("no video stream found")
	}

	// Parse duration
	duration, _ := strconv.ParseFloat(probe.Format.Duration, 64)

	return &videoMetadata{
		width:    videoStream.Width,
		height:   videoStream.Height,
		duration: time.Duration(duration * float64(time.Second)),
		codec:    videoStream.CodecName,
		bitrate:  probe.Format.BitRate,
	}, nil
}

func (t *FFmpegTranscoder) filterVariants(variants []transcode.HLSVariant, width, height int) []transcode.HLSVariant {
	var filtered []transcode.HLSVariant
	for _, v := range variants {
		if v.Width <= width && v.Height <= height {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 && len(variants) > 0 {
		// Include at least the lowest quality variant
		filtered = append(filtered, variants[len(variants)-1])
	}
	return filtered
}

func (t *FFmpegTranscoder) createMasterPlaylist(path, outputDir string, variants []transcode.HLSVariant) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "#EXTM3U")
	fmt.Fprintln(file, "#EXT-X-VERSION:3")

	for i, variant := range variants {
		fmt.Fprintf(file, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n",
			variant.Bitrate*1000, variant.Width, variant.Height)
		fmt.Fprintf(file, "variant_%d/playlist.m3u8\n", i)
	}

	return nil
}

func (t *FFmpegTranscoder) getPreset(options transcode.Options) string {
	if options.Preset != "" {
		return options.Preset
	}
	return "medium"
}

type videoMetadata struct {
	width    int
	height   int
	duration time.Duration
	codec    string
	bitrate  string
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecName string `json:"codec_name"`
	CodecType string `json:"codec_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

func parseDuration(s string) time.Duration {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.ParseFloat(parts[2], 64)

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds*float64(time.Second))
}