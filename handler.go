package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	maxSingleFileSize = 5 << 30  // 5GB
	maxTotalSize      = 10 << 30 // 10GB
)

func listfiles(c *gin.Context) {
	// 确保 uploads 目录存在
	if err := ensureUploadsDirExists(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get working directory"})
		return
	}

	// 构建 uploads 目录的路径
	uploadsDir := filepath.Join(wd, "uploads")
	// 列出 uploads 目录下的文件
	files, err := filepath.Glob(filepath.Join(uploadsDir, "*"))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to list files in uploads directory"})
		return
	}

	// 构建文件列表
	fileList := make([]string, 0, len(files))
	for _, file := range files {
		fileList = append(fileList, filepath.Base(file))
	}

	c.JSON(200, gin.H{"files": fileList})
}

func uploadfile(c *gin.Context) {
	// 确保 uploads 目录存在
	if err := ensureUploadsDirExists(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// 检查上传文件大小
	if err := c.Request.ParseMultipartForm(maxSingleFileSize); err != nil {
		c.JSON(400, gin.H{"error": "File size exceeds the limit of 1GB"})
		return
	}

	// 上传文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "No file uploaded"})
		return
	}

	// 检查单个文件大小
	if file.Size > maxSingleFileSize {
		c.JSON(400, gin.H{"error": "File size exceeds the limit of 1GB"})
		return
	}

	// 检查 uploads 文件夹总大小
	totalSize, err := getTotalSizeOfUploads()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get total size of uploads directory"})
		return
	}

	if totalSize+file.Size > maxTotalSize {
		c.JSON(400, gin.H{"error": "Total size of uploads directory exceeds the limit of 5GB"})
		return
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get working directory"})
		return
	}

	// 去除空格和特殊字符
	filename := filepath.Base(file.Filename)
	filename = filepath.Clean(filename)

	// 构建保存文件的路径，将文件保存到 uploads 目录下
	savePath := filepath.Join(wd, "uploads", filename)
	// 保存文件
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save file"})
			return
		}
	}()

	c.JSON(200, gin.H{"message": "File uploaded successfully", "filename": filename})
	wg.Wait()
}

func downloadfile(c *gin.Context) {
	// 确保 uploads 目录存在
	if err := ensureUploadsDirExists(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// 下载文件
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(400, gin.H{"error": "No filename specified"})
		return
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get working directory"})
		return
	}

	// 构建文件的路径，从 uploads 目录读取文件
	filePath := filepath.Join(wd, "uploads", filename)
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get file information"})
		return
	}

	// 设置响应头，强制浏览器下载文件
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件给客户端
	c.File(filePath)
}

func closefile(c *gin.Context) {
	// 确保 uploads 目录存在
	if err := ensureUploadsDirExists(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get working directory"})
		return
	}

	// 构建 uploads 目录的路径
	uploadsDir := filepath.Join(wd, "uploads")
	// 清空 uploads 目录
	err = os.RemoveAll(uploadsDir)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to remove uploads directory"})
		return
	}

	c.JSON(200, gin.H{"message": "Shared file system closed"})
}

// 确保 uploads 目录存在
func ensureUploadsDirExists() error {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	// 构建 uploads 目录的路径
	uploadsDir := filepath.Join(wd, "uploads")
	// 创建 uploads 目录，如果不存在
	return os.MkdirAll(uploadsDir, 0755)
}

// 获取 uploads 文件夹总大小
func getTotalSizeOfUploads() (int64, error) {
	wd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	uploadsDir := filepath.Join(wd, "uploads")

	var totalSize int64
	err = filepath.Walk(uploadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}
