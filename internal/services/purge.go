package services

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mic-360/wimo/internal/state"
)

type PurgeService struct {
	logger *Logger
}

func NewPurgeService(logger *Logger) *PurgeService {
	return &PurgeService{logger: logger}
}

func (p *PurgeService) ScanProjects(ctx context.Context, roots []string, maxDepth int) ([]state.Project, error) {
	if maxDepth <= 0 {
		maxDepth = 6
	}
	seen := map[string]bool{}
	projects := []state.Project{}
	for _, root := range roots {
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		p.scanRoot(ctx, abs, abs, 0, maxDepth, seen, &projects)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].TotalArtifactBytes == projects[j].TotalArtifactBytes {
			return strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
		}
		return projects[i].TotalArtifactBytes > projects[j].TotalArtifactBytes
	})
	return projects, nil
}

func (p *PurgeService) scanRoot(ctx context.Context, root, current string, depth, maxDepth int, seen map[string]bool, projects *[]state.Project) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	if depth > maxDepth {
		return
	}
	if skipDir(filepath.Base(current)) {
		return
	}
	ecosystems := detectProject(current)
	if len(ecosystems) > 0 && !seen[current] {
		seen[current] = true
		*projects = append(*projects, p.buildProject(current, ecosystems))
	}
	entries, err := os.ReadDir(current)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p.scanRoot(ctx, root, filepath.Join(current, entry.Name()), depth+1, maxDepth, seen, projects)
	}
}

func (p *PurgeService) buildProject(root string, ecosystems []string) state.Project {
	project := state.Project{ID: slug(root), Name: filepath.Base(root), Root: root, Ecosystems: ecosystems, LastScan: time.Now()}
	project.Analyzer = analyzeProject(root)
	project.Artifacts = scanArtifacts(project.ID, root)
	for index := range project.Artifacts {
		project.TotalArtifactBytes += project.Artifacts[index].Size
	}
	project.ArtifactCount = len(project.Artifacts)
	return project
}

func (p *PurgeService) Purge(ctx context.Context, artifacts []state.Artifact) (OperationReport, error) {
	report := OperationReport{Title: "Purge", Message: "Artifact purge complete"}
	for _, artifact := range artifacts {
		if !artifact.Selected {
			continue
		}
		select {
		case <-ctx.Done():
			return report, ctx.Err()
		default:
		}
		if err := os.RemoveAll(artifact.Path); err != nil {
			report.Errors = append(report.Errors, artifact.Label+": "+err.Error())
			continue
		}
		report.Count++
		report.Bytes += artifact.Size
	}
	return report, nil
}

func detectProject(dir string) []string {
	markers := map[string]string{
		"package.json":     "Node",
		"pnpm-lock.yaml":   "Node",
		"yarn.lock":        "Node",
		"go.mod":           "Go",
		"Cargo.toml":       "Rust",
		"pubspec.yaml":     "Flutter",
		"pyproject.toml":   "Python",
		"requirements.txt": "Python",
		"pom.xml":          "Java",
		"build.gradle":     "Gradle",
		"settings.gradle":  "Gradle",
		"CMakeLists.txt":   "CMake",
		"composer.json":    "PHP",
		"*.csproj":         ".NET",
		"*.fsproj":         ".NET",
		"*.sln":            ".NET",
	}
	found := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		name := entry.Name()
		for marker, ecosystem := range markers {
			if strings.Contains(marker, "*") {
				matched, _ := filepath.Match(marker, name)
				if matched {
					found[ecosystem] = true
				}
				continue
			}
			if name == marker {
				found[ecosystem] = true
			}
		}
	}
	result := make([]string, 0, len(found))
	for ecosystem := range found {
		result = append(result, ecosystem)
	}
	sort.Strings(result)
	return result
}

func analyzeProject(root string) []state.DirectoryUsage {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	items := make([]state.DirectoryUsage, 0, len(entries))
	total := int64(0)
	for _, entry := range entries {
		name := entry.Name()
		if skipDir(name) {
			continue
		}
		fullPath := filepath.Join(root, name)
		size, _ := pathUsage(fullPath)
		if size == 0 {
			continue
		}
		total += size
		items = append(items, state.DirectoryUsage{Name: name, Path: fullPath, Size: size})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Size > items[j].Size })
	if len(items) > 7 {
		items = items[:7]
	}
	for index := range items {
		if total > 0 {
			items[index].Percent = float64(items[index].Size) / float64(total) * 100
		}
	}
	return items
}

func scanArtifacts(projectID, root string) []state.Artifact {
	patterns := map[string]string{
		"node_modules":        "Node dependencies",
		".next":               "Next.js cache",
		".nuxt":               "Nuxt cache",
		"dist":                "Build output",
		"build":               "Build output",
		"out":                 "Output directory",
		"coverage":            "Coverage output",
		"target":              "Compiled artifacts",
		"__pycache__":         "Python bytecode",
		".pytest_cache":       "Pytest cache",
		".mypy_cache":         "Mypy cache",
		".ruff_cache":         "Ruff cache",
		".venv":               "Virtual environment",
		"venv":                "Virtual environment",
		".dart_tool":          "Flutter tool cache",
		".gradle":             "Gradle cache",
		"bin":                 "Binary output",
		"obj":                 "Object output",
		"vendor":              "Vendor cache",
		"Pods":                "CocoaPods cache",
		"cmake-build-debug":   "CMake debug output",
		"cmake-build-release": "CMake release output",
	}
	artifacts := []state.Artifact{}
	rootDepth := strings.Count(filepath.Clean(root), string(filepath.Separator))
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		depth := strings.Count(filepath.Clean(path), string(filepath.Separator)) - rootDepth
		if depth > 5 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		typeLabel, ok := patterns[name]
		if !ok {
			if entry.IsDir() {
				return nil
			}
			return nil
		}
		size, _ := pathUsage(path)
		if size == 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, statErr := os.Stat(path)
		ageDays := 0
		if statErr == nil {
			ageDays = int(time.Since(info.ModTime()).Hours() / 24)
		}
		artifacts = append(artifacts, state.Artifact{ID: slug(path), ProjectID: projectID, Label: strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator)), Path: path, Type: typeLabel, Size: size, AgeDays: ageDays, Selected: ageDays > 2})
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Size > artifacts[j].Size })
	return artifacts
}

func skipDir(name string) bool {
	blocked := map[string]bool{".git": true, ".idea": true, ".vscode": true, "node_modules": true}
	return blocked[name]
}

func slug(value string) string {
	clean := strings.ToLower(filepath.Clean(value))
	clean = strings.NewReplacer(string(filepath.Separator), "-", ":", "", " ", "-").Replace(clean)
	clean = strings.Trim(clean, "-")
	return clean
}
