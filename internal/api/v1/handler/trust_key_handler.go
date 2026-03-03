package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
)

// TrustKeyHandler 信任密钥处理器
type TrustKeyHandler struct {
	trustKeyService *service.TrustKeyService
}

// NewTrustKeyHandler 创建信任密钥处理器
func NewTrustKeyHandler(trustKeyService *service.TrustKeyService) *TrustKeyHandler {
	return &TrustKeyHandler{
		trustKeyService: trustKeyService,
	}
}

// ListTrustKeys 信任密钥列表接口
// GET /api/v1/trust-keys
func (h *TrustKeyHandler) ListTrustKeys(c *gin.Context) {
	keyType := c.Query("key_type")

	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		val := isActiveStr == "true"
		isActive = &val
	}

	req := &service.ListTrustKeysRequest{
		KeyType:  keyType,
		IsActive: isActive,
	}

	keys, err := h.trustKeyService.ListTrustKeys(c.Request.Context(), req)
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询信任密钥列表失败")
		return
	}

	Success(c, gin.H{
		"trust_keys": keys,
		"total":      len(keys),
	})
}

// GetTrustKeyDetail 信任密钥详情接口
// GET /api/v1/trust-keys/:key_id
func (h *TrustKeyHandler) GetTrustKeyDetail(c *gin.Context) {
	keyID := c.Param("key_id")
	if keyID == "" {
		Error(c, ErrCodeInvalidParam, "密钥 ID 不能为空")
		return
	}

	key, err := h.trustKeyService.GetTrustKeyDetail(c.Request.Context(), keyID)
	if err != nil {
		Error(c, ErrCodeNotFound, "信任密钥不存在")
		return
	}

	Success(c, key)
}
