package watcher

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestBrowserWorkerConfigDefaultsToHeadlessWithoutDisplay(t *testing.T) {
	t.Setenv("WATCHER_BROWSER_HEADLESS", "")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")

	config := BrowserWorkerConfigFromEnv()

	if !config.Headless {
		t.Fatal("expected browser worker to default to headless when no display is available")
	}
}

func TestBrowserWorkerConfigRespectsExplicitHeadlessFalse(t *testing.T) {
	t.Setenv("WATCHER_BROWSER_HEADLESS", "false")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")

	config := BrowserWorkerConfigFromEnv()

	if config.Headless {
		t.Fatal("expected explicit WATCHER_BROWSER_HEADLESS=false to be respected")
	}
}

func TestBrowserWorkerConfigDefaultsToWindowedWithDisplay(t *testing.T) {
	t.Setenv("WATCHER_BROWSER_HEADLESS", "")
	t.Setenv("DISPLAY", ":0")
	t.Setenv("WAYLAND_DISPLAY", "")

	config := BrowserWorkerConfigFromEnv()

	if config.Headless {
		t.Fatal("expected browser worker to default to windowed when DISPLAY is available")
	}
}

func TestConfiguredBrowserWorkerUsesFirefoxRDPForDebugWebsocket(t *testing.T) {
	controller := NewConfiguredBrowserWorker(uuid.New(), BrowserWorkerConfig{
		DebugURL: "ws://127.0.0.1:6000",
	})

	_, ok := controller.(*FirefoxRDPBrowserWorker)
	require.True(t, ok)
}

func TestConfiguredBrowserWorkerUsesFirefoxRDPForExplicitBackend(t *testing.T) {
	controller := NewConfiguredBrowserWorker(uuid.New(), BrowserWorkerConfig{
		Backend:  "firefox-rdp",
		DebugURL: "ws://127.0.0.1:6000",
	})

	_, ok := controller.(*FirefoxRDPBrowserWorker)
	require.True(t, ok)
}

func TestFirefoxRDPBrowserWorkerOpensJobInProfileJobWindow(t *testing.T) {
	server := newFirefoxRDPTestServer(t, []map[string]interface{}{
		{"from": "root", "applicationType": "browser"},
		{
			"from": "root",
			"tabs": []map[string]interface{}{
				{
					"actor": "tab-descriptor-1",
					"url":   "https://gengo.com/t/jobs/status/available/realtime",
					"title": "Realtime Jobs",
				},
				{
					"actor": "tab-descriptor-login",
					"url":   "https://gengo.com/auth/form/login",
					"title": "Login / Sign up",
				},
			},
		},
		{
			"from": "tab-descriptor-1",
			"frame": map[string]interface{}{
				"actor":         "tab-target-1",
				"consoleActor":  "tab-console-1",
				"innerWindowId": 101,
				"url":           "https://gengo.com/t/jobs/status/available/realtime",
				"title":         "Realtime Jobs",
			},
		},
		{"from": "tab-console-1", "resultID": "nav-1"},
		{
			"from":         "tab-console-1",
			"type":         "evaluationResult",
			"hasException": false,
			"resultID":     "nav-1",
			"result":       `{"queued":true,"url":"https://gengo.com/t/jobs/details/34109850"}`,
		},
		{
			"from": "root",
			"tabs": []map[string]interface{}{
				{
					"actor": "tab-descriptor-2",
					"url":   "https://gengo.com/t/jobs/details/34109850",
					"title": "Job",
				},
			},
		},
		{
			"from": "tab-descriptor-2",
			"frame": map[string]interface{}{
				"actor":         "tab-target-2",
				"consoleActor":  "tab-console-2",
				"innerWindowId": 102,
				"url":           "https://gengo.com/t/jobs/details/34109850",
				"title":         "Job",
			},
		},
		{"from": "tab-console-2", "resultID": "safety-1"},
		{
			"from":         "tab-console-2",
			"type":         "evaluationResult",
			"hasException": false,
			"resultID":     "safety-1",
			"result":       `{"outcome":"opened","url":"https://gengo.com/t/jobs/details/34109850","title":"Job","message":"No hard-stop markers detected"}`,
		},
	})

	worker := NewFirefoxRDPBrowserWorker(uuid.New(), BrowserWorkerConfig{
		DebugURL:      server.URL,
		ActionTimeout: 2 * time.Second,
	})
	result, err := worker.OpenJob(context.Background(), Job{
		ID:  "34109850",
		URL: "https://gengo.com/t/jobs/details/34109850",
	})

	if err != nil {
		t.Logf("packets=%v", server.packets)
	}
	require.NoError(t, err)
	require.Equal(t, BrowserOutcomeOpened, result.Outcome)
	require.Equal(t, "https://gengo.com/t/jobs/details/34109850", result.URL)
	require.Equal(t, []string{"listTabs", "getTarget", "evaluateJSAsync", "listTabs", "getTarget", "evaluateJSAsync"}, server.packetTypes())
	require.Contains(t, server.packets[2]["text"], `window.open(targetUrl, "GengoWatcher Job Window")`)
	require.NotContains(t, server.packets[2]["text"], "location.href = targetUrl")
}

type firefoxRDPTestServer struct {
	URL       string
	responses []map[string]interface{}
	packets   []map[string]interface{}
	done      chan struct{}
}

func newFirefoxRDPTestServer(t *testing.T, responses []map[string]interface{}) *firefoxRDPTestServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &firefoxRDPTestServer{
		URL:       "ws://" + listener.Addr().String(),
		responses: responses,
		done:      make(chan struct{}),
	}
	upgrader := websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()
		defer close(server.done)

		require.NotEmpty(t, server.responses)
		require.NoError(t, conn.WriteJSON(server.responses[0]))
		index := 1
		for index < len(server.responses) {
			var packet map[string]interface{}
			if err := conn.ReadJSON(&packet); err != nil {
				return
			}
			server.packets = append(server.packets, packet)
			require.NoError(t, conn.WriteJSON(server.responses[index]))
			index++
			if index < len(server.responses) {
				raw, _ := json.Marshal(server.responses[index])
				if string(raw) != "{}" && server.responses[index]["type"] == "evaluationResult" {
					require.NoError(t, conn.WriteJSON(server.responses[index]))
					index++
				}
			}
		}
	})
	httpServer := &http.Server{Handler: mux}
	go func() {
		_ = httpServer.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = httpServer.Close()
		select {
		case <-server.done:
		case <-time.After(time.Second):
		}
	})
	return server
}

func (s *firefoxRDPTestServer) packetTypes() []string {
	types := make([]string, 0, len(s.packets))
	for _, packet := range s.packets {
		types = append(types, packet["type"].(string))
	}
	return types
}
