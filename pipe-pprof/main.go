package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

const filePath = "./big_buck_bunny_1080p_h264.mov"

func underTest() {
	r := mux.NewRouter()

	r.HandleFunc("/pipe", pipeTestFile).Methods(http.MethodGet)
	r.HandleFunc("/load", loadTestFile).Methods(http.MethodGet)

	log.Println(http.ListenAndServe(":8000", r))

}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	underTest()

}

func pipeTestFile(w http.ResponseWriter, r *http.Request) {
	var status = http.StatusOK

	pr, pw := io.Pipe()

	// we need to wait for everything to be done
	wg := &sync.WaitGroup{}
	wg.Add(2)

	// we get some file as input
	f, err := os.Open(filePath)
	defer f.Close()

	if err != nil {
		status = http.StatusInternalServerError
		w.WriteHeader(status)
		fmt.Fprintf(w, "error reading file: %s", err.Error())
		return
	}

	// feed the pipe
	go func(f *os.File, pw *io.PipeWriter, wg *sync.WaitGroup) {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := f.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				pw.CloseWithError(errors.New(fmt.Sprintf("error reading file, err: %s", err.Error())))
				continue
			}
			if n > 0 {
				pw.Write(buf[:n])
			}
		}
	}(f, pw, wg)

	go func(w http.ResponseWriter, pr *io.PipeReader, wg *sync.WaitGroup) {
		defer wg.Done()
		if _, err := io.Copy(w, pr); err != nil {
			fmt.Fprintf(w, "\nerror reading file: %s", err.Error())
		}
	}(w, pr, wg)

	wg.Wait()

}

func loadTestFile(w http.ResponseWriter, r *http.Request) {
	var (
		out    []byte
		status int
	)

	f, err := os.ReadFile(filePath)
	if err == nil {
		status = http.StatusOK
		out = f
	} else {
		out = []byte(fmt.Sprintf("error loading file, err: %s", err.Error()))
		status = http.StatusInternalServerError
	}

	_, _ = io.Copy(w, bytes.NewReader(out))
	w.WriteHeader(status)
}
