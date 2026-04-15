package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"rentora/backend/internal/utils"
	"rentora/backend/internal/ws"

	"github.com/gin-gonic/gin"
	wslib "github.com/gorilla/websocket"
)

// Поднимаем ws для GET /ws/chats; JWT берем из ?token= или из Authorization: Bearer.
func ChatWebSocket(hub *ws.Hub, jwtSecret string, allowedOrigins []string) gin.HandlerFunc {
	upgrader := wslib.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allowedOrigins {
				o = strings.TrimSpace(o)
				if o == "" {
					continue
				}
				if o == "*" {
					return true
				}
				if strings.EqualFold(o, origin) {
					return true
				}
			}
			return len(allowedOrigins) == 0
		},
	}

	return func(c *gin.Context) {
		token := strings.TrimSpace(c.Query("token"))
		if token == "" {
			parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token = strings.TrimSpace(parts[1])
			}
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		userID, err := utils.ParseToken(token, jwtSecret)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[ws] upgrade: %v", err)
			return
		}

		hub.Register(userID, conn)
		defer func() {
			hub.Unregister(userID, conn)
			_ = conn.Close()
		}()

		conn.SetReadLimit(512)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if !wslib.IsCloseError(err, wslib.CloseGoingAway, wslib.CloseAbnormalClosure, wslib.CloseNormalClosure) {
					log.Printf("[ws] read user_id=%d: %v", userID, err)
				}
				break
			}
		}
	}
}
