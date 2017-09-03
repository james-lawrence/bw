package fil

import (
  "os"
)

/*
type Path struct {
}

func New(path string) (Path, error) {
  fi, err := os.Lstat(path)
  if err != nil {
    return fi, err
  }
  return fi, err
}

func (p *Path) IsExists() bool {
  xx
}
*/


// Exists checks whether the given file exists or not.
func IsExists(path string) (bool, error) {
	_, err := os.Stat(path)
  if err != nil {
    if os.IsNotExist(err) {
      return false, nil
    } else {
      return false, err
    }
  }
  return true, nil
}

func TypeFileInfo(fi os.FileInfo) string {
  m := fi.Mode()
  switch {
  case m.IsDir():
    return "dir"
  case m.IsRegular():
    return "regular"
  case m & os.ModeSymlink != 0:
    return "symlink"
  case m & os.ModeNamedPipe != 0:
    return "namedpipe"
  case m & os.ModeSocket != 0:
    return "socket"
  case m & os.ModeDevice != 0:
    return "device"
  default:
    return ""
  }
}

func Type(path string) (string, error) {
  fi, err := os.Lstat(path)
  if err != nil {
    return "", err
  }
  return TypeFileInfo(fi), nil
}

func IsNotExist(path string) (bool, error) {
  ret, err := IsExists(path)
  return ! ret, err
}

func IsFile(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return !fi.Mode().IsDir(), err
}

func IsDir(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return fi.Mode().IsDir(), err
}

func IsRegular(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return fi.Mode().IsRegular(), err
}

func isReadable(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return fi.Mode() & 0400 != 0 , err
}

func isWritable(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return fi.Mode() & 0200 != 0 , err
}

func isExecutable(path string) (bool, error) {
  fi, err :=  os.Stat(path)
  if err != nil {
    return false, err
  }
  return fi.Mode() & 0100 != 0 , err
}
