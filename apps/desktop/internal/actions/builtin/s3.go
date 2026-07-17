package builtin

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// s3EndpointOverride, when non-empty, replaces the region-derived S3 host
// with a custom base endpoint (path-style addressing). It exists solely so
// tests can point the client at an httptest.Server; production code never
// sets it, so the default (empty) behavior — the real region endpoint via
// virtual-hosted-style addressing — is unchanged.
var s3EndpointOverride string

// S3Upload uploads dropped files and folders to an Amazon S3 bucket and
// copies the public URL of the first file to the clipboard.
type S3Upload struct{}

func (S3Upload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "s3-upload",
		Name:        "Amazon S3 Upload",
		Description: "Upload dropped files to an Amazon S3 bucket and copy the URL.",
		Icon:        "upload",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "access_key", Label: "Access key", Type: "text", Required: true},
			{Key: "secret_key", Label: "Secret key", Type: "password", Required: true},
			{Key: "region", Label: "Region", Type: "text", Required: true, Placeholder: "us-east-1"},
			{Key: "bucket", Label: "Bucket", Type: "text", Required: true},
			{Key: "key_prefix", Label: "Key prefix", Type: "text", Placeholder: "uploads/"},
			{Key: "url_prefix", Label: "Public URL prefix (optional)", Type: "text"},
		},
	}
}

func (S3Upload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	accessKey := inv.Target.Option("access_key", "")
	secretKey := inv.Target.Option("secret_key", "")
	region := inv.Target.Option("region", "")
	bucket := inv.Target.Option("bucket", "")
	if accessKey == "" || secretKey == "" || region == "" || bucket == "" {
		return actions.Result{}, fmt.Errorf("access key, secret key, region and bucket must be configured")
	}
	if len(inv.Payload.Paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to upload")
	}

	// Holding Option zips the drop into a single archive before upload, like
	// Dropzone's S3 and FTP actions.
	paths := inv.Payload.Paths
	if inv.Payload.HasModifier("Option") {
		zipPath, zipDir, err := zipForUpload(paths)
		if err != nil {
			return actions.Result{}, fmt.Errorf("zipping before upload: %w", err)
		}
		defer os.RemoveAll(zipDir)
		paths = []string{zipPath}
	}

	entries, total, err := collectUploadEntries(paths)
	if err != nil {
		return actions.Result{}, err
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return actions.Result{}, fmt.Errorf("configuring AWS client: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if s3EndpointOverride != "" {
			o.BaseEndpoint = aws.String(s3EndpointOverride)
			o.UsePathStyle = true
		}
	})
	uploader := manager.NewUploader(client)

	keyPrefix := inv.Target.Option("key_prefix", "")
	var done int64
	var firstKey string
	for _, e := range entries {
		key := keyPrefix + e.rel
		if firstKey == "" {
			firstKey = key
		}
		inv.Progress.Detail(path.Base(e.rel))
		if err := s3UploadOne(ctx, uploader, bucket, key, e, total, &done, inv.Progress); err != nil {
			return actions.Result{}, fmt.Errorf("uploading %s: %w", e.rel, err)
		}
	}

	resultURL := s3PublicURL(bucket, region, inv.Target.Option("url_prefix", ""), firstKey)
	if err := inv.Services.CopyToClipboard(resultURL); err != nil {
		return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
	}
	return actions.Result{
		Message: fmt.Sprintf("Uploaded %d item(s) to %s", len(inv.Payload.Paths), bucket),
		URL:     resultURL,
	}, nil
}

// s3UploadOne streams one local file to the bucket, updating progress.
func s3UploadOne(ctx context.Context, uploader *manager.Uploader, bucket, key string, e uploadEntry, total int64, done *int64, progress actions.Progress) error {
	f, err := os.Open(e.local)
	if err != nil {
		return err
	}
	defer f.Close()
	body := &progressReader{r: f, onBytes: func(n int64) {
		if total > 0 {
			*done += n
			progress.Percent(int(*done * 100 / total))
		}
	}}
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

// zipForUpload archives paths into a single .zip file inside a fresh
// temporary directory (so the resulting file name stays clean, e.g.
// "photos.zip" rather than a randomized name) and returns its path plus the
// directory to clean up afterward.
func zipForUpload(paths []string) (zipPath, tmpDir string, err error) {
	tmpDir, err = os.MkdirTemp("", "dragzone-s3-zip-*")
	if err != nil {
		return "", "", err
	}
	name := strings.TrimSuffix(filepath.Base(paths[0]), filepath.Ext(paths[0]))
	if len(paths) > 1 {
		name = "Archive"
	}
	zipPath = filepath.Join(tmpDir, name+".zip")
	if err := writeZip(zipPath, paths, func(int64) {}); err != nil {
		os.RemoveAll(tmpDir)
		return "", "", err
	}
	return zipPath, tmpDir, nil
}

// s3PublicURL builds the URL copied to the clipboard: the custom prefix when
// set, otherwise the default virtual-hosted-style S3 URL.
func s3PublicURL(bucket, region, urlPrefix, key string) string {
	if urlPrefix != "" {
		return strings.TrimRight(urlPrefix, "/") + "/" + key
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, key)
}
