package router
/*-- description --
 * update: 2020/0205
 */

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// AppError makes an object it has an error and a HTTP response code.
// error == interface
// usage:AppError(http.StatusBadRequest, "BadRequest. Token is empty.")
type AppError struct {
	error
	Code int
}

/* Param:
   AppHandleの引数でurlがパラメータを含む場合に格納
*/
type Param map[string]string

// 処理を委譲する関数雛形
// 処理結果はstream(w io.Writer)に出力するので戻値はerrorのみ
type AppHandle func(io.Writer, *http.Request, Param) error

/* 処理を委譲する関数雛形が複数ある場合
例:type PushHandle func(http.ResponseWriter, *http.Request, *Streamer)
varで関数雛形を宣言する必要がある
var fooHandle rt.AppHandle = func(w io.Writer, r *http.Request, _ rt.Param) error {}
var hooHandle rt.PushHandle = func(http.ResponseWriter, *http.Request, *Streamer)
一種類しか関数雛形がない場合は
func handleWrapper(r *http.Request) error {}
と宣言できる
*/

type FileHandle func(http.ResponseWriter, *http.Request, http.FileSystem)

// loginチェックなど前処理
type Wrapper func(*http.Request, string) (string, error)

/* AppRouter
   rootはモジュールのurl起点
   例えばモジュールchatのurl起点は/chat/になる
   [/:list]は変数
   treesのKeyはmethedのGET POST PATCH DELET等
   trees[GET][path]=AppHandle
*/
type AppRouter struct {
	root         string
	fileRoot	 http.FileSystem
	trees        map[string]map[string]interface{}
	Wrapper      Wrapper
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}

func New(root string) *AppRouter {
	return &AppRouter{
		root:  root,
		trees: map[string]map[string]interface{}{
			"GET": map[string]interface{}{},
			"POST": map[string]interface{}{},
		},
	}
}

func (rt *AppRouter) Handle(method, path string, h AppHandle) {
	rt.trees[method][path] = h
}

// Usage: rt.FileServe("/js/lib/:name", http.Dir("static"))
func (rt *AppRouter) FileServe(path string, fs http.FileSystem) {
	rt.fileRoot = fs
	rt.trees["GET"][path] = serveFile
}

func (r *AppRouter) recv(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(w, req, rcv)
	}
}

// appErrorf creates a new appError given a reponse code and a message.
func AppErrorf(code int, format string, args ...interface{}) *AppError {
	return &AppError{fmt.Errorf(format, args...), code}
}

// AppRouter implements http.Handler interface.
// 各モジュールのroot pathでdispatchするので
// 各モジュールはrt.root == rt.URL.Path[len(rt.root):] になる
// しかしdefaultモジュールの"/"以下は一種ワイルドカードになる
// エラー処理はフロントエンドにリレーする
// ハンドリングする関数はAppHandleとPushHandleの2つ
func (rt *AppRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rt.PanicHandler != nil {
		defer rt.recv(w, r)
	}
	h, ps, err := rt.getHandle(r)

	// AppHandleの処理でエラーが出た場合はエラーオブジェクトを返し
	// フロント側のjavascriptの処理にまかせる
	code := http.StatusInternalServerError
	// 正しいurlがrequestされなかった
	if err != nil {
		if err, ok := err.(*AppError); ok {
			code = err.Code
		}
		w.WriteHeader(code)
		fmt.Fprint(w, err)
		return
	}

	// --- FileHandle ---
	if h, ok := h.(FileHandle); ok {
		FileHandle(h)(w, r, rt.fileRoot)
		return
	}

	// --- AppHandle ---
	buf := &bytes.Buffer{}
	if h, ok := h.(AppHandle); ok {
		err = AppHandle(h)(buf, r, ps)
		if err == nil {
			io.Copy(w, buf)
			return
		}
	}
	if err, ok := err.(*AppError); ok {
		code = err.Code
	}

	w.WriteHeader(code)
	fmt.Fprint(w, err)

	return

}

// requestパスに対応する関数の仕分け、requestパスのワイルドカードに対応
func (rt *AppRouter) getHandle(r *http.Request) (handler interface{}, p Param, err error) {
	url := r.URL.Path
	path := url
	matched, err := regexp.MatchString(`/api/.*`, url)
	if matched == true {
		path = url[len(rt.root):]
	}

	if rt.Wrapper != nil {
		path, err = rt.Wrapper(r, path)
		if err != nil {
			return nil, nil, err
		}
	}

	trees := rt.trees[r.Method]
	if trees == nil {
		return nil, nil, AppErrorf(http.StatusBadRequest, "Use invalid Method")
	}

	// request Pathがroot Pathに完全一致<-パラメータが無い
	handler, ok := trees[path]
	if ok {
		return handler, nil, nil
	}

	//url パラメータの抽出
	params := strings.Split(path, "/")[1:]
	for template := range trees {
		//template: /list/name/:name -->:nameはparamsに格納
		idx := strings.IndexAny(template, ":")
		if idx == -1 {
			continue
		}
		if strings.HasPrefix(path, template[:idx]) == false {
			continue
		}
		parts := strings.Split(template, "/")[1:]
		if len(parts) != len(params) {
			continue
		}
		p = make(Param)
		for i, part := range parts {
			if strings.HasPrefix(part, ":") {
				p[part[1:]] = params[i]
			}
		}
		handler, ok = trees[template]
		if ok {
			break
		}
	}
	return handler, p, nil
}

// ---- FileHandler function ----
var serveFile FileHandle = func(w http.ResponseWriter, r *http.Request, fs http.FileSystem) {
	fileServer := http.FileServer(fs)
	fileServer.ServeHTTP(w, r)
}
