package lmdb

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
)

func TestEnv_Path_notOpen(t *testing.T) {
	env, err := NewEnv()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer env.Close()

	// before Open the Path method returns "" and a non-nil error.
	path, err := env.Path()
	if err == nil {
		t.Errorf("no error returned before Open")
	}
	if path != "" {
		t.Errorf("non-zero path returned before Open")
	}
}

func TestEnv_Path(t *testing.T) {
	env, err := NewEnv()
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// open an environment
	dir, err := ioutil.TempDir("", "mdb_test")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)

	err = env.Open(dir, 0, 0644)
	defer env.Close()
	if err != nil {
		t.Errorf("open: %v", err)
	}
	path, err := env.Path()
	if err != nil {
		t.Errorf("path: %v", err)
	}
	if path != dir {
		t.Errorf("path: %q (!= %q)", path, dir)
	}
}

func TestEnv_Open_notExist(t *testing.T) {
	env, err := NewEnv()
	if err != nil {
		t.Fatalf("create: %s", err)
	}
	defer env.Close()

	// ensure that opening a non-existent path fails.
	err = env.Open("/path/does/not/exist/aoeu", 0, 0664)
	if !IsNotExist(err) {
		t.Errorf("open: %v", err)
	}
}

func TestEnv_Open(t *testing.T) {
	env, err := NewEnv()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := env.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	// open an environment at a temporary path.
	path, err := ioutil.TempDir("/tmp", "mdb_test")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(path)
	err = env.Open(path, 0, 0664)
	if err != nil {
		t.Errorf("open: %s", err)
	}
}

func TestEnv_Flags(t *testing.T) {
	env := setup(t)
	defer clean(env, t)

	flags, err := env.Flags()
	if err != nil {
		t.Error(err)
		return
	}

	if flags&NoTLS == 0 {
		t.Errorf("NoTLS is not set")
	}
	if flags&NoSync != 0 {
		t.Errorf("NoSync is set")
	}

	err = env.SetFlags(NoSync)
	if err != nil {
		t.Error(err)
	}

	flags, err = env.Flags()
	if err != nil {
		t.Error(err)
	}
	if flags&NoSync == 0 {
		t.Error("NoSync is not set")
	}

	err = env.UnsetFlags(NoSync)
	if err != nil {
		t.Error(err)
	}

	flags, err = env.Flags()
	if err != nil {
		t.Error(err)
	}
	if flags&NoSync != 0 {
		t.Error("NoSync is set")
	}
}

func TestEnv_SetMaxReader(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-env-setmaxreaders-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	env, err := NewEnv()
	if err != nil {
		t.Error(err)
	}

	maxreaders := 5
	err = env.SetMaxReaders(maxreaders)
	if err != nil {
		t.Error(err)
	}
	_maxreaders, err := env.MaxReaders()
	if err != nil {
		t.Error(err)
	}
	if _maxreaders != maxreaders {
		t.Errorf("unexpected MaxReaders: %v (!= %v)", _maxreaders, maxreaders)
	}

	err = env.Open(dir, 0, 0644)
	defer env.Close()
	if err != nil {
		env.Close()
		t.Error(err)
	}

	err = env.SetMaxReaders(126)
	if !IsErrnoSys(err, syscall.EINVAL) {
		t.Errorf("unexpected error: %v (!= %v)", err, Invalid)
	}
	_maxreaders, err = env.MaxReaders()
	if err != nil {
		t.Error(err)
	}
	if _maxreaders != maxreaders {
		t.Errorf("unexpected MaxReaders: %v (!= %v)", _maxreaders, maxreaders)
	}
}

func TestEnv_SetMapSize(t *testing.T) {
	env := setup(t)
	defer clean(env, t)

	const minsize = 100 << 20 // 100MB
	err := env.SetMapSize(minsize)
	if err != nil {
		t.Error(err)
	}

	err = env.Update(func(txn *Txn) (err error) {
		return nil
	})
	if err != nil {
		t.Error(err)
	}

	info, err := env.Info()
	if err != nil {
		t.Error(err)
	} else if info.MapSize < minsize {
		t.Errorf("unexpected mapsize: %v (< %v)", info.MapSize, minsize)
	}
}

func setup(t T) *Env {
	env, err := NewEnv()
	if err != nil {
		t.Fatalf("env: %s", err)
	}
	path, err := ioutil.TempDir("/tmp", "mdb_test")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	err = os.MkdirAll(path, 0770)
	if err != nil {
		t.Fatalf("mkdir: %s", path)
	}
	err = env.SetMaxDBs(64 << 10)
	if err != nil {
		t.Fatalf("setmaxdbs: %v", err)
	}
	err = env.Open(path, 0, 0664)
	if err != nil {
		t.Fatalf("open: %s", err)
	}

	return env
}

type T interface {
	Errorf(format string, vals ...interface{})
	Fatalf(format string, vals ...interface{})
}

func clean(env *Env, t T) {
	path, err := env.Path()
	if err != nil {
		t.Errorf("path: %v", err)
	}
	err = env.Close()
	if err != nil {
		t.Errorf("close: %s", err)
	}
	if path != "" {
		err = os.RemoveAll(path)
		if err != nil {
			t.Errorf("remove: %v", err)
		}
	}
}

func TestEnvCopy(t *testing.T) {
	env := setup(t)
	defer clean(env, t)
}

func TestEnv_MaxKeySize(t *testing.T) {
	env := setup(t)
	defer clean(env, t)

	n := env.MaxKeySize()
	if n <= 0 {
		t.Errorf("invaild maxkeysize: %d", n)
	}
}

func TestEnv_MaxKeySize_nil(t *testing.T) {
	var env *Env
	n := env.MaxKeySize()
	if n <= 0 {
		t.Errorf("invaild maxkeysize: %d", n)
	}
	t.Logf("mdb_env_get_maxkeysize: %d", n)
}
