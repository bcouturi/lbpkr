package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func path_exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func handle_err(err error) {
	if err != nil {
		if g_ctx != nil {
			g_ctx.msg.Errorf("%v\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "**error** %v\n", err)
		}
		os.Exit(1)
	}
}

func PrintHeader(ctx *Context) {
	now := time.Now()
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Infof(
		"<<< %s - start of lbpkr-%s installation >>>\n",
		now, Version,
	)
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Debugf("cmd line args: %v\n", os.Args)
}

func PrintTrailer(ctx *Context) {
	now := time.Now()
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Infof(
		"<<< %s - end of lbpkr-%s installation >>>\n",
		now, Version,
	)
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
}

func bincp(dst, src string) error {
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	fdst, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fdst.Close()

	fisrc, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = fdst.Chmod(fisrc.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(fdst, fsrc)
	return err
}

func _tar_gz(targ, workdir string) error {

	// FIXME: use archive/tar instead (once go-1.1 is out)
	{
		matches, err := filepath.Glob(filepath.Join(workdir, "*"))
		if err != nil {
			return err
		}
		for i, m := range matches {
			matches[i] = m[len(workdir)+1:]
		}
		args := []string{"-zcf", targ}
		args = append(args, matches...)
		//fmt.Printf(">> %v\n", args)
		cmd := exec.Command("tar", args...)
		cmd.Dir = workdir
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	f, err := os.Create(targ)
	if err != nil {
		return err
	}
	zout := gzip.NewWriter(f)
	tw := tar.NewWriter(zout)

	err = filepath.Walk(workdir, func(path string, fi os.FileInfo, err error) error {
		//fmt.Printf("::> [%s]...\n", path)
		if !strings.HasPrefix(path, workdir) {
			err = fmt.Errorf("walked filename %q doesn't begin with workdir %q", path, workdir)
			return err

		}
		name := path[len(workdir):] //path

		// make name "relative"
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		target, _ := os.Readlink(path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		hdr.Name = name
		hdr.Uname = "root"
		hdr.Gname = "root"
		hdr.Uid = 0
		hdr.Gid = 0

		// Force permissions to 0755 for executables, 0644 for everything else.
		if fi.Mode().Perm()&0111 != 0 {
			hdr.Mode = hdr.Mode&^0777 | 0755
		} else {
			hdr.Mode = hdr.Mode&^0777 | 0644
		}

		err = tw.WriteHeader(hdr)
		if err != nil {
			return fmt.Errorf("Error writing file %q: %v", name, err)
		}
		// handle directories and symlinks
		if hdr.Size <= 0 {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(tw, r)
		return err
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := zout.Close(); err != nil {
		return err
	}
	return f.Close()
}

// EOF
