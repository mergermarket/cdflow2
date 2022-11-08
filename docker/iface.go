package docker

import (
	"io"
	"time"
)

// Iface is an interface for interracting with docker.
type Iface interface {
	Run(options *RunOptions) error
	EnsureImage(image string, outputStream io.Writer) error
	PullImage(image string, outputStream io.Writer) error
	GetImageRepoDigests(image string) ([]string, error)
	Exec(options *ExecOptions) error
	Stop(id string, timeout time.Duration) error
	CreateVolume(name string) (string, error)
	VolumeExists(name string) (bool, error)
	RemoveVolume(id string) error
	CreateContainer(options *CreateContainerOptions) (string, error)
	RemoveContainer(id string) error
	CopyFromContainer(id, path string) (io.ReadCloser, error)
	CopyToContainer(id, path string, reader io.Reader) error
	SetDebugVolume(volume string)
}

// RunOptions represents the options to the Run method.
type RunOptions struct {
	Image         string
	WorkingDir    string
	Entrypoint    []string
	Cmd           []string
	Env           []string
	Binds         []string
	NamePrefix    string
	InputStream   io.Reader
	OutputStream  io.Writer
	ErrorStream   io.Writer
	Started       chan string
	Init          bool
	SuccessStatus int
	BeforeRemove  func(id string) error
}

// CreateContainerOptions represents the options to the CreateContainer method.
type CreateContainerOptions struct {
	Image string
	Binds []string
}

// ExecOptions represents options to the Exec method.
type ExecOptions struct {
	ID           string
	Cmd          []string
	Env          map[string]string
	InputStream  io.Reader
	OutputStream io.Writer
	ErrorStream  io.Writer
	Tty          bool
	TtyWidth     int
	TtyHeight    int
	WorkingDir   string
}
