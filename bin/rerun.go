package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	TestCases []JUnitTestCase `xml:"testcase"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Time      float64         `xml:"time,attr"`
}

type JUnitTestCase struct {
	Name           string               `xml:"name,attr"`
	ClassName      string               `xml:"classname,attr"`
	FailureMessage *JUnitFailureMessage `xml:"failure,omitempty"`
	Skipped        *JUnitSkipped        `xml:"skipped,omitempty"`
	Time           float64              `xml:"time,attr"`
}

type JUnitFailureMessage struct {
	Type    string `xml:"type,attr"`
	Message string `xml:",chardata"`
}

type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
}

type JUnitReporter struct {
	suite         JUnitTestSuite
	filename      string
	testSuiteName string
}

func deleteJUnitFiles(dirPath, matchString string) {
	err := filepath.Walk(dirPath,
		func(filepath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			ok, _ := path.Match(matchString, info.Name())
			if ok {
				os.RemoveAll(filepath)
			}

			return nil
		})
	if err != nil {
		fmt.Printf("Searching File error: %v\n", err)
		os.Exit(1)
	}
}

func readJUnitFiles(dirPath, matchString string) (failures, stashes []JUnitTestCase) {
	var err error
	var matchedFiles []string

	err = filepath.Walk(dirPath,
		func(filepath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			ok, _ := path.Match(matchString, info.Name())
			if ok {
				matchedFiles = append(matchedFiles, filepath)
			}

			return nil
		})
	if err != nil {
		fmt.Printf("Searching File error: %v\n", err)
		os.Exit(1)
	}

	var contents []byte
	for _, file := range matchedFiles {
		contents, err = ioutil.ReadFile(file)
		if err != nil {
			fmt.Printf("Reading File error: %v\n", err)
			os.Exit(1)
		}

		var testSuite JUnitTestSuite
		err = xml.Unmarshal(contents, &testSuite)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			os.Exit(1)
		}

		for _, v := range testSuite.TestCases {
			if v.FailureMessage != nil {
				failures = append(failures, v)
			} else {
				stashes = append(stashes, v)
			}
		}
	}

	return failures, stashes
}

func mergeXml(dirPath string, matchString string, stashes []JUnitTestCase) []byte {
	newFailures, newStashes := readJUnitFiles(dirPath, matchString)

	mergedTestCases := append(newFailures, stashes...)
	for _, v := range newStashes {
		if v.Skipped == nil {
			mergedTestCases = append(mergedTestCases, v)
			fmt.Println(v.Name)
		}
	}

	var testSuite JUnitTestSuite
	testSuite.TestCases = mergedTestCases
	testSuite.Tests = func(mergedTestCases []JUnitTestCase) int {
		i := 0
		for _, v := range mergedTestCases {
			if v.Skipped == nil {
				i += 1
			}
		}
		return i
	}(mergedTestCases)
	testSuite.Failures = len(newFailures)

	resultContents, err := xml.MarshalIndent(testSuite, "  ", "    ")
	if err != nil {
		fmt.Printf("Marshal error: %v\n", err)
		os.Exit(1)
	}

	return append([]byte(xml.Header), resultContents...)
}

func main() {
	failures, stashes := readJUnitFiles(`results`, `junit-*.xml`)
	failuresStr := []string{}

	if len(failures) == 0 {
		fmt.Println("No tests for rerun.")
		os.Exit(0)
	}

	fmt.Println("[rerun tests below]")
	for _, v := range failures {
		quoteName := regexp.QuoteMeta(v.Name)
		failuresStr = append(failuresStr, quoteName)
		fmt.Println(quoteName)
	}
	focusParam := "-focus=" + strings.Join(failuresStr, "|")
	fmt.Printf("[rerun %d tests with %s]\n", len(failuresStr), focusParam)

	deleteJUnitFiles(`results`, `junit-*.xml`)
	cmd := exec.Command("ginkgo", "-r", "-keepGoing=true", "-slowSpecThreshold=120", focusParam)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	mergedXml := mergeXml(`results`, `junit-*.xml`, stashes)
	deleteJUnitFiles(`results`, `junit-*.xml`)
	ioutil.WriteFile(`results/junit-merged.xml`, mergedXml, 0644)

	if cmd.ProcessState.Success() == true {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
