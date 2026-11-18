package httpapi

import (
	"net/http"
	"strings"

	"github.com/CreateAbit-Designs/displaylab-socket/internal/hub"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type NotifyRequest struct {
	Token string                 `json:"token" binding:"required"`
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Data  map[string]interface{} `json:"data"`
}

type Server struct {
	Hub          *hub.Hub
	InternalKey  string
	AllowOrigins string
	upgrader     websocket.Upgrader
}

func NewServer(h *hub.Hub, internalKey, allowOrigins string) *Server {
	s := &Server{
		Hub:          h,
		InternalKey:  internalKey,
		AllowOrigins: allowOrigins,
	}
	s.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     s.checkOrigin,
	}
	return s
}

func (s *Server) checkOrigin(r *http.Request) bool {
	if s.AllowOrigins == "*" || s.AllowOrigins == "" {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, o := range strings.Split(s.AllowOrigins, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}

func (s *Server) requireInternalKey(c *gin.Context) {
	key := c.GetHeader("X-Internal-Key")
	if key == "" {
		key = c.Query("key")
	}
	if s.InternalKey == "" || key != s.InternalKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	c.Next()
}

// Notify is called by DisplayLab's FCM send path with the same payload FCM uses.
// POST /internal/v1/notify
func (s *Server) Notify(c *gin.Context) {
	var req NotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "token required"})
		return
	}
	if req.Data == nil {
		req.Data = map[string]interface{}{}
	}

	ev := hub.Event{
		Event: "screen_update",
		Token: req.Token,
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	}
	delivered := s.Hub.Publish(ev)

	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"event":     ev.Event,
		"delivered": delivered,
	})
}

// WS lets the Flutter/Android player subscribe by device FCM token.
// GET /ws?device_token=<fcm_token>
func (s *Server) WS(c *gin.Context) {
	token := strings.TrimSpace(c.Query("device_token"))
	if token == "" {
		token = strings.TrimSpace(c.Query("token"))
	}
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "device_token required"})
		return
	}

	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := hub.NewClient(s.Hub, conn, token)
	client.Start()
}

func (s *Server) Health(c *gin.Context) {
	devices, connections := s.Hub.Stats()
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"service":     "displaylab-socket",
		"devices":     devices,
		"connections": connections,
	})
}
