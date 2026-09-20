package middleware

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"cups-web/internal/auth"
	"cups-web/internal/store"
)

// APIKeyPrefix 是本项目签发密钥的固定前缀，便于用户/日志识别（Issue #113）。
const APIKeyPrefix = "cw_"

// guestUsernameReserved 与 auth_handlers.go 里的 guestUsername 保持一致。
// 复制一份常量是为了避免 middleware 反向依赖 cmd/server；一旦二者出现漂移，
// APIKeyAuth 的单元测试会直接暴露。
const guestUsernameReserved = "guest"

// APIKeyAuth 识别请求头里的 API Key 并把归属 Session 挂到 context 上。
//
// 行为契约：
//   - 未携带 key 时静默放行，交由下游 RequireSession 用 cookie 判定。
//   - 携带了 key 但 key 无效 / 已过期 / guest 时立即 401，不再尝试 cookie——
//     携带者显然在走 API 通道，静默降级只会掩盖问题。
//   - 鉴权成功时用 auth.WithSession + auth.WithAPIKeyAuth 注入两个标记，
//     ValidateCSRF 据此豁免 double-submit 校验。
//
// 支持两种承载方式，任一匹配即视为携带：
//  1. Authorization: Bearer <key>
//  2. X-API-Key: <key>
func APIKeyAuth(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractAPIKey(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(token, APIKeyPrefix) {
				denyAPIKey(w, r, "invalid api key")
				return
			}
			sum := sha256.Sum256([]byte(token))
			hash := hex.EncodeToString(sum[:])

			var sess auth.Session
			var keyID int64
			err := st.WithTx(r.Context(), false, func(tx *sql.Tx) error {
				key, err := store.GetAPIKeyByHash(r.Context(), tx, hash)
				if err != nil {
					return err
				}
				if key.ExpiresAt != "" {
					t, perr := time.Parse(time.RFC3339, key.ExpiresAt)
					if perr == nil && time.Now().UTC().After(t) {
						return errAPIKeyExpired
					}
				}
				user, err := store.GetUserByID(r.Context(), tx, key.UserID)
				if err != nil {
					return err
				}
				// guest 是访客模式保留账号：无口令、被硬编码 protected，禁止
				// 通过它签发的 key 走 API 通道，以免公网访客模式部署下 key
				// 泄露就等于把接口任意暴露。
				if user.Username == guestUsernameReserved {
					return errAPIKeyGuest
				}
				sess = auth.Session{UserID: user.ID, Username: user.Username, Role: user.Role}
				keyID = key.ID
				// last_used_* 是审计信息，跟着鉴权事务一起更新；失败不阻塞放行——
				// 上层 return 非 nil 会 rollback，反倒把鉴权本身弄丢，所以这里
				// 单独判空并只记 warn。
				if terr := store.TouchAPIKey(r.Context(), tx, key.ID, clientIP(r)); terr != nil {
					log.Printf("[api-key] touch last_used failed: id=%d err=%v", key.ID, terr)
				}
				return nil
			})
			if err != nil {
				switch {
				case errors.Is(err, store.ErrAPIKeyNotFound):
					denyAPIKey(w, r, "invalid api key")
				case errors.Is(err, errAPIKeyExpired):
					denyAPIKey(w, r, "api key expired")
				case errors.Is(err, errAPIKeyGuest):
					denyAPIKey(w, r, "guest api key not allowed")
				default:
					log.Printf("[api-key] lookup failed: err=%v", err)
					http.Error(w, "api key lookup failed", http.StatusInternalServerError)
				}
				return
			}
			log.Printf("[api-key] ok: key_id=%d user=%s path=%s", keyID, sess.Username, r.URL.Path)
			ctx := auth.WithSession(r.Context(), sess)
			ctx = auth.WithAPIKeyAuth(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var (
	errAPIKeyExpired = errors.New("api key expired")
	errAPIKeyGuest   = errors.New("guest api key")
)

// extractAPIKey 优先取 Authorization: Bearer <key>，其次 X-API-Key。两者都空
// 时返回空串，让中间件走 cookie 路径。
func extractAPIKey(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("Authorization")); v != "" {
		const prefix = "Bearer "
		if len(v) > len(prefix) && strings.EqualFold(v[:len(prefix)], prefix) {
			return strings.TrimSpace(v[len(prefix):])
		}
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

// clientIP 抽取最接近真实客户端的 IP。反代下 X-Forwarded-For 的第一跳最贴近，
// 直连时回落到 RemoteAddr。这个字段仅用于审计展示，不做鉴权决策，故不追求
// 严格防伪。
func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		first, _, _ := strings.Cut(v, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func denyAPIKey(w http.ResponseWriter, r *http.Request, reason string) {
	log.Printf("[api-key] deny: reason=%q path=%s", reason, r.URL.Path)
	http.Error(w, reason, http.StatusUnauthorized)
}
