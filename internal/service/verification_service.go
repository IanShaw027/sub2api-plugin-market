package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/IanShaw027/sub2api-pluginsign"
	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
)

// VerificationService 签名验证服务
type VerificationService struct {
	mu              sync.RWMutex
	trustStore      *pluginsign.TrustStore
	trustKeyRepo    *repository.TrustKeyRepository
	hostRuntime     string
	hostAPIVersion  string
}

// loadTrustKeys 从数据库加载信任密钥并构建 TrustStore
func (s *VerificationService) loadTrustKeys(ctx context.Context) (*pluginsign.TrustStore, error) {
	store := pluginsign.NewTrustStore()
	keys, err := s.trustKeyRepo.ListTrustKeys(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load trust keys: %w", err)
	}
	for _, key := range keys {
		pubKey, err := decodePublicKey(key.PublicKey)
		if err != nil {
			slog.Warn("failed to decode public key", "key_id", key.KeyID, "error", err)
			continue
		}
		if err := store.AddTrustedKey(key.KeyID, pubKey); err != nil {
			slog.Warn("failed to add trusted key", "key_id", key.KeyID, "error", err)
			continue
		}
		if !key.IsActive {
			store.RevokeKey(key.KeyID)
		}
	}
	return store, nil
}

// NewVerificationService 创建签名验证服务
func NewVerificationService(trustKeyRepo *repository.TrustKeyRepository, hostRuntime, hostAPIVersion string) (*VerificationService, error) {
	svc := &VerificationService{
		trustKeyRepo:   trustKeyRepo,
		hostRuntime:    hostRuntime,
		hostAPIVersion: hostAPIVersion,
	}
	store, err := svc.loadTrustKeys(context.Background())
	if err != nil {
		return nil, err
	}
	svc.trustStore = store
	return svc, nil
}

// ReloadTrustKeys 重新从数据库加载信任密钥，支持热更新
func (s *VerificationService) ReloadTrustKeys(ctx context.Context) error {
	newStore, err := s.loadTrustKeys(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.trustStore = newStore
	s.mu.Unlock()
	slog.Info("trust keys reloaded successfully")
	return nil
}

// VerifyPlugin 验证插件签名和哈希
func (s *VerificationService) VerifyPlugin(ctx context.Context, pv *ent.PluginVersion, wasmData io.Reader) error {
	// 读取 WASM 数据
	wasmBytes, err := io.ReadAll(wasmData)
	if err != nil {
		return fmt.Errorf("failed to read wasm data: %w", err)
	}

	signKeyID := strings.TrimSpace(pv.SignKeyID)
	if signKeyID == "" {
		return fmt.Errorf("missing sign key id for plugin version %s", pv.Version)
	}

	signature, err := decodeSignature(pv.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	manifest := pluginsign.Manifest{
		ID:               pv.PluginID.String(),
		Version:          pv.Version,
		Runtime:          s.hostRuntime,
		PluginAPIVersion: pv.PluginAPIVersion,
		SHA256:           pv.WasmHash,
		Compatibility: pluginsign.Compatibility{
			MinPluginAPIVersion: pv.MinAPIVersion,
			MaxPluginAPIVersion: maxPluginAPIVersion(pv),
		},
	}

	checksums := map[string]string{
		"plugin.wasm": pv.WasmHash,
	}

	s.mu.RLock()
	store := s.trustStore
	s.mu.RUnlock()

	if err := pluginsign.VerifyInstall(pluginsign.VerifyInstallRequest{
		Manifest:             manifest,
		Checksums:            checksums,
		ArtifactBytes:        wasmBytes,
		Signature:            signature,
		KeyID:                signKeyID,
		HostRuntime:          s.hostRuntime,
		HostPluginAPIVersion: s.hostAPIVersion,
		TrustStore:           store,
	}); err != nil {
		return fmt.Errorf("verify install failed: %w", err)
	}

	return nil
}

func maxPluginAPIVersion(pv *ent.PluginVersion) string {
	maxVersion := strings.TrimSpace(pv.MaxAPIVersion)
	if maxVersion != "" {
		return maxVersion
	}
	return strings.TrimSpace(pv.PluginAPIVersion)
}

func decodeSignature(raw string) ([]byte, error) {
	signature := strings.TrimSpace(raw)
	signature = strings.TrimPrefix(signature, "ed25519:")
	if signature == "" {
		return nil, fmt.Errorf("signature is empty")
	}

	if decoded, err := hex.DecodeString(signature); err == nil {
		return decoded, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("unsupported signature encoding")
	}
	return decoded, nil
}

func decodePublicKey(raw string) ([]byte, error) {
	publicKey := strings.TrimSpace(raw)
	if publicKey == "" {
		return nil, fmt.Errorf("public key is empty")
	}

	if decoded, err := base64.StdEncoding.DecodeString(publicKey); err == nil {
		return decoded, nil
	}
	decoded, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil, fmt.Errorf("unsupported public key encoding")
	}
	return decoded, nil
}
