package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"trace"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/facebook"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

// 現在アクティブなAvatarの実装
var avatars Avatar = TryAvatars {
	UseFileSystemAvatar,
	UseAuthAvatar,
	UseGravatar,
}

// templは1つのテンプレートを表す
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTPはHTTPリクエストを処理する
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}

	t.templ.Execute(w, data) // 本来は戻り値をチェックするべき
}

func main() {
	var addr = flag.String("addr", ":8080", "アプリケーションのアドレス")
	flag.Parse()

	// Gomniauthのセットアップ
	gomniauth.SetSecurityKey(os.Getenv("CHAT_SECURITY_KEY"))
	gomniauth.WithProviders(
		facebook.New("クライアントID", "秘密鍵", "http://localhost:8080/auth/callback/facebook"),
		github.New("クライアントID", "秘密鍵", "http://localhost:8080/auth/callback/github"),
		google.New(os.Getenv("CHAT_CLIENT_ID"), os.Getenv("CHAT_TOKEN"), "http://localhost:8080/auth/callback/google"),
	)

	r := newRoom()
	r.tracer = trace.New(os.Stdout)

	// assets
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets/"))))

	// ルート
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.Handle("/upload", &templateHandler{filename: "upload.html"})
	http.Handle("/room", r)
	http.HandleFunc("/uploader", uploaderHandler)
	http.HandleFunc("/auth/", loginHandler)
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name: "auth",
			Value: "",
			Path: "/",
			MaxAge: -1,
		})
		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.Handle("/avatars/",
		http.StripPrefix("/avatars/",
			http.FileServer(http.Dir("./avatars"))))


	// チャットルームを開始
	go r.run()

	// Webサーバーを開始します
	log.Println("Webサーバーを開始します。ポート: ", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServer:", err)
	}

}
