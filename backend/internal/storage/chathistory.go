package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
)

// ChatMessage is a single turn in a Copilot conversation.
type ChatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// ChatSession is one saved AlemAI Copilot thread for a report.
type ChatSession struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Messages  []ChatMessage `json:"messages"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// ChatHistoryStore persists Copilot chat sessions in a MinIO/S3 bucket, one
// JSON object per report+owner containing the full list of saved sessions.
type ChatHistoryStore struct {
	client *minio.Client
	bucket string

	mu sync.Mutex
}

func NewChatHistoryStore(cfg config.Config) (*ChatHistoryStore, error) {
	if cfg.LiveKitS3Endpoint == "" || cfg.LiveKitS3AccessKey == "" || cfg.LiveKitS3Secret == "" {
		return nil, errors.New("S3/MinIO credentials are not configured")
	}

	endpoint := cfg.LiveKitS3Endpoint
	secure := true
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Scheme != "" {
		secure = parsed.Scheme == "https"
		endpoint = parsed.Host
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.LiveKitS3AccessKey, cfg.LiveKitS3Secret, cfg.LiveKitS3SessionToken),
		Secure:       secure,
		Region:       cfg.LiveKitS3Region,
		BucketLookup: bucketLookupType(cfg.LiveKitS3ForcePathStyle),
	})
	if err != nil {
		return nil, err
	}

	store := &ChatHistoryStore{client: client, bucket: cfg.ChatHistoryS3Bucket}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func bucketLookupType(forcePathStyle bool) minio.BucketLookupType {
	if forcePathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func (s *ChatHistoryStore) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: ""})
}

func (s *ChatHistoryStore) objectKey(reportID, owner string) string {
	return "chat-history/" + url.PathEscape(reportID) + "/" + url.PathEscape(owner) + ".json"
}

// ListSessions returns the saved Copilot chat sessions for a report, newest first.
func (s *ChatHistoryStore) ListSessions(ctx context.Context, reportID, owner string) ([]ChatSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readSessionsLocked(ctx, reportID, owner)
}

// UpsertSession creates or updates a chat session and returns the full,
// up-to-date session list for the report.
func (s *ChatHistoryStore) UpsertSession(ctx context.Context, reportID, owner string, session ChatSession) ([]ChatSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.readSessionsLocked(ctx, reportID, owner)
	if err != nil {
		return nil, err
	}

	session.UpdatedAt = time.Now().UTC()
	replaced := false
	for i, existing := range sessions {
		if existing.ID == session.ID {
			sessions[i] = session
			replaced = true
			break
		}
	}
	if !replaced {
		sessions = append([]ChatSession{session}, sessions...)
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	if err := s.writeSessionsLocked(ctx, reportID, owner, sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *ChatHistoryStore) readSessionsLocked(ctx context.Context, reportID, owner string) ([]ChatSession, error) {
	object, err := s.client.GetObject(ctx, s.bucket, s.objectKey(reportID, owner), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		if isNotFoundErr(err) {
			return []ChatSession{}, nil
		}
		return nil, err
	}

	var sessions []ChatSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *ChatHistoryStore) writeSessionsLocked(ctx context.Context, reportID, owner string, sessions []ChatSession) error {
	data, err := json.Marshal(sessions)
	if err != nil {
		return err
	}

	_, err = s.client.PutObject(ctx, s.bucket, s.objectKey(reportID, owner), bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	return err
}

func isNotFoundErr(err error) bool {
	response := minio.ToErrorResponse(err)
	return response.Code == "NoSuchKey" || response.Code == "NoSuchBucket"
}
