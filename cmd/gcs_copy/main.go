package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/streamingfast/tooling/cli"
	"google.golang.org/api/iterator"
)

var flagVerbose = flag.Bool("v", false, "Activate debugging log output")
var flagDryRun = flag.Bool("n", false, "Dry-run the call make it only output filename instead of real delete")
var flagProject = flag.String("project", "", "Project to use for the GCS bucket")
var flagAllFilesAbove = flag.Bool("all-above", false, "Copy all files above the specified file")
var flagPersist = flag.Bool("persist", false, "Keep polling for new files above the last one copied and copy them as they appear")

func parseBucket(bucketRaw string) (name, path string, err error) {
	bucketURL, err := url.Parse(bucketRaw)
	if err != nil {
		return "", "", fmt.Errorf("GCS bucket %q is not a valid URL", bucketRaw)
	}

	if bucketURL.Scheme != "gs" {
		return "", "", fmt.Errorf("GCS bucket %q should have gs:// scheme", bucketRaw)
	}

	if bucketURL.Host == "" {
		return "", "", fmt.Errorf("GCS bucket %q should have a name", bucketRaw)
	}

	return bucketURL.Host, strings.TrimPrefix(bucketURL.Path, "/"), nil
}

func main() {
	cli.SetupFlag(usage)
	args := flag.Args()
	cli.Ensure(len(args) == 2, "Expecting 2 argument, got %d\n\n%s", len(args), usage())

	ctx := context.Background()
	srcBucketRaw := args[0]
	dstBucketRaw := args[1]

	srcBucketName, srcBucketPath, err := parseBucket(srcBucketRaw)
	cli.NoError(err, "parsing source bucket")

	dstBucketName, dstBucketPath, err := parseBucket(dstBucketRaw)
	cli.NoError(err, "parsing dest bucket")

	client, err := storage.NewClient(ctx)
	cli.NoError(err, "Unable to create Google Cloud Storage client")
	defer client.Close()

	srcBucket := client.Bucket(srcBucketName)
	if *flagProject != "" {
		srcBucket = srcBucket.UserProject(*flagProject)
	}

	dstBucket := client.Bucket(dstBucketName)
	if *flagProject != "" {
		dstBucket = dstBucket.UserProject(*flagProject)
	}

	// copy a single file
	if !strings.HasSuffix(srcBucketPath, "/") && !*flagAllFilesAbove {
		obj := srcBucket.Object(srcBucketPath)
		_, err := obj.Attrs(ctx)
		cli.NoError(err, "cannot get file %q", srcBucketPath)

		destPath := dstBucketPath
		if strings.HasSuffix(dstBucketPath, "/") {
			destPath = filepath.Join(dstBucketPath, filepath.Base(srcBucketPath))
		}

		if *flagVerbose {
			fmt.Println("copying", srcBucketPath, "to", destPath)
		}

		_, err = dstBucket.Object(destPath).CopierFrom(obj).Run(ctx)
		cli.NoError(err, "copying object")
		return
	}

	cursor := srcBucketPath

	for {
		cli.Ensure(strings.HasSuffix(dstBucketPath, "/"), "destination path should end with a / when copying a whole directory or using `all-above` flag")
		err = walkStore(ctx, srcBucket, cursor, func(fullpath, filename string) (err error) {
			dest := filepath.Join(dstBucketPath, filepath.Base(fullpath))
			if *flagVerbose {
				if srcBucketName == dstBucketName {
					fmt.Printf("copying (%s) %q to %q\n", srcBucketName, fullpath, dest)
				} else {
					fmt.Printf("copying (%s) %q to (%s) %q\n", srcBucketName, fullpath, dstBucketName, dest)
				}
			}
			if *flagDryRun {
				return nil
			}

			_, err = dstBucket.Object(dest).CopierFrom(srcBucket.Object(fullpath)).Run(ctx)
			cli.NoError(err, "copying object")

			cursor = fullpath + string([]byte{0})
			return nil
		})

		if err == nil || err == io.EOF {
			if *flagPersist {
				time.Sleep(time.Second * 5)
				continue
			}
			break
		}

		cli.NoError(err, "error during execution")
	}

}

func usage() string {
	return `usage: gcs_copy [-n] [-v] [--all-above] [--persist] gs://<bucket>/<source>/ gs://<bucket>/<destination>

Copy elements from source to destination in Google Cloud Storage.

If source is a single file, only that file is copied, unless '--all-above' is set, in which case all files above the source file are copied.
If source has a trailing slash, all the content of the folder is copied.

Flags:
` + cli.FlagUsage() + `
Examples:
  # Copy all files from 'test-bucket/v1/outputs/' to 'test-bucket/v2/outputs/'
  gcs_copy -v gs://test-bucket/v1/outputs/ gs://test-bucket/v2/outputs/

  # Copy a single file, from 'test-bucket/v1/outputs/001000.gz' to 'test-bucket/v2/outputs/'
  gcs_copy -v gs://test-bucket/v1/outputs/001000.gz gs://test-bucket/v2/outputs/

  # Copy all files greater than 'test-bucket/v1/outputs/001000.gz' to 'test-bucket/v2/outputs/', then keep watching for new files above the last one and copy them too
  gcs_copy -v --all-above --persist gs://test-bucket/v1/outputs/001000.gz gs://test-bucket/v2/outputs/
`
}

// walkStore will walk under startingPoint if it ends with `/` or it will walk siblings of 'startingPoint' (starting from it), and call 'f' for each file found.
func walkStore(ctx context.Context, bucket *storage.BucketHandle, startingPoint string, f func(fullpath, filename string) (err error)) error {

	var prefix string
	var startOffset string
	if strings.HasSuffix(startingPoint, "/") {
		prefix = startingPoint
	} else {
		prefix = filepath.Dir(startingPoint) + "/"
		startOffset = startingPoint
	}

	q := &storage.Query{
		Prefix:      prefix,
		StartOffset: startOffset,
	}
	q.SetAttrSelection([]string{"Name"}) // only fetch the name, 25% faster
	it := bucket.Objects(ctx, q)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if err := f(attrs.Name, filepath.Base(attrs.Name)); err != nil {
			return err
		}
	}
	return nil
}
