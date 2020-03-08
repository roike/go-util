package router

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAppHandle(t *testing.T) {
	routed := false
	var writerHandle AppHandle = func(w io.Writer, r *http.Request, ps Param) error {
		routed = true
		want := Param{"entry": "thirdpen", "tag": "euler", "offset": "0"}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
		io.WriteString(w, "<html><body>Hello World!</body></html>")
		return nil
	}
	router := New("/")
	router.Handle("GET", "/latest/:entry/:tag/:offset", writerHandle)
	router.Handle("GET", "/latest/:entry/:offset", writerHandle)

	req := httptest.NewRequest("GET", "/latest/thirdpen/euler/0", nil)

	h, _, err := router.getHandle(req)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	if _, ok := h.(AppHandle); !ok {
		t.Fatal("Handle type is invalid.")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	t.Logf("StatusCode is %v, body is %v", resp.StatusCode, string(body))
	if !routed {
		t.Fatal("routing failed")
	}
}

func TestFileHandle(t *testing.T) {

	router := New("/")
	router.FileServe("/:filepath", http.Dir("./static"))

	req := httptest.NewRequest("GET", "/test.html", nil)

	h, _, err := router.getHandle(req)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	if _, ok := h.(FileHandle); !ok {
		t.Fatal("Handle type is invalid.")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		t.Fatalf("StatusCode is %v.", resp.StatusCode)
	}
	t.Logf("StatusCode is %v, body is %v", resp.StatusCode, string(body))
}
