package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

// FileUploadTask represents a task for uploading a file
type FileUploadTask struct {
	Index      int
	File       *multipart.FileHeader
	BucketName string
	ImagePrefix string
}

// FileUploadResult represents the result of a file upload task
type FileUploadResult struct {
	Index   int
	Success bool
	Error   error
	FileKey string
}

// uploadFilesWithWorkerPool uses a worker pool to upload files concurrently
func UploadFilesWithWorkerPool(c *gin.Context, bucketName, imagePrefix string, files []*multipart.FileHeader) bool {
	numWorkers := 3 // Configure the number of workers based on your needs
	if len(files) < numWorkers {
		numWorkers = len(files)
	}
	
	// Create channels for tasks and results
	tasks := make(chan FileUploadTask, len(files))
	results := make(chan FileUploadResult, len(files))
	
	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go fileUploadWorker(tasks, results, &wg)
	}
	
	// Close the results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// Queue tasks for workers
	for i, file := range files {
		tasks <- FileUploadTask{
			Index:       i,
			File:        file,
			BucketName:  bucketName,
			ImagePrefix: imagePrefix,
		}
	}
	close(tasks) // No more tasks will be added
	
	// Process results and check for failures
	uploadedFiles := make(map[int]string)
	for result := range results {
		if result.Error != nil || !result.Success {
			// If any upload fails, cleanup files already uploaded and return failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file %d: %v", result.Index, result.Error)})
			
			// Clean up all files that were successfully uploaded
			for idx, fileKey := range uploadedFiles {
				DeleteFileFromS3(bucketName, fileKey)
				delete(uploadedFiles, idx) // Remove from map after cleanup
			}
			return false
		}
		uploadedFiles[result.Index] = result.FileKey
	}
	
	return true
}

// fileUploadWorker processes file upload tasks
func fileUploadWorker(tasks <-chan FileUploadTask, results chan<- FileUploadResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for task := range tasks {
		// Process the file upload
		fileKey, err := uploadFileToS3(task)
		
		result := FileUploadResult{
			Index:   task.Index,
			Success: err == nil,
			Error:   err,
			FileKey: fileKey,
		}
		
		results <- result
	}
}

// uploadFileToS3 handles the actual upload to S3
func uploadFileToS3(task FileUploadTask) (string, error) {
	// Open the uploaded file
	src, err := task.File.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()
	
	// Read the file content
	fileBytes := bytes.Buffer{}
	if _, err := io.Copy(&fileBytes, src); err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}
	
	// Create a unique key for this file
	fileKey := fmt.Sprintf("%s/image_%d%s", task.ImagePrefix, task.Index, filepath.Ext(task.File.Filename))
	
	// Upload to S3
	if err := UploadFileToS3(task.BucketName, fileKey, fileBytes.Bytes()); err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}
	
	return fileKey, nil
}

// cleanupS3Files removes all uploaded files in case of failure
func cleanupS3Files(bucketName, imagePrefix string, count int) {
	// Create a worker pool for deletion
	numWorkers := 3
	if count < numWorkers {
		numWorkers = count
	}
	
	// Create channels for deletion tasks
	tasks := make(chan string, count)
	var wg sync.WaitGroup
	
	// Start worker pool for deletion
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileKey := range tasks {
				DeleteFileFromS3(bucketName, fileKey)
			}
		}()
	}
	
	// Queue deletion tasks
	for i := 0; i < count; i++ {
		fileKey := fmt.Sprintf("%s/image_%d", imagePrefix, i)
		tasks <- fileKey
	}
	
	close(tasks)
	wg.Wait() // Wait for all deletions to complete
}
