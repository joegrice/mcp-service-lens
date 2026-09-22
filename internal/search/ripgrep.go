package search

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"mcp-service-lens/internal/config"
)

const (
	MaxResultsLimit = 1000
	searchWorkers   = 4
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

type target struct {
	service config.Service
	source  string
	path    string
	exclude []string
}

type runResult struct {
	matches    []Match
	matchCount int
	truncated  bool
}

type rgRecord struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		Lines struct {
			Text string `json:"text"`
		} `json:"lines"`
		LineNumber int `json:"line_number"`
	} `json:"data"`
}

func TraceDocumentation(ctx context.Context, services []config.Service, serviceName, query string, maxResults int) (Result, error) {
	if err := validateQuery(query, maxResults); err != nil {
		return Result{}, err
	}
	var targets []target
	for _, service := range services {
		if serviceName == "" || service.Name == serviceName {
			targets = append(targets, target{service: service, source: "documentation", path: filepath.Join(service.Root, "docs")})
		}
	}
	if serviceName != "" && len(targets) == 0 {
		return Result{}, fmt.Errorf("unknown service %q", serviceName)
	}
	return collect(ctx, targets, query, maxResults)
}

func Trace(ctx context.Context, services []config.Service, query string, maxResults int) (Result, error) {
	if err := validateQuery(query, maxResults); err != nil {
		return Result{}, err
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
		targets = append(targets, target{service: service, source: "code", path: service.Root, exclude: excludes})
		for _, dir := range service.LogDirectories {
			targets = append(targets, target{service: service, source: "log", path: dir})
		}
	}
	return collect(ctx, targets, query, maxResults)
}

func validateQuery(query string, maxResults int) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("query must not be empty")
	}
	if maxResults < 1 || maxResults > MaxResultsLimit {
		return fmt.Errorf("max results must be between 1 and %d", MaxResultsLimit)
	}
	return nil
}

func collect(ctx context.Context, targets []target, query string, maxResults int) (Result, error) {
	if len(targets) == 0 {
		return Result{Query: query}, nil
	}
	results := make(chan runResult, len(targets))
	errs := make(chan error, len(targets))
	jobs := make(chan target)
	workerCount := min(searchWorkers, len(targets))
	var wg sync.WaitGroup
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				result, err := run(ctx, t, query, maxResults)
				if err != nil {
					errs <- err
					continue
				}
				results <- result
			}
		}()
	}
	go func() {
		for _, t := range targets {
			jobs <- t
		}
		close(jobs)
		wg.Wait()
		close(results)
		close(errs)
	}()

	var all []Match
	matchCount := 0
	truncated := false
	for result := range results {
		all = append(all, result.matches...)
		matchCount += result.matchCount
		truncated = truncated || result.truncated
	}
	for err := range errs {
		return Result{}, err
	}
	sortMatches(all)
	if len(all) > maxResults {
		truncated = true
		all = all[:maxResults]
	}
	return Result{Query: query, Matches: all, MatchCount: matchCount, Truncated: truncated}, nil
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
		if matches[i].Line != matches[j].Line {
			return matches[i].Line < matches[j].Line
		}
		return matches[i].Text < matches[j].Text
	})
}

func run(ctx context.Context, t target, query string, maxMatches int) (runResult, error) {
	searchPath := t.path
	workingDirectory := ""
	if t.source == "code" {
		workingDirectory = t.path
		searchPath = "."
	}
	args := []string{
		"--json", "--hidden", "--fixed-strings",
		"--glob", "!.git/**", "--glob", "!vendor/**", "--glob", "!node_modules/**",
		"--glob", "!dist/**", "--glob", "!build/**", "--glob", "!coverage/**", "--glob", "!tmp/**",
	}
	for _, exclude := range t.exclude {
		args = append(args, "--glob", "!"+exclude)
	}
	args = append(args, query, searchPath)

	cmd := exec.CommandContext(ctx, "rg", args...)
	cmd.Dir = workingDirectory
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return runResult{}, fmt.Errorf("prepare rg %s: %w", t.path, err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return runResult{}, fmt.Errorf("start rg %s: %w", t.path, err)
	}

	result := runResult{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var record rgRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil || record.Type != "match" {
			continue
		}
		result.matchCount++
		if len(result.matches) < maxMatches {
			file := filepath.Clean(record.Data.Path.Text)
			if t.source == "code" {
				file = filepath.Join(t.path, file)
			}
			result.matches = append(result.matches, Match{
				Service: t.service.Name,
				Source:  t.source,
				File:    file,
				Line:    record.Data.LineNumber,
				Text:    strings.TrimSuffix(strings.TrimSuffix(record.Data.Lines.Text, "\n"), "\r"),
			})
		}
	}
	if err := scanner.Err(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return runResult{}, fmt.Errorf("read rg %s: %w", t.path, err)
	}
	waitErr := cmd.Wait()
	if waitErr != nil {
		if exit, ok := waitErr.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return result, nil
		}
		return runResult{}, fmt.Errorf("rg %s: %w: %s", t.path, waitErr, strings.TrimSpace(stderr.String()))
	}
	result.truncated = result.matchCount > len(result.matches)
	return result, nil
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
