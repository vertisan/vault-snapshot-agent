package storage

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/charmbracelet/log"
	"google.golang.org/api/iterator"
)

type GCSStorageDriver struct {
	Bucket string
	Prefix string
	client *storage.Client
	ctx    context.Context
}

func NewGCSStorageDriver(bucket, prefix string) (*GCSStorageDriver, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Error("Cannot create GCS client!", "err", err.Error())
		return nil, err
	}

	return &GCSStorageDriver{
		Bucket: bucket,
		Prefix: prefix,
		client: client,
		ctx:    ctx,
	}, nil
}

func (g *GCSStorageDriver) Name() string {
	return "gcs"
}

func (g *GCSStorageDriver) Write(fileName string, data []byte) (string, error) {
	objectName := fileName
	if g.Prefix != "" {
		objectName = fmt.Sprintf("%s/%s", strings.TrimSuffix(g.Prefix, "/"), fileName)
	}

	obj := g.client.Bucket(g.Bucket).Object(objectName)
	writer := obj.NewWriter(g.ctx)

	if _, err := writer.Write(data); err != nil {
		if err := writer.Close(); err != nil {
			log.Error("Cannot close GCS writer!", "err", err.Error())
			return "", err
		}

		log.Error("Cannot write to GCS!", "err", err.Error())
		return "", err
	}

	if err := writer.Close(); err != nil {
		log.Error("Cannot close GCS writer!", "err", err.Error())
		return "", err
	}

	fullPath := fmt.Sprintf("gs://%s/%s", g.Bucket, objectName)
	return fullPath, nil
}

func (g *GCSStorageDriver) Remove(fileName string) error {
	objectName := fileName
	if g.Prefix != "" {
		objectName = fmt.Sprintf("%s/%s", strings.TrimSuffix(g.Prefix, "/"), fileName)
	}

	obj := g.client.Bucket(g.Bucket).Object(objectName)
	if err := obj.Delete(g.ctx); err != nil {
		log.Error("Cannot delete object from GCS!", "err", err.Error())
		return err
	}

	return nil
}

func (g *GCSStorageDriver) List() ([]FileInfo, error) {
	prefix := g.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	query := &storage.Query{Prefix: prefix}
	it := g.client.Bucket(g.Bucket).Objects(g.ctx, query)

	files := make([]FileInfo, 0)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Error("Cannot list objects from GCS!", "err", err.Error())
			return nil, err
		}

		// Extract just the filename from the full path
		name := attrs.Name
		if g.Prefix != "" {
			name = strings.TrimPrefix(name, strings.TrimSuffix(g.Prefix, "/")+"/")
		}

		// Filter for snapshot files
		if strings.HasPrefix(name, "vault-snapshot-") && strings.HasSuffix(name, ".snap") {
			files = append(files, FileInfo{
				Name:    name,
				ModTime: attrs.Updated,
				Size:    attrs.Size,
			})
		}
	}

	return files, nil
}

func (g *GCSStorageDriver) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}
