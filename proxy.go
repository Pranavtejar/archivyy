import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	ui, err := url.Parse("http://localhost:8081")
	if err != nil {
		log.Fatal(err)
	}
	
	uiProxy := httputil.NewSingleHostReverseProxy(ui)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		uiProxy.ServeHTTP(w, r)
	})

	api, err := url.Parse("http://localhost:8082")
	if err != nil {
		log.Fatal(err)
	}
	apiProxy := httputil.NewSingleHostReverseProxy(api)
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		apiProxy.ServeHTTP(w, r)
	})
	log.Println("starting proxy server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
	
}
