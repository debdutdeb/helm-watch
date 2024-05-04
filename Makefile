build:
	for os in "darwin" "linux"; do \
		for arch in "amd64" "arm64"; do \
			GOOS=$$os GOARCH=$$arch go build -o watch/helm-watch-$$os-$$arch .; \
		done; \
	done;
