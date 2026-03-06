package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
	"github.com/IanShaw027/sub2api-storage"
)

const presignedURLTTL = 5 * time.Minute

var (
	ErrPluginVersionNotFound    = errors.New("plugin version not found")
	ErrPluginVerificationFailed = errors.New("plugin verification failed")
)

// PluginVerifier 定义插件校验能力，便于测试替换实现。
type PluginVerifier interface {
	VerifyPlugin(ctx context.Context, pv *ent.PluginVersion, wasmData io.Reader) error
}

// DownloadService 下载业务逻辑层
type DownloadService struct {
	pluginRepo    *repository.PluginRepository
	storage       storage.Storage
	client        *ent.Client
	verifier      PluginVerifier
	presignCache  *repository.TTLCache
}

// NewDownloadService 创建下载服务
func NewDownloadService(pluginRepo *repository.PluginRepository, storage storage.Storage, client *ent.Client, verifier PluginVerifier) *DownloadService {
	return &DownloadService{
		pluginRepo:   pluginRepo,
		storage:      storage,
		client:       client,
		verifier:     verifier,
		presignCache: repository.NewTTLCache(presignedURLTTL),
	}
}

// DownloadPlugin 下载插件
func (s *DownloadService) DownloadPlugin(ctx context.Context, pluginName, version, clientIP, userAgent string) (*ent.PluginVersion, io.ReadCloser, error) {
	// 获取插件版本信息
	pv, err := s.pluginRepo.GetPluginVersion(ctx, pluginName, version)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrPluginVersionNotFound, err)
	}

	wasmBytes, err := s.loadAndVerifyArtifact(ctx, pv)
	if err != nil {
		if logErr := s.recordDownloadLog(ctx, pv, clientIP, userAgent, false, err.Error()); logErr != nil {
			slog.Error("failed to record download log", "error", logErr)
		}
		return nil, nil, err
	}

	// 记录下载日志
	if err := s.recordDownloadLog(ctx, pv, clientIP, userAgent, true, ""); err != nil {
		// 日志记录失败不影响下载
		slog.Error("failed to record download log", "error", err)
	}

	// 增加下载计数
	if err := s.pluginRepo.IncrementDownloadCount(ctx, pv.PluginID.String()); err != nil {
		// 计数失败不影响下载
		slog.Error("failed to increment download count", "error", err)
	}

	return pv, io.NopCloser(bytes.NewReader(wasmBytes)), nil
}

// recordDownloadLog 记录下载日志
func (s *DownloadService) recordDownloadLog(ctx context.Context, pv *ent.PluginVersion, clientIP, userAgent string, success bool, errorMsg string) error {
	hashedIP := hashClientIP(clientIP)
	errorMsg = strings.TrimSpace(errorMsg)
	userAgent = strings.TrimSpace(userAgent)

	create := s.client.DownloadLog.Create().
		SetPluginID(pv.PluginID).
		SetVersion(pv.Version).
		SetClientIP(hashedIP).
		SetSuccess(success)

	if userAgent != "" {
		create = create.SetNillableUserAgent(&userAgent)
	}
	if errorMsg != "" {
		create = create.SetNillableErrorMessage(&errorMsg)
	}

	_, err := create.Save(ctx)
	return err
}

// GetDownloadURL 获取下载 URL（预签名 URL）
func (s *DownloadService) GetDownloadURL(ctx context.Context, pluginName, version, clientIP, userAgent string) (string, error) {
	// 获取插件版本信息
	pv, err := s.pluginRepo.GetPluginVersion(ctx, pluginName, version)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPluginVersionNotFound, err)
	}

	// 下载前强制校验工件完整性与签名
	if _, err := s.loadAndVerifyArtifact(ctx, pv); err != nil {
		if logErr := s.recordDownloadLog(ctx, pv, clientIP, userAgent, false, err.Error()); logErr != nil {
			slog.Error("failed to record download log", "error", logErr)
		}
		return "", err
	}

	// Reuse pre-signed URL within the same TTL window
	cacheKey := fmt.Sprintf("presign:%s:%d", pv.WasmURL, time.Now().Unix()/int64(presignedURLTTL.Seconds()))
	if cached, ok := s.presignCache.Get(cacheKey); ok {
		return cached.(string), nil
	}

	url, err := s.storage.GetPresignedURL(ctx, pv.WasmURL, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	s.presignCache.Set(cacheKey, url)

	// 记录下载日志
	if err := s.recordDownloadLog(ctx, pv, clientIP, userAgent, true, ""); err != nil {
		slog.Error("failed to record download log", "error", err)
	}

	// 增加下载计数
	if err := s.pluginRepo.IncrementDownloadCount(ctx, pv.PluginID.String()); err != nil {
		slog.Error("failed to increment download count", "error", err)
	}

	return url, nil
}

func (s *DownloadService) loadAndVerifyArtifact(ctx context.Context, pv *ent.PluginVersion) ([]byte, error) {
	if s.verifier == nil {
		return nil, fmt.Errorf("%w: verification service unavailable", ErrPluginVerificationFailed)
	}

	reader, err := s.storage.Download(ctx, pv.WasmURL)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch wasm file: %v", ErrPluginVerificationFailed, err)
	}
	defer reader.Close()

	wasmBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read wasm file: %v", ErrPluginVerificationFailed, err)
	}

	if err := s.verifier.VerifyPlugin(ctx, pv, bytes.NewReader(wasmBytes)); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPluginVerificationFailed, err)
	}

	return wasmBytes, nil
}

func hashClientIP(clientIP string) string {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return "unknown"
	}
	sum := sha256.Sum256([]byte(clientIP))
	return hex.EncodeToString(sum[:])
}
