package storage

import (
	"path"
	"path/filepath"
)

func manifestPathForFile(filePath, digest string) string {
	dir := filepath.Dir(filePath)
	prefix := digestPrefix(digest)
	name := digestFileName(digest)
	return filepath.Join(dir, "manifests", "sha256", prefix, name+".json")
}

func manifestKeyForObject(key, digest string) string {
	dir := path.Dir(key)
	prefix := digestPrefix(digest)
	name := digestFileName(digest)
	if dir == "." {
		return path.Join("manifests", "sha256", prefix, name+".json")
	}
	return path.Join(dir, "manifests", "sha256", prefix, name+".json")
}

func digestPrefix(digest string) string {
	if len(digest) < 2 {
		return digest
	}
	return digest[:2]
}

func digestFileName(digest string) string {
	if len(digest) <= 2 {
		return digest
	}
	return digest[2:]
}
