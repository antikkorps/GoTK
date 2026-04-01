package detect

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Node.js module resolution internal stack frames
	nodeRequireStack = regexp.MustCompile(`^\s+at (Module\._resolveFilename|Module\._load|Module\.require|require \(internal)`)
	// Node.js experimental warnings
	nodeExperimentalWarn = regexp.MustCompile(`^\(node:\d+\) ExperimentalWarning:`)
	// Node.js deprecation warnings (keep first, count rest)
	nodeDeprecationWarn = regexp.MustCompile(`^\(node:\d+\) \[DEP\d+\] DeprecationWarning:`)
	// Webpack/Vite/esbuild build noise
	webpackProgress  = regexp.MustCompile(`^\s*\d+%\s+(building|sealing|emitting|optimizing)`)
	webpackModule    = regexp.MustCompile(`^\s*(asset|chunk|modules by path|orphan modules|runtime modules|cacheable modules)`)
	webpackAssetSize = regexp.MustCompile(`^\s+\S+\.(js|css|map)\s+[\d.]+ [kKMG]iB`)
	viteOptimize     = regexp.MustCompile(`^(Optimized dependencies|Pre-bundling|✓ \d+ modules)`)
	// Node.js internal trace lines (not app frames)
	// Matches "at ... (node:internal/...)" or "at ... (node:...)" or "at require (node:...)"
	nodeInternalTrace = regexp.MustCompile(`^\s+at .+\(node:`)
	// npm/npx script runner noise
	npxResolve = regexp.MustCompile(`^(Need to install|Ok to proceed|npm warn exec)`)
)

// compressNodeOutput compresses node/npx/tsx/deno runtime output.
// Preserves: error messages, app stack frames, build results.
// Removes: module resolution internals, experimental warnings, build progress, internal stack frames.
func compressNodeOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	experimentalCount := 0
	deprecationCount := 0
	firstDeprecation := ""
	webpackProgressCount := 0
	webpackModuleCount := 0
	internalFrameCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			flushNodeCounters(&result, &experimentalCount, &deprecationCount, &firstDeprecation,
				&webpackProgressCount, &webpackModuleCount, &internalFrameCount)
			result = append(result, line)
			continue
		}

		// Count and skip experimental warnings
		if nodeExperimentalWarn.MatchString(trimmed) {
			experimentalCount++
			continue
		}

		// Compress deprecation warnings: keep first, count rest
		if nodeDeprecationWarn.MatchString(trimmed) {
			deprecationCount++
			if deprecationCount == 1 {
				firstDeprecation = line
			}
			continue
		}

		// Skip webpack/vite build progress
		if webpackProgress.MatchString(trimmed) {
			webpackProgressCount++
			continue
		}

		// Skip verbose webpack module/asset listings
		if webpackModule.MatchString(trimmed) || webpackAssetSize.MatchString(trimmed) {
			webpackModuleCount++
			continue
		}

		// Skip vite optimization noise
		if viteOptimize.MatchString(trimmed) {
			continue
		}

		// Skip npx resolve prompts
		if npxResolve.MatchString(trimmed) {
			continue
		}

		// Count internal stack frames (compress separately from app frames)
		if nodeInternalTrace.MatchString(line) {
			internalFrameCount++
			continue
		}

		// Skip module resolution internal stack frames
		if nodeRequireStack.MatchString(trimmed) {
			internalFrameCount++
			continue
		}

		// Flush counters before real content
		flushNodeCounters(&result, &experimentalCount, &deprecationCount, &firstDeprecation,
			&webpackProgressCount, &webpackModuleCount, &internalFrameCount)

		// Keep everything else: errors, app stack frames, build results, actual output
		result = append(result, line)
	}

	flushNodeCounters(&result, &experimentalCount, &deprecationCount, &firstDeprecation,
		&webpackProgressCount, &webpackModuleCount, &internalFrameCount)

	return strings.Join(result, "\n")
}

func flushNodeCounters(result *[]string, experimentalCount, deprecationCount *int, firstDeprecation *string,
	webpackProgressCount, webpackModuleCount, internalFrameCount *int) {
	if *experimentalCount > 0 {
		*result = append(*result, "("+strconv.Itoa(*experimentalCount)+" experimental warnings)")
		*experimentalCount = 0
	}
	if *deprecationCount > 0 {
		*result = append(*result, *firstDeprecation)
		if *deprecationCount > 1 {
			*result = append(*result, "... and "+strconv.Itoa(*deprecationCount-1)+" more deprecation warnings")
		}
		*deprecationCount = 0
		*firstDeprecation = ""
	}
	if *webpackProgressCount > 0 {
		*result = append(*result, "("+strconv.Itoa(*webpackProgressCount)+" build progress updates)")
		*webpackProgressCount = 0
	}
	if *webpackModuleCount > 0 {
		*result = append(*result, "("+strconv.Itoa(*webpackModuleCount)+" module/asset details)")
		*webpackModuleCount = 0
	}
	if *internalFrameCount > 0 {
		*result = append(*result, "    [... "+strconv.Itoa(*internalFrameCount)+" node internal frames]")
		*internalFrameCount = 0
	}
}
