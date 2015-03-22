package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type GjeTestSuite struct {
	suite.Suite
}

func (s *GjeTestSuite) SetupTest() {
}

func (s *GjeTestSuite) compile(file string) {
	cmd := exec.Command("gojsen", file)
	cmd.Stderr = os.Stdout

	err := cmd.Run()
	if err != nil {
		s.Fail("", "Compilation of %v failed.", file)
	}
}

func (s *GjeTestSuite) run(target string) []string {
	file := filepath.Join("tests", target+".go")
	s.compile(file)
	jsFile := file + ".js"

	cmd := exec.Command("node", jsFile)
	cmd.Stderr = os.Stdout
	b, err := cmd.Output()
	if err != nil {
		s.Fail("", "Running of %v failed.", jsFile)
	}

	return strings.Split(string(b), "\n")
}

func (s *GjeTestSuite) TestBasicInstructions() {
	output := s.run("basic_instructions")
	s.Equal(output[0], "12")
	s.Equal(output[1], "1")
	s.Equal(output[2], "2")
}

func TestCompilations(t *testing.T) {
	b, err := exec.Command("go", "get").CombinedOutput()
	fmt.Println(string(b))
	if err != nil {
		log.Fatal(err)
	}

	suite.Run(t, new(GjeTestSuite))
}
