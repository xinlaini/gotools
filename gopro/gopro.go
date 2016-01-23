package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"

	"gen/pb/makepro"
)

var (
	dryRun    = flag.Bool("dry_run", false, "view generated makefile only")
	protoRoot = os.Getenv("PROTOROOT")
	pbGen     = filepath.Join(os.Getenv("GOPATH"), "src", "gen", "pb")
)

type dependency struct {
	mkTarget    string
	path        string
	protoTarget string
	src         []string
	goOutMap    string
}

type walkState struct {
	buf     *bytes.Buffer
	current *dependency
	visited map[string]bool
	stack   []string
	depInfo map[string]*dependency
}

func makeLine(current *dependency, mkDeps []*dependency) string {
	var (
		depTargets   []string
		depGoOutMaps []string
	)
	for _, dep := range mkDeps {
		depTargets = append(depTargets, dep.mkTarget)
		depGoOutMaps = append(depGoOutMaps, dep.goOutMap)
	}
	// Format current's goOutMap using its src.
	var mp []string
	for _, s := range current.src {
		mp = append(mp, fmt.Sprintf(
			"M%s=%s", filepath.Join(current.path, s), filepath.Join("gen", "pb", current.path, current.protoTarget)))
	}
	current.goOutMap = strings.Join(mp, ",")
	genDir := filepath.Join(pbGen, current.path, current.protoTarget)
	return fmt.Sprintf(
		"%s: %s\n\tcd %s && mkdir -p %s && protoc --proto_path=%s/ --proto_path=./ --go_out=%s:%s %s\n",
		current.mkTarget, strings.Join(depTargets, " "), filepath.Join(protoRoot, current.path),
		genDir, protoRoot, strings.Join(depGoOutMaps, ","), genDir, strings.Join(current.src, " "))
}

func gopro(fullTarget, parentDepFile string, state *walkState) error {
	current := state.depInfo[fullTarget]
	targetParts := strings.Split(fullTarget, ":")
	if len(targetParts) != 2 {
		errMsg := fmt.Sprintf("Invalid target '%s', must be in the form of 'path:target'", fullTarget)
		if parentDepFile != "" {
			return fmt.Errorf("%s: %s", parentDepFile, errMsg)
		}
		return errors.New(errMsg)
	}
	current.path = targetParts[0]
	current.protoTarget = targetParts[1]

	state.visited[fullTarget] = true
	state.stack = append(state.stack, fullTarget)
	defer func() {
		delete(state.visited, fullTarget)
		state.stack = state.stack[0 : len(state.stack)-1]
	}()

	depFile := filepath.Join(protoRoot, current.path, "dep.pb")
	depBytes, err := ioutil.ReadFile(depFile)
	if err != nil {
		return fmt.Errorf("Failed to read '%s'", depFile)
	}
	make := &makepro_proto.Make{}
	if err := proto.UnmarshalText(string(depBytes), make); err != nil {
		return fmt.Errorf("Failed to unmarshal '%s': %s", depFile, err)
	}
	found := false
	for _, build := range make.Build {
		if build.GetTarget() == current.protoTarget {
			if len(build.Src) == 0 {
				return fmt.Errorf("%s must have at least one src", fullTarget)
			}
			current.src = build.Src
			var mkDeps []*dependency
			found = true
			for i, dep := range build.Dep {
				if _, visited := state.visited[dep]; visited {
					return fmt.Errorf("Circular dependency: %s -> %s", strings.Join(state.stack, " -> "), dep)
				}
				var (
					depInfo         *dependency
					targetProcessed bool
				)
				depInfo, targetProcessed = state.depInfo[dep]
				if !targetProcessed {
					depInfo = &dependency{
						mkTarget: fmt.Sprintf("%s_%d", current.mkTarget, i),
					}
					state.depInfo[dep] = depInfo
					if err := gopro(dep, depFile, state); err != nil {
						return err
					}
				}
				mkDeps = append(mkDeps, depInfo)
			}
			if _, err := state.buf.WriteString(makeLine(current, mkDeps)); err != nil {
				return fmt.Errorf("Buffer write failed: %s", err)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("Cannot find target '%s' from '%s'", current.protoTarget, depFile)
	}
	return nil
}

func main() {
	flag.Parse()

	logger := xlog.NewPlainLogger()
	usage := fmt.Sprintf("Usage: gopro [--dry_run] path:target")

	if len(flag.Args()) != 1 {
		logger.Fatal(usage)
	}

	root := &dependency{mkTarget: "goal"}
	state := walkState{
		buf:     &bytes.Buffer{},
		current: root,
		visited: make(map[string]bool),
		depInfo: map[string]*dependency{flag.Arg(0): root},
	}
	if err := gopro(flag.Arg(0), "", &state); err != nil {
		logger.Fatal(err)
	}

	if *dryRun {
		logger.Info(string(state.buf.Bytes()))
	} else {
		makefile, err := ioutil.TempFile("", "Makefile")
		if err != nil {
			logger.Fatalf("Failed to create temporary Makefile: %s", err)
		}
		defer func() {
			makefile.Close()
			os.Remove(makefile.Name())
		}()
		if _, err := io.Copy(makefile, state.buf); err != nil {
			logger.Fatalf("Failed to write to temporary Makefile: %s", err)
		}

		cmd := exec.Command("make", "-f", makefile.Name(), "goal")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		logger.Info("Running make")
		if err := cmd.Run(); err != nil {
			logger.Fatalf("make failed: %s", err)
		}
		logger.Info("make SUCCEEDED")
	}
}
