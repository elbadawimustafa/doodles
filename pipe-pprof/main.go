package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

const filePath = "./oceans_hevc.m4f"

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
	defer pr.Close()
	defer pw.Close()

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
		defer pw.Close()
		if _, err := io.Copy(pw, f); err != nil {
			pw.CloseWithError(fmt.Errorf("error reading file: %s", err.Error()))
		}
	}(f, pw, wg)

	go func(w http.ResponseWriter, pr *io.PipeReader, wg *sync.WaitGroup, status int) {
		defer wg.Done()
		if _, err := io.Copy(w, pr); err != nil {
			fmt.Fprintf(w, "\nerror reading file: %s", err.Error())
			status = http.StatusInternalServerError
		}
		w.WriteHeader(status)
	}(w, pr, wg, status)

	wg.Wait()

}

func loadTestFile(w http.ResponseWriter, r *http.Request) {
	var (
		status int
	)

	f, err := os.Open(filePath)
	defer f.Close()
	if err == nil {
		status = http.StatusOK
	} else {
		fmt.Fprintf(w, "error reading file: %s", err.Error())
		status = http.StatusInternalServerError
		w.WriteHeader(status)
		return
	}

	_, _ = io.Copy(w, f)
	w.WriteHeader(status)
}
