package memory

import (
	"context"
	"io"
	"strings"

	"github.com/beyondstorage/go-storage/v4/services"
	. "github.com/beyondstorage/go-storage/v4/types"
)

func (s *Storage) commitAppend(ctx context.Context, o *Object, opt pairStorageCommitAppend) (err error) {
	return
}

func (s *Storage) copy(ctx context.Context, src string, dst string, opt pairStorageCopy) (err error) {
	rs := s.absPath(src)
	rd := s.absPath(dst)

	ro := s.root.getObjectByPath(rs)
	if ro == nil {
		return services.ErrObjectNotExist
	}

	r := s.root.getObjectByPath(rd)
	if r != nil && r.mode.IsDir() {
		return services.ErrObjectModeInvalid
	}

	o := s.root.insertChildByPath(rd)
	if o == nil {
		return services.ErrObjectModeInvalid
	}

	o.length = ro.length
	o.mode = ro.mode

	o.data = make([]byte, ro.length)
	copy(o.data, ro.data)
	return nil
}

func (s *Storage) create(path string, opt pairStorageCreate) (o *Object) {
	o = NewObject(s, true)
	o.ID = s.absPath(path)
	o.Path = s.relPath(path)
	if opt.HasObjectMode && opt.ObjectMode.IsDir() {
		o.Mode = ModeDir
	}
	return o
}

func (s *Storage) createAppend(ctx context.Context, path string, opt pairStorageCreateAppend) (o *Object, err error) {
	child := s.root.insertChildByPath(s.absPath(path))
	if child == nil {
		return nil, services.ErrObjectModeInvalid
	}
	child.mode = ModeRead | ModeAppend

	o = NewObject(s, true)
	o.ID = s.absPath(path)
	o.Path = s.relPath(path)
	o.Mode = ModeRead | ModeAppend
	o.SetAppendOffset(0)

	return o, nil
}

func (s *Storage) createDir(ctx context.Context, path string, opt pairStorageCreateDir) (o *Object, err error) {
	if s.root.makeDirAll(strings.Split(s.absPath(path), "/")) == nil {
		return nil, services.ErrObjectModeInvalid
	}

	o = NewObject(s, true)
	o.ID = s.absPath(path)
	o.Path = s.relPath(path)
	o.Mode |= ModeDir
	return o, nil
}

func (s *Storage) delete(ctx context.Context, path string, opt pairStorageDelete) (err error) {
	o := s.root.getObjectByPath(s.absPath(path))
	if o == nil {
		return nil
	}
	o.parent.removeChild(o.name)
	return nil
}

func (s *Storage) list(ctx context.Context, path string, opt pairStorageList) (oi *ObjectIterator, err error) {
	fn := NextObjectFunc(func(ctx context.Context, page *ObjectPage) error {
		o := s.root.getObjectByPath(s.absPath(path))
		if o == nil {
			// If the object is not exist, we should return IterateDone instead.
			return IterateDone
		}
		if !o.mode.IsDir() {
			// If the object mode is not dir, we should return directly.
			return services.ErrObjectModeInvalid
		}

		o.mu.Lock()
		defer o.mu.Unlock()

		for k, v := range o.child {
			xo := NewObject(s, true)
			xo.ID = s.absPath(path + "/" + k)
			xo.Path = s.relPath(path + "/" + k)
			xo.Mode = v.mode
			xo.SetContentLength(v.length)

			page.Data = append(page.Data, xo)
		}
		return IterateDone
	})
	return NewObjectIterator(ctx, fn, nil), nil
}

func (s *Storage) metadata(opt pairStorageMetadata) (meta *StorageMeta) {
	return &StorageMeta{
		Name:    "memory",
		WorkDir: "/",
	}
}

func (s *Storage) move(ctx context.Context, src string, dst string, opt pairStorageMove) (err error) {
	rs := s.absPath(src)
	rd := s.absPath(dst)

	rso := s.root.getObjectByPath(rs)
	if rso == nil {
		return services.ErrObjectNotExist
	}

	rdo := s.root.getObjectByPath(rd)
	if rdo != nil && rdo.mode.IsDir() {
		return services.ErrObjectModeInvalid
	}

	ps := strings.Split(dst, "/")
	rso.parent.removeChild(rso.name)
	rso.parent.insertChild(ps[len(ps)-1], rso)
	return
}

func (s *Storage) read(ctx context.Context, path string, w io.Writer, opt pairStorageRead) (n int64, err error) {
	o := s.root.getObjectByPath(s.absPath(path))
	if o == nil {
		return 0, services.ErrObjectNotExist
	}

	offset := int64(0)
	if opt.HasOffset {
		offset = opt.Offset
	}

	written, err := w.Write(o.data[offset:])
	if err != nil {
		return int64(written), err
	}
	return int64(written), nil
}

func (s *Storage) stat(ctx context.Context, path string, opt pairStorageStat) (o *Object, err error) {
	ro := s.root.getObjectByPath(s.absPath(path))
	if ro == nil {
		return nil, services.ErrObjectNotExist
	}

	o = NewObject(s, true)
	o.ID = s.absPath(path)
	o.Path = s.relPath(path)
	o.Mode = ro.mode
	o.SetContentLength(ro.length)
	return o, nil
}

func (s *Storage) write(ctx context.Context, path string, r io.Reader, size int64, opt pairStorageWrite) (n int64, err error) {
	o := s.root.insertChildByPath(s.absPath(path))
	if o == nil {
		return 0, services.ErrObjectModeInvalid
	}

	o.length = size
	o.mode = ModeRead

	o.data = make([]byte, size)
	read, err := r.Read(o.data)
	if err != nil {
		return int64(read), nil
	}

	return int64(read), nil
}

func (s *Storage) writeAppend(ctx context.Context, o *Object, r io.Reader, size int64, opt pairStorageWriteAppend) (n int64, err error) {
	ro := s.root.getObjectByPath(o.ID)
	if ro == nil {
		ro = s.root.insertChildByPath(o.ID)
		if ro == nil {
			return 0, services.ErrObjectModeInvalid
		}
	}
	buf := make([]byte, size)
	read, err := r.Read(buf)
	ro.data = append(ro.data, buf[:read]...)
	ro.length += int64(read)
	if err != nil {
		return int64(read), nil
	}
	return int64(read), nil
}
