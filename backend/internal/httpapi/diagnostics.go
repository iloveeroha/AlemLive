package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type diagnosticCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func (s *Server) recordingDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	minioStatus := s.checkRecordingBucket(ctx)
	payload := map[string]any{
		"status": "ok",
		"checks": map[string]diagnosticCheck{
			"livekit_url":          configPresenceCheck(s.cfg.LiveKitURL, "LIVEKIT_URL is set", "LIVEKIT_URL is missing"),
			"livekit_credentials":  configPairCheck(s.cfg.LiveKitAPIKey, s.cfg.LiveKitSecret, "LiveKit API key/secret are set", "LIVEKIT_API_KEY or LIVEKIT_API_SECRET is missing"),
			"egress_enabled":       boolCheck(s.egress != nil && s.egress.Configured(), "LiveKit Egress is configured", "LiveKit Egress is disabled or missing required config"),
			"egress_webhook":       configPresenceCheck(s.cfg.LiveKitEgressWebhookURL, "LIVEKIT_EGRESS_WEBHOOK_URL is set", "LIVEKIT_EGRESS_WEBHOOK_URL is missing"),
			"minio_endpoint":       configPresenceCheck(s.cfg.LiveKitS3Endpoint, "LIVEKIT_S3_ENDPOINT is set", "LIVEKIT_S3_ENDPOINT is missing"),
			"minio_bucket_config":  configPresenceCheck(s.cfg.LiveKitS3Bucket, "LIVEKIT_S3_BUCKET is set", "LIVEKIT_S3_BUCKET is missing"),
			"minio_credentials":    configPairCheck(s.cfg.LiveKitS3AccessKey, s.cfg.LiveKitS3Secret, "MinIO/S3 credentials are set", "LIVEKIT_S3_ACCESS_KEY or LIVEKIT_S3_SECRET is missing"),
			"minio_bucket_reached": minioStatus,
			"stt":                  boolCheck(s.stt != nil && s.stt.Configured(), "STT is configured", "STT_BASE_URL/STT_API_KEY are not configured"),
			"llm":                  boolCheck(s.ai != nil && s.ai.Configured(), "LLM is configured", "LLM_BASE_URL/LLM_API_KEY are not configured"),
			"diarization":          boolCheck(s.diarizer != nil && s.diarizer.Configured(), "Diarization service is configured", "DIARIZATION_BASE_URL is not configured"),
			"participant_tracks":   participantTrackSupportCheck(s.cfg.RecordingMode),
		},
		"config": map[string]any{
			"livekitUrl":            redactedURL(s.cfg.LiveKitURL),
			"livekitPublicUrl":      redactedURL(s.cfg.LiveKitPublicURL),
			"egressEnabled":         s.cfg.LiveKitEgressEnabled,
			"egressWebhookUrl":      redactedURL(s.cfg.LiveKitEgressWebhookURL),
			"s3Endpoint":            redactedURL(s.cfg.LiveKitS3Endpoint),
			"s3Bucket":              s.cfg.LiveKitS3Bucket,
			"s3ForcePathStyle":      s.cfg.LiveKitS3ForcePathStyle,
			"recordingsStoragePath": s.cfg.RecordingsStoragePath,
			"reportsStoragePath":    s.cfg.ReportsStoragePath,
			"sttBaseUrl":            redactedURL(s.cfg.STTBaseURL),
			"sttModel":              s.cfg.STTModel,
			"llmBaseUrl":            redactedURL(s.cfg.LLMBaseURL),
			"llmModel":              s.cfg.LLMModel,
			"diarizationBaseUrl":    redactedURL(s.cfg.DiarizationBaseURL),
			"recordingMode":         s.cfg.RecordingMode,
			"recordingFallbackMode": s.cfg.RecordingFallbackMode,
			"diarizationFallback":   s.cfg.EnableDiarizationFallback,
			"speakerManualMapping":  s.cfg.EnableSpeakerManualMapping,
			"transcriptPreferNames": s.cfg.TranscriptPreferNames,
			"livekitApiKeyPresent":  s.cfg.LiveKitAPIKey != "",
			"livekitSecretPresent":  s.cfg.LiveKitSecret != "",
			"s3AccessKeyPresent":    s.cfg.LiveKitS3AccessKey != "",
			"s3SecretPresent":       s.cfg.LiveKitS3Secret != "",
			"sttApiKeyPresent":      s.cfg.STTAPIKey != "",
			"llmApiKeyPresent":      s.cfg.LLMAPIKey != "",
			"diarizationKeyPresent": s.cfg.DiarizationAPIKey != "",
			"chatHistoryBucket":     s.cfg.ChatHistoryS3Bucket,
			"demoReportsEnabled":    s.cfg.DemoReportsEnabled,
		},
	}
	if minioStatus.Status != "ok" {
		payload["status"] = "degraded"
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) checkRecordingBucket(ctx context.Context) diagnosticCheck {
	if s.cfg.LiveKitS3Endpoint == "" || s.cfg.LiveKitS3Bucket == "" || s.cfg.LiveKitS3AccessKey == "" || s.cfg.LiveKitS3Secret == "" {
		return diagnosticCheck{Status: "missing", Message: "MinIO/S3 endpoint, bucket or credentials are not configured"}
	}

	endpoint, secure, err := minioEndpointFromURL(s.cfg.LiveKitS3Endpoint)
	if err != nil {
		return diagnosticCheck{Status: "error", Message: "Invalid MinIO/S3 endpoint: " + err.Error()}
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(s.cfg.LiveKitS3AccessKey, s.cfg.LiveKitS3Secret, s.cfg.LiveKitS3SessionToken),
		Secure:       secure,
		Region:       s.cfg.LiveKitS3Region,
		BucketLookup: minioBucketLookupType(s.cfg.LiveKitS3ForcePathStyle),
	})
	if err != nil {
		return diagnosticCheck{Status: "error", Message: "Could not create MinIO/S3 client: " + err.Error()}
	}

	exists, err := client.BucketExists(ctx, s.cfg.LiveKitS3Bucket)
	if err != nil {
		return diagnosticCheck{Status: "error", Message: "Could not reach recording bucket: " + err.Error()}
	}
	if !exists {
		return diagnosticCheck{Status: "error", Message: "Recording bucket does not exist: " + s.cfg.LiveKitS3Bucket}
	}
	return diagnosticCheck{Status: "ok", Message: "Recording bucket is reachable"}
}

func minioEndpointFromURL(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	secure := true
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false, err
	}
	if parsed.Scheme == "" {
		return raw, true, nil
	}
	secure = parsed.Scheme == "https"
	return parsed.Host, secure, nil
}

func minioBucketLookupType(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func configPresenceCheck(value, okMessage, missingMessage string) diagnosticCheck {
	if strings.TrimSpace(value) == "" {
		return diagnosticCheck{Status: "missing", Message: missingMessage}
	}
	return diagnosticCheck{Status: "ok", Message: okMessage}
}

func configPairCheck(left, right, okMessage, missingMessage string) diagnosticCheck {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return diagnosticCheck{Status: "missing", Message: missingMessage}
	}
	return diagnosticCheck{Status: "ok", Message: okMessage}
}

func boolCheck(ok bool, okMessage, failMessage string) diagnosticCheck {
	if ok {
		return diagnosticCheck{Status: "ok", Message: okMessage}
	}
	return diagnosticCheck{Status: "missing", Message: failMessage}
}

func participantTrackSupportCheck(recordingMode string) diagnosticCheck {
	if strings.EqualFold(strings.TrimSpace(recordingMode), "participant_tracks") {
		return diagnosticCheck{
			Status:  "degraded",
			Message: "Participant track mode is configured, but processing still falls back to room composite until per-track egress files are fully processed.",
		}
	}
	return diagnosticCheck{
		Status:  "ok",
		Message: "Room composite recording is active. Speaker names come from participant metadata only when track/manual mapping is available; otherwise diarization uses Speaker labels.",
	}
}

func redactedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
