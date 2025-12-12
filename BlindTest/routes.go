package blindtest

import "net/http"

func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func RegisterRoutes(authMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	http.HandleFunc("/BlindTest", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "BlindTest/static/index.html")
	}))

	http.HandleFunc("/blindtest/ws", handleWebSocket)

	fs := http.FileServer(http.Dir("BlindTest/static"))
	http.Handle("/blindtest/static/", http.StripPrefix("/blindtest/static/", fs))
}
