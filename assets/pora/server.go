package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const backgroundLoadDirName = "pora-background-load"

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/env", env)
	http.HandleFunc("/write", write)
	http.HandleFunc("/create", createFile)
	http.HandleFunc("/loadtest", dataLoad)
	http.HandleFunc("/loadtestcleanup", dataLoadCleanup)
	http.HandleFunc("/read/", readFile)
	http.HandleFunc("/chmod/", chmodFile)
	http.HandleFunc("/delete/", deleteFile)
	http.HandleFunc("/mkdir-for-background-load", mkdirForBackgroundLoad)
	fmt.Println("listening...")

	ports := os.Getenv("PORT")
	portArray := strings.Split(ports, " ")

	errCh := make(chan error)

	for _, port := range portArray {
		println(port)
		go func(port string) {
			errCh <- http.ListenAndServe(":"+port, nil)
		}(port)
	}

	if runBackgroundLoad := os.Getenv("RUN_BACKGROUND_LOAD_THREAD"); runBackgroundLoad != "" {
		fmt.Println("starting background load thread...")
		go backgroundLoad()
	}

	err := <-errCh
	if err != nil {
		panic(err)
	}
}

type VCAPApplication struct {
	InstanceIndex int `json:"instance_index"`
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "instance index: %s", os.Getenv("INSTANCE_INDEX"))
}

func getPath() string {
	r, err := regexp.Compile(`"container_dir":\s*"([^"]+)"`)
	if err != nil {
		panic(err)
	}

	vcapEnv := os.Getenv("VCAP_SERVICES")
	match := r.FindStringSubmatch(vcapEnv)
	if len(match) < 2 {
		fmt.Fprintf(os.Stderr, "VCAP_SERVICES is %s", vcapEnv)
		panic("failed to find container_dir in environment json")
	}

	return match[1]
}

func write(res http.ResponseWriter, req *http.Request) {
	mountPointPath := getPath() + "/poratest-" + randomString(10)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		writeError(res, "Writing \n", err)
		return
	}

	body, err := ioutil.ReadFile(mountPointPath)
	if err != nil {
		writeError(res, "Reading \n", err)
		return
	}

	err = os.Remove(mountPointPath)
	if err != nil {
		writeError(res, "Deleting \n", err)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(body)
	return
}

func dataLoad(res http.ResponseWriter, req *http.Request) {
	// this method will read and write data to a single file for 4 seconds, then clean up.
	mountPointPath := getPath() + "/poraload-" + randomString(10)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		writeError(res, "Writing \n", err)
		return
	}

	var totalIO int
	for startTime := time.Now(); time.Since(startTime) < 4*time.Second; {
		d2 := []byte(randomString(1048576))
		err := ioutil.WriteFile(mountPointPath, d2, 0644)
		if err != nil {
			writeError(res, "Writing Load\n", err)
			return
		}
		body, err := ioutil.ReadFile(mountPointPath)
		if err != nil {
			writeError(res, "Reading Load\n", err)
			return
		}
		if string(body) != string(d2) {
			writeError(res, "Data Mismatch\n", err)
			return
		}
		totalIO = totalIO + 1
	}

	err = os.Remove(mountPointPath)
	if err != nil {
		writeError(res, "Deleting\n", err)
		return
	}

	res.WriteHeader(http.StatusOK)
	body := fmt.Sprintf("%d MiB written\n", totalIO)
	res.Write([]byte(body))
	return
}

func backgroundLoad() {
	// this method will run forever reading and writing and cleaning up data files
	dirPath := filepath.Join(getPath(), backgroundLoadDirName)

	for true {
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			fmt.Println("background load directory doesn't exist... waiting")
			time.Sleep(10 * time.Second)
			continue
		}

		filePath := filepath.Join(dirPath, "poraload-"+os.Getenv("INSTANCE_INDEX"))

		d2 := []byte(randomString(1048576))
		err := ioutil.WriteFile(filePath, d2, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}

		body, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Println(err)
			return
		}

		if string(body) != string(d2) {
			fmt.Println("Data Mismatch!")
			return
		}

		os.Remove(filePath)
	}
}

func dataLoadCleanup(res http.ResponseWriter, req *http.Request) {
	// this method will clean up any files that couldn't be deleted during load testing due to interruptions.
	mountPointPath := getPath() + "/poraload-*"

	files, err := filepath.Glob(mountPointPath)
	if err != nil {
		writeError(res, "Unable to find files \n", err)
		return
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			writeError(res, "Unable to remove "+f+" \n", err)
			return
		}
	}

	res.WriteHeader(http.StatusOK)
	body := fmt.Sprintf("%d Files Removed\n", len(files))
	res.Write([]byte(body))
	return
}

func createFile(res http.ResponseWriter, _ *http.Request) {
	fileName := "pora" + randomString(10)
	mountPointPath := filepath.Join(getPath(), fileName)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		writeError(res, "Writing \n", err)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fileName))
	return
}

func mkdirForBackgroundLoad(res http.ResponseWriter, _ *http.Request) {
	dirPath := filepath.Join(getPath(), backgroundLoadDirName)

	err := os.MkdirAll(dirPath, 0777)
	if err != nil {
		writeError(res, "Error creating directory", err)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(dirPath))
}

func readFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts)-1]
	mountPointPath := filepath.Join(getPath(), fileName)

	body, err := ioutil.ReadFile(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(body)
	res.Write([]byte("instance index: " + os.Getenv("INSTANCE_INDEX")))
	return
}

func chmodFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts)-2]
	mountPointPath := filepath.Join(getPath(), fileName)
	mode := parts[len(parts)-1]
	parsedMode, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		res.Write([]byte(err.Error()))
	}
	err = os.Chmod(mountPointPath, os.FileMode(uint(parsedMode)))
	if err != nil {
		res.WriteHeader(http.StatusForbidden)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fileName + "->" + mode))
	res.Write([]byte("instance index: " + os.Getenv("INSTANCE_INDEX")))
	return
}

func deleteFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts)-1]
	mountPointPath := filepath.Join(getPath(), fileName)

	err := os.Remove(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusForbidden)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte("deleted " + fileName))
	return
}

func env(res http.ResponseWriter, req *http.Request) {
	for _, e := range os.Environ() {
		fmt.Fprintf(res, "%s\n", e)
	}
}

var isSeeded = false

func randomString(n int) string {
	if !isSeeded {
		rand.Seed(time.Now().UnixNano())
		isSeeded = true
	}
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func writeError(res http.ResponseWriter, msg string, err error) {
	res.WriteHeader(http.StatusInternalServerError)
	res.Write([]byte(msg))
	res.Write([]byte(err.Error()))
}
