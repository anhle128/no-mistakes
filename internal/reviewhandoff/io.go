package reviewhandoff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func ReadBounded(path string, maxBytes int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read review file: %w", err)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("review file exceeds %d bytes", maxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read review file: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("review file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func WriteFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create review file dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp review file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp review file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp review file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename review file: %w", err)
	}
	return nil
}

func WriteRenderedFile(path string, file HandoffFile) error {
	data, err := Render(file)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data)
}

func WriteRenderedFileFromSnapshot(path string, snapshot []byte, file HandoffFile) error {
	current, err := ReadBounded(path, MaxFileBytes)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, snapshot) {
		return fmt.Errorf("review file changed while processing")
	}
	data, err := Render(file)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data)
}

func UpdateProcessedMetadata(path string, snapshot []byte, processedAt string, processedAction string) error {
	current, err := ReadBounded(path, MaxFileBytes)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, snapshot) {
		return fmt.Errorf("review file changed while processing")
	}
	meta, body, err := ParseFrontMatter(snapshot)
	if err != nil {
		return err
	}
	meta.ProcessedAt = &processedAt
	meta.ProcessedAction = processedAction
	data, err := renderWithBody(meta, body)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data)
}
