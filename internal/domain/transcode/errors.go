package transcode

import "errors"

var (
	// Job errors
	ErrJobNotFound        = errors.New("job not found")
	ErrJobAlreadyStarted  = errors.New("job already started")
	ErrJobNotStarted      = errors.New("job not started")
	ErrJobCompleted       = errors.New("job already completed")
	ErrJobFailed          = errors.New("job failed")
	ErrJobCancelled       = errors.New("job cancelled")
	ErrInvalidJobStatus   = errors.New("invalid job status")
	ErrInvalidProfile     = errors.New("invalid transcode profile")

	// Storage errors
	ErrStorageKeyNotFound = errors.New("storage key not found")
	ErrStorageFailure     = errors.New("storage operation failed")

	// Transcoder errors
	ErrTranscoderNotFound = errors.New("transcoder not found")
	ErrUnsupportedProfile = errors.New("unsupported transcode profile")
	ErrUnsupportedCodec   = errors.New("unsupported codec")
)