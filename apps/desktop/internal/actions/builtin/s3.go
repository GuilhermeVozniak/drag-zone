package builtin

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

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

	entries, total, err := collectUploadEntries(inv.Payload.Paths)
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
	uploader := manager.NewUploader(s3.NewFromConfig(cfg))

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

	resultURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, firstKey)
	if prefix := inv.Target.Option("url_prefix", ""); prefix != "" {
		resultURL = strings.TrimRight(prefix, "/") + "/" + firstKey
	}
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
