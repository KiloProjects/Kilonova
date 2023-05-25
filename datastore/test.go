package datastore

import (
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"

	"go.uber.org/zap"
)

var (
	dos2unixPath  string
	dos2unixFound bool
)

func (m *StorageManager) TestInput(testID int) (io.ReadSeekCloser, error) {
	return os.Open(m.TestInputPath(testID))
}

func (m *StorageManager) TestOutput(testID int) (io.ReadSeekCloser, error) {
	return os.Open(m.TestOutputPath(testID))
}

func (m *StorageManager) TestInputPath(testID int) string {
	return path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".in")
}

func (m *StorageManager) TestOutputPath(testID int) string {
	return path.Join(m.RootPath, "tests", strconv.Itoa(testID)+".out")
}

func (m *StorageManager) SaveTestInput(testID int, input io.Reader) error {
	testPath := m.TestInputPath(testID)
	defer func() {
		go m.dos2unixify(testPath)
	}()
	return writeFile(testPath, input, 0777)
}

func (m *StorageManager) SaveTestOutput(testID int, output io.Reader) error {
	testPath := m.TestOutputPath(testID)
	defer func() {
		go m.dos2unixify(testPath)
	}()
	return writeFile(testPath, output, 0777)
}

func (m *StorageManager) dos2unixify(path string) {
	if !dos2unixFound {
		return
	}
	zap.S().Debugf("Running dos2unix on %q", path)

	cmd := exec.Command(dos2unixPath, path)
	if val, err := cmd.CombinedOutput(); err != nil {
		zap.S().Warn("dos2unix exited with nonzero status code: ", err)
		zap.S().Infof("dos2unix output: %q", string(val))
	}
}

func (m *StorageManager) initDos2Unix() {
	path, err := exec.LookPath("dos2unix")
	if err != nil {
		zap.S().Warn("dos2unix was not found. Added tests will not be automatically converted.")
		return
	}
	dos2unixFound = true
	dos2unixPath = path
}
