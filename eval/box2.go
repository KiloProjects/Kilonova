package eval

import (
	"io/fs"
)

type BucketFile struct {
	Bucket   Bucket
	Filename string
	Mode     fs.FileMode
}

type ByteFile struct {
	Data []byte
	Mode fs.FileMode
}

type Box2Request struct {
	// key - path, value - reference to bucket file
	InputBucketFiles map[string]*BucketFile
	// key - path, value - contents
	InputByteFiles map[string]*ByteFile

	// Command to run
	Command []string
	// Box run config
	RunConfig *RunConfig

	// File paths to return
	OutputByteFiles []string
	// key - path, value - file to save into (will have mode set to whatever is in the struct)
	OutputBucketFiles map[string]*BucketFile
}

type Box2Response struct {
	Stats *RunStats

	// Files specified in the Request and not present in the response were not found (did not exist)
	// key - path, value - contents
	ByteFiles map[string][]byte
	// key - path, value - reference to bucket file
	BucketFiles map[string]*BucketFile
}
