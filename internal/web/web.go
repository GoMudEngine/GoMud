package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/gorilla/websocket"
)

var (
	httpServer  *http.Server
	httpsServer *http.Server

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	httpRoot = ``

	// Used to interface with plugins and request web stuff
	webPlugins WebPlugin = nil
)

type WebNav struct {
	Name   string
	Target string
}

type WebPlugin interface {
	NavLinks() map[string]string                                                    // Name=>Path pairs
	WebRequest(r *http.Request) (html string, templateData map[string]any, ok bool) // Get the first handler of a given request
}

func SetWebPlugin(wp WebPlugin) {
	webPlugins = wp
}

// serveTemplate searches for the requested file in the HTTP_ROOT,
// parses it as a template, and serves it.
func serveTemplate(w http.ResponseWriter, r *http.Request) {

	if httpRoot == "" {
		httpRoot = filepath.Clean(configs.GetFilePathsConfig().PublicHtml.String())
	}

	// Clean the path to prevent directory traversal.
	// Use path.Clean (not filepath.Clean) so URL paths stay with forward slashes on Windows.
	reqPath := path.Clean(r.URL.Path) // Example: / or /info/faq

	// Build the full file path.
	fullPath := filepath.Join(httpRoot, reqPath)

	// If the path is a directory, look for an index.html.
	info, err := os.Stat(fullPath)
	if err != nil {
		if filepath.Ext(fullPath) != ".html" {
			fullPath += ".html"
		}
	} else if info.IsDir() {
		fullPath = filepath.Join(fullPath, "index.html")
	}

	fileExt := filepath.Ext(fullPath)
	fileBase := filepath.Base(fullPath)

	// All template files to load from the filesystem
	templateFiles := []string{}

	var pageFound bool = true

	var pluginHtml string = ``
	var pluginTplData map[string]any = nil
	var ok bool = false
	var fSize int64 = 0
	var source string = `PublicHtml folder`

	// Check if the file exists, else 404
	fInfo, err := os.Stat(fullPath)
	if err != nil {
		pageFound = false
	}

	// Allow plugin to override request
	if webPlugins != nil {
		pluginHtml, pluginTplData, ok = webPlugins.WebRequest(r)
		fSize = int64(len([]byte(pluginHtml)))
		if ok {
			source = `module`
			pageFound = true
		}
	}

	if !pageFound || len(fileBase) > 0 && fileBase[0] == '_' {
		mudlog.Info("Web", "ip", r.RemoteAddr, "ref", r.Header.Get("Referer"), "file path", fullPath, "file extension", fileExt, "error", "Not found")

		fullPath = filepath.Join(httpRoot, `404.html`)
		fInfo, err = os.Stat(fullPath)

		if err != nil {
			http.NotFound(w, r)
			return
		}

		fSize = fInfo.Size()

		w.WriteHeader(http.StatusNotFound)
	}

	// Log the request
	mudlog.Info("Web", "ip", r.RemoteAddr, "ref", r.Header.Get("Referer"), "file path", fullPath, "file extension", fileExt, "file source", source, "size", fmt.Sprintf(`%.2fk`, float64(fSize)/1024))

	// For non-HTML files, serve them statically.
	if fileExt != ".html" {
		http.ServeFile(w, r, fullPath)
		return
	}

	templateData := map[string]any{
		"REQUEST": r,
		"PATH":    reqPath,
		"CONFIG":  configs.GetConfig(),
		"STATS":   GetStats(),
		"NAV": []WebNav{
			{`Home`, `/`},
			{`Who's Online`, `/online`},
			{`Web Client`, `/webclient`},
			{`See Configuration`, `/viewconfig`},
		},
	}

	// Copy any plugin navigation
	if webPlugins != nil {

		currentNav := templateData[`NAV`].([]WebNav)
		coreCount := len(currentNav)

		for name, path := range webPlugins.NavLinks() {

			found := false
			for i := len(currentNav) - 1; i >= 0; i-- {

				if currentNav[i].Name == name {
					found = true
					if path == `` {
						currentNav = append(currentNav[:i], currentNav[i+1:]...)
						if i < coreCount {
							coreCount--
						}
					} else {
						currentNav[i].Target = path
					}
					break
				}

			}

			if !found {
				currentNav = append(currentNav, WebNav{name, path})
			}
		}

		// Sort plugin-added items alphabetically so tab order is stable
		// (Go map iteration is non-deterministic)
		if len(currentNav) > coreCount {
			pluginNav := currentNav[coreCount:]
			sort.Slice(pluginNav, func(i, j int) bool {
				return pluginNav[i].Name < pluginNav[j].Name
			})
		}

		templateData[`NAV`] = currentNav
	}

	// Copy over any plugin data loaded.
	for name, value := range pluginTplData {
		// Don't allow overwriting defaults
		if _, ok := templateData[name]; !ok {
			templateData[name] = value
		}
	}

	// Parse special files intended to be used as template includes
	globFiles, err := filepath.Glob(filepath.Join(httpRoot, "_*.html"))
	if err == nil {
		templateFiles = append(templateFiles, globFiles...)
	}

	// Parse special files intended to be used as template includes (from the request folder)
	requestDir := filepath.Dir(fullPath)
	if httpRoot != requestDir {
		globFiles, err = filepath.Glob(filepath.Join(requestDir, "_*.html"))
		if err == nil {
			templateFiles = append(templateFiles, globFiles...)
		}
	}

	// Add the final (actual) file

	// Parse
	tmpl := template.New(filepath.Base(fullPath)).Funcs(funcMap)

	if pluginHtml == `` {
		templateFiles = append(templateFiles, fullPath)

	}

	tmpl, err = tmpl.ParseFiles(templateFiles...)
	if err != nil {
		mudlog.Error("HTML ERROR", "action", "ParseFiles", "error", err)
		http.Error(w, "Error parsing template files", http.StatusInternalServerError)
	}

	if pluginHtml != `` {
		tmpl, err = tmpl.Parse(pluginHtml)
		if err != nil {
			mudlog.Error("HTML ERROR", "action", "Parse", "error", err)
			http.Error(w, "Error parsing plugin html", http.StatusInternalServerError)
		}
	}

	// Execute the template and write it to the response.
	if err := tmpl.Execute(w, templateData); err != nil {
		mudlog.Error("HTML ERROR", "action", "Execute", "error", err)
		http.Error(w, "Error executing template", http.StatusInternalServerError)
	}
}

func Listen(wg *sync.WaitGroup, webSocketHandler func(*websocket.Conn)) {

	networkConfig := configs.GetNetworkConfig()

	if networkConfig.HttpPort == 0 && networkConfig.HttpsPort == 0 {
		slog.Error(`Web`, "error", "No ports defined. No web server will be started.")
		return
	}

	// Routing
	// Basic homepage

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = `/static/images/favicon.png`
		serveTemplate(w, r)
	})

	http.HandleFunc("/", serveTemplate)

	// websocket upgrade
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			return
		}
		defer conn.Close()

		webSocketHandler(conn)
	})

	http.Handle("GET /admin/static/", RunWithMUDLocked(
		doBasicAuth(
			handlerToHandlerFunc(
				http.StripPrefix("/admin/static/", http.FileServer(http.Dir(configs.GetFilePathsConfig().AdminHtml.String()+"/static"))),
			),
		),
	))

	// Admin tools
	http.HandleFunc("GET /admin/", RunWithMUDLocked(
		doBasicAuth(adminIndex),
	))

	// Item Admin
	http.HandleFunc("GET /admin/items/", RunWithMUDLocked(
		doBasicAuth(itemsIndex),
	))
	http.HandleFunc("GET /admin/items/itemdata/", RunWithMUDLocked(
		doBasicAuth(itemData),
	))

	// Species Admin
	http.HandleFunc("GET /admin/species/", RunWithMUDLocked(
		doBasicAuth(speciesIndex)),
	)
	http.HandleFunc("GET /admin/species/speciesdata/", RunWithMUDLocked(
		doBasicAuth(speciesData)),
	)

	// Mob Admin
	http.HandleFunc("GET /admin/mobs/", RunWithMUDLocked(
		doBasicAuth(mobsIndex),
	))
	http.HandleFunc("GET /admin/mobs/mobdata/", RunWithMUDLocked(
		doBasicAuth(mobData),
	))

	// Mutator Admin
	http.HandleFunc("GET /admin/mutators/", RunWithMUDLocked(
		doBasicAuth(mutatorsIndex),
	))
	http.HandleFunc("GET /admin/mutators/mutatordata/", RunWithMUDLocked(
		doBasicAuth(mutatorData),
	))

	// Combat Stats Admin
	http.HandleFunc("GET /admin/combat-stats/", RunWithMUDLocked(
		doBasicAuth(combatStatsIndex),
	))
	http.HandleFunc("GET /admin/api/combat-stats/", RunWithMUDLocked(
		doBasicAuth(combatStatsAPI),
	))

	// Room Admin
	http.HandleFunc("GET /admin/rooms/", RunWithMUDLocked(
		doBasicAuth(roomsIndex),
	))
	http.HandleFunc("GET /admin/rooms/roomdata/", RunWithMUDLocked(
		doBasicAuth(roomData),
	))

	//
	// Https server start up
	//

	if networkConfig.HttpsPort > 0 {

		filePaths := configs.GetFilePathsConfig()

		if len(filePaths.HttpsCertFile) == 0 || len(filePaths.HttpsKeyFile) == 0 {

			mudlog.Info("HTTPS", "stage", "skipping", "error", "Undefined public/private key files", "Public Cert", filePaths.HttpsCertFile, "Private Key", filePaths.HttpsKeyFile)

		} else {

			if filePaths.HttpsCertFile != `` && filePaths.HttpsKeyFile != `` {

				mudlog.Info("HTTPS", "stage", "Validating public/private key pair", "Public Cert", filePaths.HttpsCertFile, "Private Key", filePaths.HttpsKeyFile)

				cert, err := tls.LoadX509KeyPair(string(filePaths.HttpsCertFile), string(filePaths.HttpsKeyFile))

				if err != nil {

					mudlog.Error("HTTPS", "error", fmt.Errorf("Error loading certificate and key: %w", err))

				} else {

					tlsConfig := &tls.Config{
						Certificates: []tls.Certificate{cert},
					}

					wg.Add(1)

					httpsServer = &http.Server{
						Addr:      fmt.Sprintf(`:%d`, networkConfig.HttpsPort),
						TLSConfig: tlsConfig,
					}

					mudlog.Info("HTTPS", "stage", "Starting https server", "port", networkConfig.HttpsPort)
					go func() {
						defer wg.Done()
						if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
							mudlog.Error("HTTPS", "error", fmt.Errorf("Error starting HTTPS web server: %w", err))
						}
					}()
				}
			}
		}
	}

	//
	// Http server start up
	//

	if networkConfig.HttpPort > 0 {

		httpServer = &http.Server{
			Addr: fmt.Sprintf(`:%d`, networkConfig.HttpPort),
		}

		if networkConfig.HttpsRedirect {

			if httpsServer == nil {

				mudlog.Error("HTTP", "error", "Cannot enable https redirect. There is no https server configured/running.")

			} else {

				var redirectHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {

					host := r.Host
					// If the host header includes a port (e.g. "example.com:80"), strip it out.
					if strings.Contains(host, ":") {
						host, _, _ = net.SplitHostPort(host)
					}

					// Build the target URL with your known HTTPS port (443 in this case).
					target := fmt.Sprintf("https://%s:%d%s", host, networkConfig.HttpsPort, r.RequestURI)

					http.Redirect(w, r, target, http.StatusMovedPermanently)
				}

				httpServer.Handler = redirectHandler

			}

		}

		// HTTP Server
		wg.Add(1)

		mudlog.Info("HTTP", "stage", "Starting http server", "port", networkConfig.HttpPort)
		go func() {
			defer wg.Done()

			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				mudlog.Error("HTTP", "error", fmt.Errorf("Error starting web server: %w", err))
			}
		}()
	}

}

// This wraps the handler functiojn with a game lock (mutex) to keep the mud from
// Concurrently accessing the same memory
func RunWithMUDLocked(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		util.LockMud()
		defer util.UnlockMud()

		next.ServeHTTP(w, r)
	})
}

func Shutdown() {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			mudlog.Error("HTTP", "error", fmt.Errorf("HTTP server shutdown failed: %w", err))
		} else {
			mudlog.Info("HTTPS", "stage", "stopped")
		}
	}

	if httpsServer != nil {
		if err := httpsServer.Shutdown(ctx); err != nil {
			mudlog.Error("HTTPS", "error", fmt.Errorf("HTTP server shutdown failed: %w", err))
		} else {
			mudlog.Info("HTTPS", "stage", "stopped")
		}
	}
}

func sendError(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	if status == http.StatusNotFound {
		fmt.Fprint(w, "custom 404")
	}
}
