package linux

import "os"

// classifyPaths splits paths into existing directories and existing files.
// Landlock rules are typed, and a file passed as a directory rule makes the
// library reject the whole ruleset, which would silently drop the second
// enforcement layer. Missing paths are skipped.
func classifyPaths(paths []string, stat func(string) (os.FileInfo, error)) (dirs, files []string) {
	for _, p := range paths {
		st, err := stat(p)
		if err != nil {
			continue
		}
		if st.IsDir() {
			dirs = append(dirs, p)
		} else {
			files = append(files, p)
		}
	}
	return dirs, files
}
