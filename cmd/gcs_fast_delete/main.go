package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/streamingfast/tooling/cli"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

const Unlimited = math.MaxInt64

var flagVerbose = flag.Bool("v", false, "Activate debugging log output")
var flagDryRun = flag.Bool("n", false, "Dry-run the call make it only output filename instead of real delete")
var flagForce = flag.Bool("f", false, "Force running the command without asking for user intervention")

var bucket *storage.BucketHandle
var bucketURL *url.URL
var client *storage.Client
var zlog = zap.NewNop()

func main() {
	cli.SetupFlag(usage)

	if *flagVerbose {
		logging.ApplicationLogger("gcs_fast_delete", "github.com/streamingfast/tooling/cmd/gcs_fast_delete", &zlog)
	}

	args := flag.Args()
	cli.Ensure(len(args) == 1, cli.ErrorUsage(usage, "Expecting 1 argument, got %d", len(args)))

	ctx := context.Background()
	bucketRaw := args[0]

	var err error
	bucketURL, err = url.Parse(bucketRaw)
	cli.NoError(err, "GCS bucket %q is not a valid URL", bucketRaw)
	cli.Ensure(bucketURL.Scheme == "gs", "GCS bucket %q should have gs:// scheme", bucketRaw)
	cli.Ensure(bucketURL.Host != "", "GCS bucket %q should have a name", bucketRaw)

	bucketName := bucketURL.Host
	objectPrefix := strings.TrimPrefix(bucketURL.Path, "/")

	client, err = storage.NewClient(ctx)
	cli.NoError(err, "Unable to create Google Cloud Storage client")
	defer client.Close()

	objectPrefix = strings.TrimPrefix(objectPrefix, "/")

	zlog.Info("About to run fast delete operation",
		zap.String("bucket", bucketName),
		zap.String("object_prefix", objectPrefix),
	)
	bucket = client.Bucket(bucketName)

	firstFiveFiles := []string{}
	err = listFiles(ctx, objectPrefix, func(f string) {
		firstFiveFiles = append(firstFiveFiles, f)
	}, 5)

	if len(firstFiveFiles) == 0 {
		cli.End("No files from GSC bucket %q matches %q, nothing to do", bucketName, objectPrefix)
	}

	if !*flagForce {
		fileList := "- " + strings.Join(firstFiveFiles, "\n- ")
		message := fmt.Sprintf("About to delete all objects from GSC bucket %q matching %q, sample files:\n%s\n\nDo you want to continue (dry run: %t)?", bucketName, objectPrefix, fileList, *flagDryRun)

		if confirmed := cli.AskForConfirmation(message); !confirmed {
			cli.Quit("Aborting deletion")
		}
	}

	start := time.Now()
	fmt.Println()
	fmt.Println("Starting deletion ....")

	jobs := make(chan job, 1000)
	var wg sync.WaitGroup

	for w := 1; w <= 500; w++ {
		wg.Add(1)
		go worker(ctx, w, &wg, jobs)
	}

	fileCount := 0
	err = listFiles(ctx, objectPrefix, func(f string) {
		fileCount++
		jobs <- job{file: f}
	}, Unlimited)
	close(jobs)
	cli.NoError(err, "Unable to list files")

	fmt.Println("Waiting for jobs to complete ....")
	wg.Wait()

	fmt.Printf("Deleted %d files in %s\n", fileCount, time.Since(start))
}

type job struct {
	file string
}

func worker(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan job) {
	defer wg.Done()
	for j := range jobs {
		err := deleteFile(ctx, j.file)
		if err != nil {
			panic(err)
		}
	}
}

// listFiles lists objects within specified bucket.
func listFiles(ctx context.Context, prefix string, f func(file string), limit int) error {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
	defer cancel()

	zlog.Info("Listing files from bucket", zap.String("prefix", prefix))
	it := bucket.Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	count := 0
	for {
		if limit != -1 && count > limit {
			return nil
		}

		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("Bucket(%q).Objects: %v", bucketURL, err)
		}

		count++
		f(attrs.Name)
	}

	return nil
}

func deleteFile(ctx context.Context, object string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := bucket.Object(object)
	if *flagDryRun {
		fmt.Println("Would delete " + o.ObjectName())
	} else {
		if err := o.Delete(ctx); err != nil {
			return fmt.Errorf("Object(%q).Delete: %v", object, err)
		}
	}

	return nil
}

func usage() string {
	return `usage: gcs_fast_delete [-n] [-f] [-v] <bucket>

Fast delete all elements found under a Google Cloud Storage <bucket> that
that matches bucket path provided

Flags:
` + cli.FlagUsage() + `
Examples:
  # Delete all files under 'test-bucket' that matches prefix 'folder/element'
  gcs_fast_delete gs://test-bucket/folder/element

  # Delete all files under 'test-bucket' that matches prefix 'folder/element/file.txt' (usually a single files but could delete way more)
  gcs_fast_delete gs://test-bucket/folder/element/file.txt
`
}
