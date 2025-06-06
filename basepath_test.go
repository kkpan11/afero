package afero

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasePath(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	if _, err := bp.Create("/tmp/foo"); err != nil {
		t.Errorf("Failed to set real path")
	}

	if fh, err := bp.Create("../tmp/bar"); err == nil {
		t.Errorf("succeeded in creating %s ...", fh.Name())
	}
}

func TestBasePathRoot(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/foo/baz", 0o777)
	baseFs.MkdirAll("/base/path/boo/", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	rd, err := ReadDir(bp, string(os.PathSeparator))

	if len(rd) != 2 {
		t.Errorf("base path doesn't respect root")
	}

	if err != nil {
		t.Error(err)
	}
}

func TestRealPath(t *testing.T) {
	fs := NewOsFs()
	baseDir, err := TempDir(fs, "", "base")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer fs.RemoveAll(baseDir)
	anotherDir, err := TempDir(fs, "", "another")
	if err != nil {
		t.Fatal("error creating tempDir", err)
	}
	defer fs.RemoveAll(anotherDir)

	bp := NewBasePathFs(fs, baseDir).(*BasePathFs)

	subDir := filepath.Join(baseDir, "s1")

	realPath, err := bp.RealPath("/s1")
	if err != nil {
		t.Errorf("Got error %s", err)
	}

	if realPath != subDir {
		t.Errorf("Expected \n%s got \n%s", subDir, realPath)
	}

	if runtime.GOOS == "windows" {
		_, err = bp.RealPath(anotherDir)

		if err != os.ErrNotExist {
			t.Errorf("Expected os.ErrNotExist")
		}

	} else {
		// on *nix we have no way of just looking at the path and tell that anotherDir
		// is not inside the base file system.
		// The user will receive an os.ErrNotExist later.
		surrealPath, err := bp.RealPath(anotherDir)
		if err != nil {
			t.Errorf("Got error %s", err)
		}

		excpected := filepath.Join(baseDir, anotherDir)

		if surrealPath != excpected {
			t.Errorf("Expected \n%s got \n%s", excpected, surrealPath)
		}
	}
}

func TestNestedBasePaths(t *testing.T) {
	type dirSpec struct {
		Dir1, Dir2, Dir3 string
	}
	dirSpecs := []dirSpec{
		{Dir1: "/", Dir2: "/", Dir3: "/"},
		{Dir1: "/", Dir2: "/path2", Dir3: "/"},
		{Dir1: "/path1/dir", Dir2: "/path2/dir/", Dir3: "/path3/dir"},
		{Dir1: "C:/path1", Dir2: "path2/dir", Dir3: "/path3/dir/"},
	}

	for _, ds := range dirSpecs {
		memFs := NewMemMapFs()
		level1Fs := NewBasePathFs(memFs, ds.Dir1)
		level2Fs := NewBasePathFs(level1Fs, ds.Dir2)
		level3Fs := NewBasePathFs(level2Fs, ds.Dir3)

		type spec struct {
			BaseFs   Fs
			FileName string
		}
		specs := []spec{
			{BaseFs: level3Fs, FileName: "f.txt"},
			{BaseFs: level2Fs, FileName: "f.txt"},
			{BaseFs: level1Fs, FileName: "f.txt"},
		}

		for _, s := range specs {
			if err := s.BaseFs.MkdirAll(s.FileName, 0o755); err != nil {
				t.Errorf("Got error %s", err.Error())
			}
			if _, err := s.BaseFs.Stat(s.FileName); err != nil {
				t.Errorf("Got error %s", err.Error())
			}

			switch s.BaseFs {
			case level3Fs:
				pathToExist := filepath.Join(ds.Dir3, s.FileName)
				if _, err := level2Fs.Stat(pathToExist); err != nil {
					t.Errorf("Got error %s (path %s)", err.Error(), pathToExist)
				}

			case level2Fs:
				pathToExist := filepath.Join(ds.Dir2, ds.Dir3, s.FileName)
				if _, err := level1Fs.Stat(pathToExist); err != nil {
					t.Errorf("Got error %s (path %s)", err.Error(), pathToExist)
				}
			}
		}
	}
}

func TestBasePathOpenFile(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")
	f, err := bp.OpenFile("/tmp/file.txt", os.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	if filepath.Dir(f.Name()) != filepath.Clean("/tmp") {
		t.Fatalf("realpath leaked: %s", f.Name())
	}
}

func TestBasePathCreate(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")
	f, err := bp.Create("/tmp/file.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if filepath.Dir(f.Name()) != filepath.Clean("/tmp") {
		t.Fatalf("realpath leaked: %s", f.Name())
	}
}

func TestBasePathTempFile(t *testing.T) {
	baseFs := &MemMapFs{}
	baseFs.MkdirAll("/base/path/tmp", 0o777)
	bp := NewBasePathFs(baseFs, "/base/path")

	tDir, err := TempDir(bp, "/tmp", "")
	if err != nil {
		t.Fatalf("Failed to TempDir: %v", err)
	}
	if filepath.Dir(tDir) != filepath.Clean("/tmp") {
		t.Fatalf("Tempdir realpath leaked: %s", tDir)
	}
	tempFile, err := TempFile(bp, tDir, "")
	if err != nil {
		t.Fatalf("Failed to TempFile: %v", err)
	}
	defer tempFile.Close()
	if expected, actual := tDir, filepath.Dir(tempFile.Name()); expected != actual {
		t.Fatalf("TempFile realpath leaked: expected %s, got %s", expected, actual)
	}
}
