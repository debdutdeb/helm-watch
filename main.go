package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	//"runtime"
	"flag"
	"syscall"
	"time"

	"github.com/fsnotify/fsevents"
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

		err := decoder.Decode(&node)

		var m map[string]interface{}

		if err := node.Decode(&m); err != nil {
			return nil, err
		}

		manifests = append(manifests, m)

		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return manifests, cmd.Wait()
}

func main() {
	chart := flag.String("chart", "", "--chart <path to local chart>")

	values := flag.String("values", "", "--values <path to values file>")

	name := flag.String("name", "", "--name <name of installation>")

	kind := flag.String("kind", "", "--kind <kind of resource to watch>")

	resource := flag.String("resource", "", "--resource <name of resource to watch>")

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

	*kind = strings.ToLower(*kind)

	stream := fsevents.EventStream{
		Paths:   []string{*chart, *values},
		Latency: 500 * time.Millisecond,
		Flags:   fsevents.FileEvents,
	}

	stream.Start()

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

	printManifest(*chart, *name, *values, *kind, *resource)

	for msg := range stream.Events {
		for _, event := range msg {
			if event.Flags&fsevents.ItemModified == 0 {
				continue
			}

			fmt.Printf("chart modified, regenerating template for %s/%s\n", *kind, *resource)

			fmt.Println("---")
			printManifest(*chart, *name, *values, *kind, *resource)
			fmt.Println("---")
		}
	}
}

func printManifest(chartPath string, appName string, valuesPath string, kind string, resourceName string) {
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

		if name != resourceName {
			continue
		}

		output, err := yaml.Marshal(manifest)
		if err != nil {
			fmt.Printf("invalid YAML: %v\n", err)
			break
		}

		fmt.Println(string(output))

		break
	}
}
