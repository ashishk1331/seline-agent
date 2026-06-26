package tools

import (
	"context"
	"fmt"
	"os"
)

// registerFiles registers read_file and write_file.
func (r *Registry) registerFiles() {
	r.Register(Tool{
		Name:        "read_file",
		Description: "Read the contents of a file.",
		Params:      []Param{{Name: "path", Type: "string", Description: "The path to the file."}},
		StatusMessage: "Reading $path",
		Handler:       readFile,
	})

	r.Register(Tool{
		Name:        "write_file",
		Description: "Write content to a file, replacing any existing content.",
		Params: []Param{
			{Name: "path", Type: "string", Description: "The path to the file."},
			{Name: "content", Type: "string", Description: "The content to write to the file."},
		},
		StatusMessage: "Writing $path",
		Handler:       writeFile,
	})
}

func readFile(_ context.Context, args map[string]any) (string, error) {
	path := argString(args, "path")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func writeFile(_ context.Context, args map[string]any) (string, error) {
	path := argString(args, "path")
	content := argString(args, "content")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s written.", path), nil
}
