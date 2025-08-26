package main

import (
	"bufio"
	"context"
	"fmt"
	"iter"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
)

const Unlimited = math.MaxInt64

var client *storage.Client

var zlog, _ = logging.ApplicationLogger("gcs_fast_delete", "github.com/streamingfast/tooling/cmd/gcs_fast_delete",
	logging.WithConsoleToStderr(),
	logging.WithDefaultSpec(".*=error"),
)

func main() {
	Run(
		"gcs_fast_delete <flags> <bucket_or_filename>",
		"Fast delete Google Cloud Storage objects",
		Description(`
			Fast delete Google Cloud Storage objects. Can process either a bucket/prefix
			or a list of individual files.

			The argument can be either:
			- A direct GCS bucket/prefix URL (starts with gs://) - deletes all files matching the prefix
			- A filename containing individual GCS file URLs (one per line, each starting with gs://)
		`),
		ExactArgs(1),
		Flags(func(flags *pflag.FlagSet) {
			flags.BoolP("dry-run", "n", false, "Dry-run the call make it only output filename instead of real delete")
			flags.BoolP("force", "f", false, "Force running the command without asking for user intervention")
			flags.StringP("project", "p", "", "Project to use for the GCS bucket")
		}),
		Example(`
			# Delete all files under 'test-bucket' that matches prefix 'folder/element'
			gcs_fast_delete gs://test-bucket/folder/element

			# Delete a specific file (or all files matching this exact prefix)
			gcs_fast_delete gs://test-bucket/folder/element/file.txt

			# Delete specific files listed in a file (each line should be a complete gs:// file URL)
			gcs_fast_delete files_to_delete.txt

			# Dry run to see what would be deleted
			gcs_fast_delete --dry-run gs://test-bucket/folder/element
		`),
		Execute(executeGCSFastDelete),
	)
}

func executeGCSFastDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	argInput := args[0]

	var err error
	client, err = storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to create Google Cloud Storage client: %w", err)
	}
	defer client.Close()

	if looksLikeFilename(argInput) {
		zlog.Info("Input looks like a filename, reading file URLs from file", zap.String("filename", argInput))
		err = processFileList(ctx, cmd, argInput)
		if err != nil {
			return fmt.Errorf("failed to process file list: %w", err)
		}
	} else {
		err = processBucket(ctx, cmd, argInput)
		if err != nil {
			return fmt.Errorf("failed to process bucket %q: %w", argInput, err)
		}
	}

	return nil
}

type FileSequence struct {
	Tag  string
	List func(ctx context.Context, count int) iter.Seq2[string, error]
}

func processStream(ctx context.Context, cmd *cobra.Command, bucketName string, fileSequence FileSequence) error {
	zlog.Info("About to run fast delete operation",
		zap.String("bucket", bucketName),
		zap.String("stream", fileSequence.Tag),
	)

	firstFiveFiles := []string{}
	for fileName, err := range fileSequence.List(ctx, 5) {
		if err != nil {
			return fmt.Errorf("error reading file from stream: %w", err)
		}

		firstFiveFiles = append(firstFiveFiles, fileName)
		if len(firstFiveFiles) >= 5 {
			break
		}
	}

	if len(firstFiveFiles) == 0 {
		fmt.Printf("No files from %q file sequence %s, nothing to do\n", bucketName, fileSequence.Tag)
		return nil
	}

	force := sflags.MustGetBool(cmd, "force")
	dryRun := sflags.MustGetBool(cmd, "dry-run")

	if !force {
		fileList := "- " + strings.Join(firstFiveFiles, "\n- ")
		message := fmt.Sprintf("About to delete all objects from GCS bucket %q matching %q, sample files:\n%s\n\nDo you want to continue (dry run: %t)?", bucketName, fileSequence.Tag, fileList, dryRun)

		if confirmed, _ := cli.PromptConfirm(message); !confirmed {
			return fmt.Errorf("user aborted deletion for bucket %q", bucketName)
		}
	}

	bucket := createBucketHandle(cmd, bucketName)

	start := time.Now()
	fmt.Println()
	fmt.Printf("Starting deletion for %s (%s)....\n", bucket.BucketName(), fileSequence.Tag)

	jobs := make(chan job, 1000)
	var wg sync.WaitGroup

	for w := 1; w <= 250; w++ {
		wg.Add(1)
		go worker(ctx, cmd, w, &wg, jobs)
	}

	ctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
	defer cancel()

	fileCount := 0
	for fileName, err := range fileSequence.List(ctx, Unlimited) {
		if err != nil {
			return fmt.Errorf("error reading file from stream: %w", err)
		}

		fileCount++
		jobs <- job{file: fileName, bucket: bucket}
		if fileCount%1000 == 0 {
			zlog.Info("progress", zap.Int("processed", fileCount), zap.String("bucket", bucketName))
		}
	}
	close(jobs)

	fmt.Println("Waiting for jobs to complete ....")
	wg.Wait()

	fmt.Printf("Deleted %d files from %s (%s) in %s\n", fileCount, bucket.BucketName(), fileSequence.Tag, time.Since(start))
	return nil
}

func processBucket(ctx context.Context, cmd *cobra.Command, bucketRaw string) error {
	var err error
	bucketURL, err := url.Parse(bucketRaw)
	if err != nil {
		return fmt.Errorf("GCS bucket %q is not a valid URL: %w", bucketRaw, err)
	}

	if bucketURL.Scheme != "gs" {
		return fmt.Errorf("GCS bucket %q should have gs:// scheme", bucketRaw)
	}

	if bucketURL.Host == "" {
		return fmt.Errorf("GCS bucket %q should have a name", bucketRaw)
	}

	bucketName := bucketURL.Host
	objectPrefix := strings.TrimPrefix(bucketURL.Path, "/")
	objectPrefix = strings.TrimPrefix(objectPrefix, "/")

	return processStream(ctx, cmd, bucketName, FileSequence{
		Tag: fmt.Sprintf("gcs matching %q", objectPrefix),
		List: func(ctx context.Context, count int) iter.Seq2[string, error] {
			return listFiles(ctx, cmd, bucketName, objectPrefix, count)
		},
	})
}

func processFileList(ctx context.Context, cmd *cobra.Command, filename string) error {
	fileURLs, err := readFileURLsFromFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read file URLs from file %q: %w", filename, err)
	}

	// Group files by bucket for efficient processing
	bucketFiles := make(map[string][]string)

	for _, fileURL := range fileURLs {
		parsedURL, err := url.Parse(fileURL)
		if err != nil {
			return fmt.Errorf("invalid GCS file URL %q: %w", fileURL, err)
		}

		if parsedURL.Scheme != "gs" {
			return fmt.Errorf("file URL %q should have gs:// scheme", fileURL)
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("file URL %q should have a bucket name", fileURL)
		}

		bucketName := parsedURL.Host
		objectPath := strings.TrimPrefix(parsedURL.Path, "/")

		if objectPath == "" {
			return fmt.Errorf("file URL %q should specify a file path", fileURL)
		}

		bucketFiles[bucketName] = append(bucketFiles[bucketName], objectPath)
	}

	for bucketName, files := range bucketFiles {
		err := processStream(ctx, cmd, bucketName, FileSequence{
			Tag: fmt.Sprintf("file list from %q", filename),
			List: func(ctx context.Context, count int) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					for i := 0; i < min(count, len(files)); i++ {
						if !yield(files[i], ctx.Err()) {
							break
						}
					}
				}
			},
		})
		if err != nil {
			return fmt.Errorf("unable to process file list for bucket %q: %w", bucketName, err)
		}
	}

	return nil
}

type job struct {
	file   string
	bucket *storage.BucketHandle
}

func worker(ctx context.Context, cmd *cobra.Command, _ int, wg *sync.WaitGroup, jobs <-chan job) {
	defer wg.Done()
	for j := range jobs {
		err := deleteFile(ctx, cmd, j.bucket, j.file)
		if err != nil {
			zlog.Info("retrying file", zap.String("file", j.file))
			err = deleteFile(ctx, cmd, j.bucket, j.file)
			if err != nil {
				zlog.Info("skipping failed file", zap.String("file", j.file))
			}
		}
	}
}

// listFiles lists objects within specified bucket.
func listFiles(ctx context.Context, cmd *cobra.Command, bucketName string, prefix string, limit int) iter.Seq2[string, error] {
	zlog.Info("Listing files from bucket", zap.String("prefix", prefix))
	it := createBucketHandle(cmd, bucketName).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	return func(yield func(string, error) bool) {
		count := 0
		for {
			if limit != -1 && count > limit {
				return
			}

			name := ""
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}

			if err == nil {
				name = attrs.Name
			} else {
				err = fmt.Errorf("getting next object from bucket %q: %w", bucketName, err)
			}

			if !yield(name, err) {
				break
			}

			count++
		}
	}
}

func deleteFile(ctx context.Context, cmd *cobra.Command, bucket *storage.BucketHandle, object string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	o := bucket.Object(object)
	dryRun := sflags.MustGetBool(cmd, "dry-run")

	if dryRun {
		fmt.Println("Would delete " + o.ObjectName())
	} else {
		if err := o.Delete(ctx); err != nil {
			return fmt.Errorf("Object(%q).Delete: %v", object, err)
		}
	}

	return nil
}

func looksLikeFilename(arg string) bool {
	// Check if it's a valid file path that exists
	_, err := os.Stat(arg)
	return err == nil
}

func readFileURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %q: %w", filename, err)
	}
	defer file.Close()

	var fileURLs []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "gs://") {
			return nil, fmt.Errorf("line %d in file %q does not start with gs://: %q", lineNum, filename, line)
		}

		fileURLs = append(fileURLs, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %q: %w", filename, err)
	}

	if len(fileURLs) == 0 {
		return nil, fmt.Errorf("no valid gs:// file URLs found in file %q", filename)
	}

	return fileURLs, nil
}

func createBucketHandle(cmd *cobra.Command, bucketName string) *storage.BucketHandle {
	project := sflags.MustGetString(cmd, "project")

	if project == "" {
		return client.Bucket(bucketName)
	}

	zlog.Debug("Using project", zap.String("project", project))
	return client.Bucket(bucketName).UserProject(project)
}
