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

func template(path, name string) ([]map[string]interface{}, error) {
	args := []string{"template"}

	if name != "" {
		args = append(args, name, path)
	} else {
		args = append(args, path)
	}

	args = append(args, flag.Args()...)

	cmd := exec.Command("helm", args...)

	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = os.Stderr

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

	kinds := flag.String("kinds", "", "--kinds <kind of resource to watch> (no shorthands, separated by comma)")

	names := flag.String("names", "", "--names <name of resource to watch> (regex, separated by comma)")

	releaseName := flag.String("release-name", "", "--release-name <name of release> or use \"-- --generate-name\"")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: helm watch --chart <chart> --kinds <kinds> --names <resources> [--release-name [release]] -- [optional args for \"helm template\" command\n", os.Args[0])

		flag.PrintDefaults()
	}

	flag.Parse()

	tracking := []string{*chart}

	args := flag.Args()

	for i := 0; i < len(args); i++ {
		if args[i] == "-f" || args[i] == "--values" {
			for _, file := range strings.Split(args[i+1], ",") {
				tracking = append(tracking, file)
			}
			break
		}
	}

	if chart == nil {
		fmt.Printf("--chart missing\n")
		os.Exit(1)
	}

	if kinds == nil {
		fmt.Printf("--kinds missing\n")
		os.Exit(1)
	}

	if names == nil {
		fmt.Printf("--names missing\n")
		os.Exit(1)
	}

	var kindToName = make(map[string]*regexp.Regexp)

	{
		kinds := strings.Split(*kinds, ",")

		names := strings.Split(*names, ",")

		for i, kind := range kinds {
			kindToName[kind] = regexp.MustCompile(names[i])
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	if err := watchAll(watcher, tracking); err != nil {
		panic(err)
	}

	fmt.Printf("watching chart %s for changes, displaying %s/%s\n", *chart, *kinds, *names)

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

	printManifest(*chart, *releaseName, kindToName)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return

			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			fmt.Printf("chart modified, regenerating template for %s/%s\n", *kinds, *names)

			fmt.Println("---")
			printManifest(*chart, *releaseName, kindToName)
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

func printManifest(chartPath, releaseName string, tracking map[string]*regexp.Regexp) {
	manifests, err := template(chartPath, releaseName)
	if err != nil {
		fmt.Printf("error generating manifests: %v\n", err)
		return
	}

	for _, manifest := range manifests {

		name := manifest["metadata"].(map[string]interface{})["name"].(string)

		for kind, resource := range tracking {
			if strings.ToLower(manifest["kind"].(string)) != kind {
				continue
			}

			if !resource.Match([]byte(name)) {
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
