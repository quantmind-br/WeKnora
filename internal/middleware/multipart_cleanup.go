package middleware

import "github.com/gin-gonic/gin"

// MultipartFormCleanup removes, once the request finishes, the files that multipart parsing wrote to the system temp directory.
// Input: the Gin request context c; output: none. Requests that never parsed a multipart form or kept it in memory delete nothing.
// Register this middleware before all routes so it covers uploads that succeed, fail, or recover from a panic.
func MultipartFormCleanup() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// Go sets MultipartForm once ParseMultipartForm spills to disk.
			// RemoveAll deletes only the multipart temp files it created and never touches persisted business files.
			if c.Request != nil && c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		c.Next()
	}
}
