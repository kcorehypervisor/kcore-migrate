// Package convert shells out to virt-v2v or qemu-img. It does not interpret
// disk formats itself.
package convert

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options selects one exported disk or guest and the kcore image format.
type Options struct {
	Input  string
	Output string
	Format string
	Tool   string
}

// Command is the process convert will run.
type Command struct {
	Name   string
	Args   []string
	Result string
	// Directory is true when Result is the virt-v2v output directory.
	Directory bool
}

// Plan chooses virt-v2v or qemu-img and returns the argv. It does not run it.
func Plan(opt Options) (Command, error) {
	if strings.TrimSpace(opt.Input) == "" {
		return Command{}, fmt.Errorf("convert: --input is required")
	}
	format := opt.Format
	if format == "" {
		format = "qcow2"
	}
	if format != "qcow2" && format != "raw" {
		return Command{}, fmt.Errorf("convert: --format must be qcow2 or raw")
	}
	tool, err := chooseTool(opt.Input, opt.Tool)
	if err != nil {
		return Command{}, err
	}
	input := opt.Input
	switch tool {
	case "virt-v2v":
		dir := opt.Output
		if dir == "" {
			dir = "converted"
		}
		return Command{
			Name:      "virt-v2v",
			Args:      []string{"-i", virtV2VMode(input), input, "-o", "disk", "-os", dir, "-of", format},
			Result:    dir,
			Directory: true,
		}, nil
	case "qemu-img":
		output := opt.Output
		if output == "" {
			stem := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
			output = filepath.Join("converted", stem+"."+format)
		}
		if filepath.Clean(output) == filepath.Clean(input) {
			return Command{}, fmt.Errorf("convert: output path is the input path")
		}
		args := []string{"convert"}
		if src := qemuFormat(input); src != "" {
			args = append(args, "-f", src)
		}
		args = append(args, "-O", format, input, output)
		return Command{Name: "qemu-img", Args: args, Result: output}, nil
	default:
		return Command{}, fmt.Errorf("convert: unknown tool %q", tool)
	}
}

// Run plans the command, creates the output directory, and runs the tool.
func Run(ctx context.Context, opt Options) (Command, error) {
	return run(ctx, opt, execCommand)
}

func run(ctx context.Context, opt Options, execFn func(context.Context, string, ...string) error) (Command, error) {
	cmd, err := Plan(opt)
	if err != nil {
		return Command{}, err
	}
	if _, err := os.Stat(opt.Input); err != nil {
		return Command{}, fmt.Errorf("convert: input: %w", err)
	}
	dir := cmd.Result
	if !cmd.Directory {
		dir = filepath.Dir(cmd.Result)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Command{}, err
	}
	if err := execFn(ctx, cmd.Name, cmd.Args...); err != nil {
		return Command{}, fmt.Errorf("convert: %s: %w", cmd.Name, err)
	}
	return cmd, nil
}

func execCommand(ctx context.Context, name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s is not on PATH; run this from nix develop on Linux", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func chooseTool(input, tool string) (string, error) {
	ext := strings.ToLower(filepath.Ext(input))
	if tool == "" {
		switch ext {
		case ".ova", ".ovf", ".vmx", ".vmdk":
			return "virt-v2v", nil
		case ".qcow2", ".raw", ".img":
			return "qemu-img", nil
		default:
			return "", fmt.Errorf("convert: pass --tool virt-v2v or qemu-img for %s", ext)
		}
	}
	if tool != "virt-v2v" && tool != "qemu-img" {
		return "", fmt.Errorf("convert: --tool must be virt-v2v or qemu-img")
	}
	if tool == "qemu-img" && (ext == ".ova" || ext == ".ovf" || ext == ".vmx") {
		return "", fmt.Errorf("convert: %s needs virt-v2v", ext)
	}
	return tool, nil
}

func virtV2VMode(input string) string {
	switch strings.ToLower(filepath.Ext(input)) {
	case ".ova", ".ovf":
		return "ova"
	case ".vmx":
		return "vmx"
	default:
		return "disk"
	}
}

func qemuFormat(input string) string {
	switch strings.ToLower(filepath.Ext(input)) {
	case ".vmdk":
		return "vmdk"
	case ".qcow2":
		return "qcow2"
	case ".raw", ".img":
		return "raw"
	default:
		return ""
	}
}
