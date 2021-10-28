package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/streamingfast/tooling/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/api/googleapi"
)

const Unlimited = math.MaxInt64

var flagVerbose = flag.Bool("v", false, "Activate debugging log output")
var flagOverwrite = flag.Bool("o", false, "Allow overwriting file on gs destination")

//var bucket *storage.BucketHandle
var zlog = zap.NewNop()

func main() {
	cli.SetupFlag(usage)


	if *flagVerbose {
		logging.ApplicationLogger("gcs_upload", "github.com/streamingfast/tooling/cmd/gcs_upload", &zlog)
	}

	args := flag.Args()
	cli.Ensure(len(args) == 2, cli.ErrorUsage(usage, "Expecting 2 argument, got %d", len(args)))

	ctx := context.Background()
	src := args[0]
	dest := args[1]

	bucketURL, err := url.Parse(dest)
	cli.NoError(err, "GCS bucket %q is not a valid URL", dest)
	cli.Ensure(bucketURL.Scheme == "gs", "GCS bucket %q should have gs:// scheme", dest)
	cli.Ensure(bucketURL.Host != "", "GCS bucket %q should have a name", dest)

	bucketName := bucketURL.Host
	destPath := strings.TrimPrefix(bucketURL.Path, "/")

	client, err := storage.NewClient(ctx)
	cli.NoError(err, "Unable to create Google Cloud Storage client")
	defer client.Close()

	zlog.Info("About to upload operation",
		zap.String("source filename", src),
		zap.String("dest_bucket", bucketName),
		zap.String("dest_path", destPath),
	)
	bucket := client.Bucket(bucketName)

	err = pushLocalFile(ctx, src, bucket, destPath, *flagOverwrite)
	cli.NoError(err, "pushing local file")
	zlog.Info("Success")
}

func pushLocalFile(ctx context.Context, localFile string, bucket *storage.BucketHandle, destPath string, overwrite bool) error {
	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("opening local file %q: %s", localFile, err)
	}
	defer f.Close()

	object := bucket.Object(destPath)
	if !overwrite {
		object = object.If(storage.Conditions{DoesNotExist: true})
	}
	w := object.NewWriter(ctx)
	w.ContentType = "application/octet-stream"
	w.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(w, f); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		if e, ok := err.(*googleapi.Error); ok {
			if !overwrite && e.Code == http.StatusPreconditionFailed {
				return fmt.Errorf("use '-o' to allow overwriting remote file")
			}
		}
		return err
	}
	return nil
}

func usage() string {
	return `usage: gcs_upload [-o] [-v] {src-file} {gs-dest-url}

Uploads a local file to Google Storage bucket. Make sure that you write full destination URL with file

Flags:
` + cli.FlagUsage() + `
Example:
  gcs_upload /mnt/bigfile gs://test-bucket/somewhere/bigfile.bin
`
}
