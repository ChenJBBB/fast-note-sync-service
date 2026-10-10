package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yeka/zip"
)

var sevenZipExecutableNames = []string{"7zz", "7z", "7za"}

// Zip compresses files or directories into a zip file
// source: path to file or directory
// target: path to output zip file
func Zip(source, target string) error {
	return ZipWithPassword(source, target, "")
}

// ZipWithPassword compresses files or directories into a zip file with password
func ZipWithPassword(source, target, password string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	if _, err := os.Stat(source); err != nil {
		return err
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source
		// 获取相对于 source 的路径
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// Skip root directory itself
		// 跳过根目录本身
		if relPath == "." {
			return nil
		}

		// Use / as separator in zip file (Standard requirements)
		// ZIP 文件内必须使用 / 作为路径分隔符
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Use relative path as filename in zip
		// 使用相对路径作为压缩包内的文件名
		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		var writer io.Writer
		if password != "" && !info.IsDir() {
			// Set password and encryption method
			// 设置密码和加密方法
			header.SetPassword(password)
			// StandardEncryption is more compatible with Windows built-in ZIP tool
			// StandardEncryption 对 Windows 自带 ZIP 工具兼容性更好
			header.SetEncryptionMethod(zip.StandardEncryption)
			writer, err = archive.CreateHeader(header)
		} else {
			writer, err = archive.CreateHeader(header)
		}

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// SevenZipWithPassword creates a password-protected 7z archive and encrypts
// both file contents and archive headers (including file and directory names).
// It requires 7zz, 7z, or 7za to be installed, unless SEVENZIP_BINARY points
// to a compatible executable.
func SevenZipWithPassword(source, target, password string) error {
	return SevenZipWithPasswordContext(context.Background(), source, target, password)
}

// SevenZipWithPasswordContext is SevenZipWithPassword with cancellation support.
func SevenZipWithPasswordContext(ctx context.Context, source, target, password string) error {
	if password == "" {
		return errors.New("7z archive password must not be empty")
	}

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	executable, err := findSevenZipExecutable()
	if err != nil {
		return err
	}

	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}

	workingDir := filepath.Dir(source)
	inputPath := filepath.Base(source)
	if info.IsDir() {
		workingDir = source
		inputPath = "."
	}

	// 7-Zip updates an existing archive by default. Remove the target first so
	// a retry cannot retain entries from a previous, partially written archive.
	if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	args := sevenZipArchiveArgs(target, inputPath, password)
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = workingDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		_ = os.Remove(target)
		message := strings.TrimSpace(strings.ReplaceAll(output.String(), password, "[REDACTED]"))
		if message == "" {
			return fmt.Errorf("7z archive creation failed: %w", err)
		}
		return fmt.Errorf("7z archive creation failed: %w: %s", err, message)
	}

	return nil
}

func sevenZipArchiveArgs(target, inputPath, password string) []string {
	return []string{
		"a",
		"-t7z",
		"-mhe=on",
		"-p" + password,
		"-y",
		"-bd",
		"--",
		target,
		inputPath,
	}
}

func findSevenZipExecutable() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("SEVENZIP_BINARY")); configured != "" {
		path, err := exec.LookPath(configured)
		if err != nil {
			return "", fmt.Errorf("configured SEVENZIP_BINARY %q was not found: %w", configured, err)
		}
		return path, nil
	}

	if executable, err := os.Executable(); err == nil {
		if path, ok := findSevenZipBesideExecutable(executable); ok {
			return path, nil
		}
	}

	for _, name := range sevenZipExecutableNames {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", errors.New("7-Zip executable not found; install 7zz, 7z, or 7za, or set SEVENZIP_BINARY")
}

func findSevenZipBesideExecutable(executable string) (string, bool) {
	dir := filepath.Dir(executable)
	for _, name := range sevenZipExecutableNames {
		candidates := []string{filepath.Join(dir, name)}
		if runtime.GOOS == "windows" {
			candidates = append(candidates, filepath.Join(dir, name+".exe"))
		}
		for _, candidate := range candidates {
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}
	return "", false
}

// ZipBytes creates a zip archive from a map of filenames and their contents (bytes)
func ZipBytes(files map[string][]byte, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	for name, content := range files {
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		_, err = writer.Write(content)
		if err != nil {
			return err
		}
	}

	return nil
}
