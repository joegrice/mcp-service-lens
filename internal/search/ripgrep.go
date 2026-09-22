package search

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"mcp-service-lens/internal/config"
)

type Match struct {
	Service string `json:"service"`
	Source  string `json:"source"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Text    string `json:"text"`
}

type Result struct {
	Query      string  `json:"query"`
	Matches    []Match `json:"matches"`
	MatchCount int     `json:"match_count"`
	Truncated  bool    `json:"truncated"`
}

const MaxResultsLimit = 1000

func TraceDocumentation(ctx context.Context, services []config.Service, serviceName, query string, maxResults int) (Result, error) {
	if strings.TrimSpace(query) == "" {
		return Result{}, fmt.Errorf("query must not be empty")
	}
	if maxResults < 1 || maxResults > MaxResultsLimit {
		return Result{}, fmt.Errorf("max results must be between 1 and %d", MaxResultsLimit)
	}

	var targets []config.Service
	for _, service := range services {
		if serviceName == "" || service.Name == serviceName {
			targets = append(targets, service)
		}
	}
	if serviceName != "" && len(targets) == 0 {
		return Result{}, fmt.Errorf("unknown service %q", serviceName)
	}

	results := make(chan []Match, len(targets))
	errs := make(chan error, len(targets))
	var wg sync.WaitGroup
	for _, service := range targets {
		service := service
		wg.Add(1)
		go func() {
			defer wg.Done()
			matches, err := run(ctx, service.Name, "documentation", filepath.Join(service.Root, "docs"), nil, query)
			if err != nil {
				errs <- err
				return
			}
			results <- matches
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	var all []Match
	for matches := range results {
		all = append(all, matches...)
	}
	for err := range errs {
		return Result{}, err
	}
	result := Result{Query: query, MatchCount: len(all)}
	sortMatches(all)
	if len(all) > maxResults {
		result.Truncated = true
		all = all[:maxResults]
	}
	result.Matches = all
	return result, nil
}

func Trace(ctx context.Context, services []config.Service, query string, maxResults int) (Result, error) {
	if strings.TrimSpace(query) == "" {
		return Result{}, fmt.Errorf("query must not be empty")
	}
	if maxResults < 1 || maxResults > MaxResultsLimit {
		return Result{}, fmt.Errorf("max results must be between 1 and %d", MaxResultsLimit)
	}
	type target struct {
		service config.Service
		source  string
		path    string
		exclude []string
	}
	var targets []target
	for _, service := range services {
		var excludes []string
		for _, dir := range service.LogDirectories {
			relative, err := filepath.Rel(service.Root, dir)
			if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				excludes = append(excludes, filepath.ToSlash(relative)+"/**")
			}
		}
		targets = append(targets, target{service, "code", service.Root, excludes})
		for _, dir := range service.LogDirectories {
			targets = append(targets, target{service, "log", dir, nil})
		}
	}

	results := make(chan []Match, len(targets))
	errs := make(chan error, len(targets))
	var wg sync.WaitGroup
	for _, t := range targets {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			matches, err := run(ctx, t.service.Name, t.source, t.path, t.exclude, query)
			if err != nil {
				errs <- err
				return
			}
			results <- matches
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	var all []Match
	for matches := range results {
		for _, match := range matches {
			if match.Source == "code" {
				service := serviceByName(services, match.Service)
				if isInConfiguredLogDirectory(match.File, service.LogDirectories) {
					continue
				}
			}
			all = append(all, match)
		}
	}
	for err := range errs {
		return Result{}, err
	}
	result := Result{Query: query, MatchCount: len(all)}
	sortMatches(all)
	if len(all) > maxResults {
		result.Truncated = true
		all = all[:maxResults]
	}
	result.Matches = all
	return result, nil
}

func sortMatches(matches []Match) {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Service != matches[j].Service {
			return matches[i].Service < matches[j].Service
		}
		if matches[i].Source != matches[j].Source {
			return matches[i].Source < matches[j].Source
		}
		if matches[i].File != matches[j].File {
			return matches[i].File < matches[j].File
		}
		return matches[i].Line < matches[j].Line
	})
}

func serviceByName(services []config.Service, name string) config.Service {
	for _, service := range services {
		if service.Name == name {
			return service
		}
	}
	return config.Service{}
}

func isInConfiguredLogDirectory(path string, directories []string) bool {
	path = filepath.Clean(path)
	for _, directory := range directories {
		directory = filepath.Clean(directory)
		relative, err := filepath.Rel(directory, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func run(ctx context.Context, service, source, root string, excludes []string, query string) ([]Match, error) {
	searchPath := root
	workingDirectory := ""
	if source == "code" {
		workingDirectory = root
		searchPath = "."
	}
	args := []string{"--line-number", "--with-filename", "--no-heading", "--color=never", "--fixed-strings", "--glob", "!.git/**", "--glob", "!vendor/**", "--glob", "!node_modules/**", "--glob", "!dist/**", "--glob", "!build/**", "--glob", "!coverage/**", "--glob", "!tmp/**", query, searchPath}
	if len(excludes) > 0 {
		args = []string{"--line-number", "--with-filename", "--no-heading", "--color=never", "--fixed-strings", "--glob", "!.git/**", "--glob", "!vendor/**", "--glob", "!node_modules/**", "--glob", "!dist/**", "--glob", "!build/**", "--glob", "!coverage/**", "--glob", "!tmp/**"}
		for _, exclude := range excludes {
			args = append(args, "--glob", "!"+exclude)
		}
		args = append(args, query, searchPath)
	}
	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = workingDirectory
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("prepare rg %s: %w", root, err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start rg %s: %w", root, err)
	}

	var matches []Match
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		lineNumber, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		file := filepath.Clean(parts[0])
		if source == "code" {
			file = filepath.Join(root, file)
		}
		matches = append(matches, Match{Service: service, Source: source, File: file, Line: lineNumber, Text: parts[2]})
	}
	scanErr := scanner.Err()
	waitErr := cmd.Wait()
	if scanErr != nil {
		return nil, fmt.Errorf("read rg %s: %w", root, scanErr)
	}
	err = waitErr
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("rg %s: %w: %s", root, err, strings.TrimSpace(stderr.String()))
	}
	return matches, nil
}
