package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	//"runtime"
	"flag"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

func template(path string, name string, values string) ([]map[string]interface{}, error) {
	args := append(flag.Args(), "template", name, path, "-f", values)

	cmd := exec.Command("helm", args...)

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(r)

	var manifests []map[string]interface{}

	for {
		var node yaml.Node

		if err := decoder.Decode(&node); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		var m map[string]interface{}

		if err := node.Decode(&m); err != nil {
			return nil, err
		}

		// must always have a kind
		if m["kind"] == nil || m["apiVersion"] == nil {
			continue
		}

		manifests = append(manifests, m)
	}

	return manifests, cmd.Wait()
}

func main() {
	chart := flag.String("chart", "", "--chart <path to local chart>")

	values := flag.String("values", "", "--values <path to values file>")

	name := flag.String("name", "", "--name <name of installation>")

	kind := flag.String("kind", "", "--kind <kind of resource to watch> (no shorthands)")

	resource := flag.String("resource", "", "--resource <name of resource to watch> (regex)")

	flag.Parse()

	if chart == nil {
		fmt.Printf("--chart missing\n")
		os.Exit(1)
	}

	if values == nil {
		fmt.Printf("--values missing\n")
		os.Exit(1)
	}

	if name == nil {
		fmt.Printf("--name missing\n")
		os.Exit(1)
	}

	if kind == nil {
		fmt.Printf("--kind missing\n")
		os.Exit(1)
	}

	if resource == nil {
		fmt.Printf("--resource missing\n")
		os.Exit(1)
	}

	resourceRegex := regexp.MustCompile(*resource)

	*kind = strings.ToLower(*kind)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	if err := watchAll(watcher, []string{*values, *chart}); err != nil {
		panic(err)
	}

	fmt.Printf("watching chart %s for changes, displaying %s/%s\n", *chart, *kind, *resource)

	//go func() {
	//	t := time.NewTicker(time.Second * 5)

	//	for {
	//		<-t.C
	//		runtime.GC()
	//	}
	//}()

	sig := make(chan os.Signal, 1)

	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig

		fmt.Println("exiting..")

		os.Exit(0)
	}()

	printManifest(*chart, *name, *values, *kind, resourceRegex)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return

			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			fmt.Printf("chart modified, regenerating template for %s/%s\n", *kind, *resource)

			fmt.Println("---")
			printManifest(*chart, *name, *values, *kind, resourceRegex)
			fmt.Println("---")
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}

			fmt.Printf("error: %v\n", err)
			os.Exit(1)
		}
	}

}

func printManifest(chartPath string, appName string, valuesPath string, kind string, resourceRegex *regexp.Regexp) {
	manifests, err := template(chartPath, appName, valuesPath)
	if err != nil {
		fmt.Printf("error generating manifests: %v\n", err)
		return
	}

	for _, manifest := range manifests {
		if strings.ToLower(manifest["kind"].(string)) != kind {
			continue
		}

		name := manifest["metadata"].(map[string]interface{})["name"].(string)

		if !resourceRegex.Match([]byte(name)) {
			continue
		}

		output, err := yaml.Marshal(manifest)
		if err != nil {
			fmt.Printf("invalid YAML: %v\n", err)
			break
		}

		fmt.Println(string(output))
	}
}

func watchAll(watcher *fsnotify.Watcher, paths []string) error {
	var walk fs.WalkDirFunc = func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		watcher.Add(path)

		return nil
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to understand if %s is a file or directory: %v", path, err)
		}

		if !info.IsDir() {
			watcher.Add(path)
			continue
		}

		if err := filepath.WalkDir(path, walk); err != nil {
			return err
		}
	}

	return nil
}
