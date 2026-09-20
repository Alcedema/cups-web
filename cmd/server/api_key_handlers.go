package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"cups-web/internal/auth"
	"cups-web/internal/middleware"
	"cups-web/internal/store"
)

type apiKeyResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt"`
	LastUsedIP string `json:"lastUsedIp"`
	ExpiresAt  string `json:"expiresAt"`
}

type createAPIKeyPayload struct {
	Name      string `json:"name"`
	ExpiresIn *int   `json:"expiresInDays"`
}

type createAPIKeyResponse struct {
	Key   string         `json:"key"`
	Entry apiKeyResponse `json:"entry"`
}

// apiKeyMaxPerUser 每个用户最多允许的密钥数量。防止误操作或凭据泄露后攻击者
// 无限刷条目撑爆表；数值上足以覆盖多渠道对接（微信 / 飞书 / 钉钉 / 自建脚本）。
const apiKeyMaxPerUser = 20

// apiKeyMaxExpireDays 最长过期时间上限（约 5 年）。留一个明确上限，防止有人
// 传 int 溢出或者写成秒把过期时间推到远古之后。
const apiKeyMaxExpireDays = 365 * 5

// apiKeyListHandler 列出当前用户持有的密钥。仅返回展示前缀，不返回明文与哈希。
func apiKeyListHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// API Key 通过 API Key 通道调用本接口是允许的（就是"我这条 key 属于哪个用户"
	// 的自查），但为了避免密钥可以自繁殖，创建/删除接口下面会显式禁止。
	if strings.EqualFold(sess.Username, guestUsername) {
		writeJSONError(w, http.StatusForbidden, "guest 不支持 API 密钥")
		return
	}
	var resp []apiKeyResponse
	err = appStore.WithTx(r.Context(), true, func(tx *sql.Tx) error {
		keys, err := store.ListAPIKeysByUser(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}
		resp = mapAPIKeys(keys)
		return nil
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}
	writeJSON(w, resp)
}

// apiKeyCreateHandler 生成一枚新密钥并返回明文（一次性）。禁止用 API Key 通道
// 调用，避免密钥自繁殖：只有真人登录 session 才能签发新 key。
func apiKeyCreateHandler(w http.ResponseWriter, r *http.Request) {
	if auth.IsAPIKeyAuth(r.Context()) {
		writeJSONError(w, http.StatusForbidden, "API 密钥不能签发新密钥，请使用浏览器登录后再创建")
		return
	}
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if strings.EqualFold(sess.Username, guestUsername) {
		writeJSONError(w, http.StatusForbidden, "guest 不支持 API 密钥")
		return
	}
	var payload createAPIKeyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "name required")
		return
	}
	if len(name) > 64 {
		writeJSONError(w, http.StatusBadRequest, "name too long")
		return
	}
	var expiresAt string
	if payload.ExpiresIn != nil {
		days := *payload.ExpiresIn
		if days < 0 || days > apiKeyMaxExpireDays {
			writeJSONError(w, http.StatusBadRequest, "invalid expiresInDays")
			return
		}
		if days > 0 {
			expiresAt = time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
		}
	}

	// 明文格式：cw_<32B 随机 base32>；randomToken() 用 crypto/rand.Text，
	// 全大写字母数字，与项目现有随机 id 风格一致。
	token := middleware.APIKeyPrefix + randomToken()
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	prefix := apiKeyDisplayPrefix(token)

	var created store.APIKey
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		existing, err := store.ListAPIKeysByUser(r.Context(), tx, sess.UserID)
		if err != nil {
			return err
		}
		if len(existing) >= apiKeyMaxPerUser {
			return errAPIKeyLimit
		}
		key, err := store.CreateAPIKey(r.Context(), tx, store.CreateAPIKeyInput{
			UserID:    sess.UserID,
			Name:      name,
			Prefix:    prefix,
			TokenHash: hash,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}
		created = key
		return nil
	})
	if err != nil {
		if errors.Is(err, errAPIKeyLimit) {
			writeJSONError(w, http.StatusBadRequest, "已达到密钥数量上限")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}
	writeJSON(w, createAPIKeyResponse{
		Key:   token,
		Entry: mapAPIKey(created),
	})
}

// apiKeyDeleteHandler 删除自己名下的一枚密钥。API Key 通道禁用，理由同 create——
// 一枚被泄露的 key 不应该能反过来清除其他 key 干扰运维追溯。
func apiKeyDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if auth.IsAPIKeyAuth(r.Context()) {
		writeJSONError(w, http.StatusForbidden, "API 密钥不能删除密钥，请使用浏览器登录后再操作")
		return
	}
	sess, err := auth.GetSession(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = appStore.WithTx(r.Context(), false, func(tx *sql.Tx) error {
		return store.DeleteAPIKey(r.Context(), tx, id, sess.UserID)
	})
	if err != nil {
		if errors.Is(err, store.ErrAPIKeyNotFound) {
			writeJSONError(w, http.StatusNotFound, "api key not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to delete api key")
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

var errAPIKeyLimit = errors.New("api key limit reached")

// apiKeyDisplayPrefix 生成前端展示用的打码串："<前 12 位>…<后 4 位>"。
// 前 12 位包含 "cw_" 前缀与 9 位随机，尾 4 位方便用户对照记忆；中间的原始
// token 被彻底隐去，即使拿到 prefix 也无法从中反推。
func apiKeyDisplayPrefix(token string) string {
	if len(token) <= 16 {
		return token
	}
	return token[:12] + "…" + token[len(token)-4:]
}

func mapAPIKey(k store.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		Prefix:     k.Prefix,
		CreatedAt:  k.CreatedAt,
		LastUsedAt: k.LastUsedAt,
		LastUsedIP: k.LastUsedIP,
		ExpiresAt:  k.ExpiresAt,
	}
}

func mapAPIKeys(keys []store.APIKey) []apiKeyResponse {
	resp := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp = append(resp, mapAPIKey(k))
	}
	return resp
}
