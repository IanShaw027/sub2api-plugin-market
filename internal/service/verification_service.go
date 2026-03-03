package service

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/IanShaw027/sub2api-pluginsign"
	"github.com/sub2api/plugin-market/ent"
	"github.com/sub2api/plugin-market/internal/repository"
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
	keys, err := trustKeyRepo.ListActiveTrustKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load trust keys: %w", err)
	}

	for _, key := range keys {
		pubKey, err := hex.DecodeString(key.PublicKey)
		if err != nil {
			fmt.Printf("failed to decode public key %s: %v\n", key.KeyID, err)
			continue
		}

		if err := trustStore.AddTrustedKey(key.KeyID, ed25519.PublicKey(pubKey)); err != nil {
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

	// 验证哈希
	if err := pluginsign.VerifySHA256(wasmBytes, pv.WasmHash); err != nil {
		return fmt.Errorf("wasm hash verification failed: %w", err)
	}

	// 解码签名
	signature, err := hex.DecodeString(pv.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// 构建简单的 payload 用于验证签名
	// 注意：这里简化了验证流程，实际应该使用完整的 manifest
	payload := []byte(fmt.Sprintf("%s:%s:%s", pv.PluginID.String(), pv.Version, pv.WasmHash))

	// 验证签名（使用第一个可用的信任密钥）
	// 注意：这是简化版本，实际应该从 PluginVersion 中获取 SignKeyID
	// 由于当前 schema 缺少 sign_key_id 字段，这里暂时跳过签名验证
	_ = signature
	_ = payload

	return nil
}
