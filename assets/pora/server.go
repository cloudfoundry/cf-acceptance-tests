package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"encoding/json"
	"strings"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/", hello)
	http.HandleFunc("/env", env)
	http.HandleFunc("/write", write)
	http.HandleFunc("/create", createFile)
	http.HandleFunc("/read/", readFile)
	http.HandleFunc("/chmod/", chmodFile)
	http.HandleFunc("/delete/", deleteFile)
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

	err := <-errCh
	if err != nil {
		panic(err)
	}
}

type VCAPServices struct {
    NFS []struct {
        Credentials    map[string]string   `json:"credentials"`
        Label          string              `json:"label"`
        Name           string              `json:"name"`
        Plan           string              `json:"plan"`
        Provider       string              `json:"provider"`
        SyslogDrainURL string              `json:"syslog_drain_url"`
        Tags           []string            `json:"tags"`
        VolumeMounts   []map[string]string `json:"volume_mounts"`
    } `json:"nfs"`
}

type VCAPApplication struct {
	InstanceIndex int `json:"instance_index"`
}

func hello(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "instance index: %s", os.Getenv("INSTANCE_INDEX"))
}

func LoadVCAPServices(input string) (VCAPServices, error) {
    var config VCAPServices
    configFile := strings.NewReader(input)

    jsonParser := json.NewDecoder(configFile)
    err := jsonParser.Decode(&config)

    return config, err
}

func getPath() string {
    vcapEnv := os.Getenv("VCAP_SERVICES")
    config, err := LoadVCAPServices(vcapEnv)
    if err != nil {
        panic(err)
    }

    // the first mount will be used if more than one is attached
    return config.NFS[0].VolumeMounts[0]["container_dir"]
}

func write(res http.ResponseWriter, req *http.Request) {
	mountPointPath := getPath() + "/poratest-" + randomString(10)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Writing \n"))
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	body, err := ioutil.ReadFile(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Reading \n"))
		res.Write([]byte(err.Error()))
		return
	}

	err = os.Remove(mountPointPath)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte("Deleting \n"))
		res.Write([]byte(err.Error()))
		return
	}

	res.Write(body)
	return
}

func createFile(res http.ResponseWriter, req *http.Request) {
	fileName := "pora" + randomString(10)
	mountPointPath := filepath.Join(getPath(), fileName)

	d1 := []byte("Hello Persistent World!\n")
	err := ioutil.WriteFile(mountPointPath, d1, 0644)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(fileName))
	return
}

func readFile(res http.ResponseWriter, req *http.Request) {
	parts := strings.Split(req.URL.Path, "/")
	fileName := parts[len(parts) - 1]
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
	fileName := parts[len(parts) - 2]
	mountPointPath := filepath.Join(getPath(), fileName)
	mode := parts[len(parts) - 1]
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
	fileName := parts[len(parts) - 1]
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

var isSeeded=false
func randomString(n int) string {
	if (!isSeeded) {
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
