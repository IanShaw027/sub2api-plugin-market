package service

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/IanShaw027/sub2api-pluginsign"
	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
)

// VerificationService 签名验证服务
type VerificationService struct {
	trustStore      *pluginsign.TrustStore
	trustKeyRepo    *repository.TrustKeyRepository
	hostRuntime     string
	hostAPIVersion  string
}

// NewVerificationService 创建签名验证服务
func NewVerificationService(trustKeyRepo *repository.TrustKeyRepository, hostRuntime, hostAPIVersion string) (*VerificationService, error) {
	trustStore := pluginsign.NewTrustStore()

	// 从数据库加载信任密钥
	ctx := context.Background()
	keys, err := trustKeyRepo.ListTrustKeys(ctx, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load trust keys: %w", err)
	}

	for _, key := range keys {
		pubKey, err := decodePublicKey(key.PublicKey)
		if err != nil {
			fmt.Printf("failed to decode public key %s: %v\n", key.KeyID, err)
			continue
		}

		if err := trustStore.AddTrustedKey(key.KeyID, pubKey); err != nil {
			fmt.Printf("failed to add trusted key %s: %v\n", key.KeyID, err)
			continue
		}

		if !key.IsActive {
			trustStore.RevokeKey(key.KeyID)
		}
	}

	return &VerificationService{
		trustStore:     trustStore,
		trustKeyRepo:   trustKeyRepo,
		hostRuntime:    hostRuntime,
		hostAPIVersion: hostAPIVersion,
	}, nil
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

	if err := pluginsign.VerifyInstall(pluginsign.VerifyInstallRequest{
		Manifest:             manifest,
		Checksums:            checksums,
		ArtifactBytes:        wasmBytes,
		Signature:            signature,
		KeyID:                signKeyID,
		HostRuntime:          s.hostRuntime,
		HostPluginAPIVersion: s.hostAPIVersion,
		TrustStore:           s.trustStore,
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
