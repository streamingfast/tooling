package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/api/iterator"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

func main() {
	flag.Parse()

	start := time.Now()

	if len(os.Args) != 3 {
		//"dfuseio-global-blocks-us", "sol-mainnet/v1-oneblock"
		fmt.Println("Require argument [bucket] [object-prefix]")
		os.Exit(0)
	}

	bucket := os.Args[1]
	objectPrefix := os.Args[2]

	jobs := make(chan job, 1000)
	var wg sync.WaitGroup

	for w := 1; w <= 500; w++ {
		wg.Add(1)
		go worker(w, &wg, jobs)
	}

	fileCount := 0
	_, err := listFiles(bucket, objectPrefix, func(bucket string, f string) {
		fileCount++
		jobs <- job{
			bucket: bucket,
			file:   f,
		}
	})
	close(jobs)
	if err != nil {
		panic(err)
	}

	fmt.Println("Waiting ....")
	wg.Wait()
	fmt.Println("Deleted: ", fileCount, " objects in: ", time.Since(start))

}

type job struct {
	bucket string
	file   string
}

func worker(id int, wg *sync.WaitGroup, jobs <-chan job) {
	defer wg.Done()
	for j := range jobs {
		fmt.Println("worker:", id, " deleting file: ", j.file)
		err := deleteFile(j.bucket, j.file)
		if err != nil {
			panic(err)
		}
	}
}

// listFiles lists objects within specified bucket.
func listFiles(bucket string, prefix string, f func(bucket string, file string)) ([]string, error) {
	// bucket := "bucket-name"
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})
	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Bucket(%q).Objects: %v", bucket, err)
		}
		f(bucket, attrs.Name)
		files = append(files, attrs.Name)
	}
	return files, nil
}

func deleteFile(bucket, object string) error {
	// bucket := "bucket-name"
	// object := "object-name"
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	o := client.Bucket(bucket).Object(object)
	if err := o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}
	return nil
}

func createBucketClassLocation(projectID, bucketName string) error {
	// projectID := "my-project-id"
	// bucketName := "bucket-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	storageClassAndLocation := &storage.BucketAttrs{
		StorageClass: "COLDLINE",
		Location:     "asia",
	}
	bucket := client.Bucket(bucketName)
	if err := bucket.Create(ctx, projectID, storageClassAndLocation); err != nil {
		return fmt.Errorf("Bucket(%q).Create: %v", bucketName, err)
	}
	fmt.Printf("Created bucket %v in %v with storage class %v\n", bucketName, storageClassAndLocation.Location, storageClassAndLocation.StorageClass)
	return nil
}
