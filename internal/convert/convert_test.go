package convert

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPlanVirtV2V(t *testing.T) {
	cmd, err := Plan(Options{Input: "guest.ova", Output: "out"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-i", "ova", "guest.ova", "-o", "disk", "-os", "out", "-of", "qcow2"}
	if cmd.Name != "virt-v2v" || !slices.Equal(cmd.Args, want) || !cmd.Directory {
		t.Fatalf("%+v", cmd)
	}
	if _, err := Plan(Options{Input: "guest.ova", Tool: "qemu-img"}); err == nil {
		t.Fatal("ova was accepted by qemu-img")
	}
}

func TestPlanQEMUImg(t *testing.T) {
	cmd, err := Plan(Options{Input: "disk.qcow2", Format: "raw"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"convert", "-f", "qcow2", "-O", "raw", "disk.qcow2", filepath.Join("converted", "disk.raw")}
	if cmd.Name != "qemu-img" || !slices.Equal(cmd.Args, want) || cmd.Directory {
		t.Fatalf("%+v", cmd)
	}
	if _, err := Plan(Options{Input: "disk.qcow2", Output: "disk.qcow2"}); err == nil {
		t.Fatal("output overwriting input was accepted")
	}
}

func TestRunRecordsCommand(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "disk.qcow2")
	if err := os.WriteFile(input, []byte("not a disk"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "disk.raw")
	var got []string
	cmd, err := run(context.Background(), Options{Input: input, Output: output, Format: "raw"}, func(_ context.Context, name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Result != output || got[0] != "qemu-img" || !slices.Contains(got, input) || !slices.Contains(got, output) {
		t.Fatalf("cmd %+v args %v", cmd, got)
	}
}
