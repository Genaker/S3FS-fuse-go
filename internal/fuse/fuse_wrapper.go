package fuse

import (
	"context"
	"log"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// FuseFS implements the fuse.FS interface
type FuseFS struct {
	filesystem *Filesystem
}

var _ fs.FS = (*FuseFS)(nil)
var _ fs.FSStatfser = (*FuseFS)(nil)

// Root returns the root directory
func (f *FuseFS) Root() (fs.Node, error) {
	return &Dir{
		filesystem: f.filesystem,
		path:       "/",
	}, nil
}

// Statfs returns filesystem statistics
func (f *FuseFS) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	statfs, err := f.filesystem.Statfs(ctx)
	if err != nil {
		return err
	}
	resp.Blocks = statfs.Blocks
	resp.Bfree = statfs.Bfree
	resp.Bavail = statfs.Bavail
	resp.Files = statfs.Files
	resp.Ffree = statfs.Ffree
	resp.Bsize = uint32(statfs.Bsize)
	resp.Namelen = statfs.Namelen
	resp.Frsize = uint32(statfs.Bsize)
	return nil
}

// Dir represents a directory node
type Dir struct {
	filesystem *Filesystem
	path       string
}

var _ fs.Node = (*Dir)(nil)
var _ fs.NodeStringLookuper = (*Dir)(nil)
var _ fs.HandleReadDirAller = (*Dir)(nil)
var _ fs.NodeSetattrer = (*Dir)(nil)
var _ fs.NodeGetxattrer = (*Dir)(nil)
var _ fs.NodeSetxattrer = (*Dir)(nil)
var _ fs.NodeRemovexattrer = (*Dir)(nil)
var _ fs.NodeListxattrer = (*Dir)(nil)
var _ fs.NodeMkdirer = (*Dir)(nil)
var _ fs.NodeCreater = (*Dir)(nil)
var _ fs.NodeRemover = (*Dir)(nil)
var _ fs.NodeSymlinker = (*Dir)(nil)
var _ fs.NodeMknoder = (*Dir)(nil)
var _ fs.NodeAccesser = (*Dir)(nil)

// Attr returns directory attributes
func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	attr, err := d.filesystem.GetAttr(ctx, d.path)
	if err != nil {
		return err
	}
	a.Mode = os.ModeDir | attr.Mode
	a.Size = uint64(attr.Size)
	a.Mtime = attr.Mtime
	a.Uid = attr.Uid
	a.Gid = attr.Gid
	return nil
}

// Lookup looks up a child node
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += name

	attr, err := d.filesystem.GetAttr(ctx, childPath)
	if err != nil {
		return nil, syscall.ENOENT
	}

	if attr.Mode.IsDir() {
		return &Dir{
			filesystem: d.filesystem,
			path:       childPath,
		}, nil
	}

	return &File{
		filesystem: d.filesystem,
		path:       childPath,
	}, nil
}

// ReadDirAll reads all directory entries
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	entries, err := d.filesystem.ReadDir(ctx, d.path)
	if err != nil {
		return nil, err
	}

	dirents := make([]fuse.Dirent, 0, len(entries))
	for _, entry := range entries {
		dirent := fuse.Dirent{
			Name: entry.Name,
		}
		if entry.IsDir {
			dirent.Type = fuse.DT_Dir
		} else {
			dirent.Type = fuse.DT_File
		}
		dirents = append(dirents, dirent)
	}

	return dirents, nil
}

// Setattr sets directory attributes
func (d *Dir) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.Mode() {
		err := d.filesystem.Chmod(ctx, d.path, req.Mode)
		if err != nil {
			return err
		}
	}
	if req.Valid.Uid() || req.Valid.Gid() {
		uid := req.Uid
		gid := req.Gid
		if !req.Valid.Uid() || !req.Valid.Gid() {
			attr, err := d.filesystem.GetAttr(ctx, d.path)
			if err == nil {
				if !req.Valid.Uid() {
					uid = attr.Uid
				}
				if !req.Valid.Gid() {
					gid = attr.Gid
				}
			}
		}
		err := d.filesystem.Chown(ctx, d.path, uid, gid)
		if err != nil {
			return err
		}
	}
	attr, err := d.filesystem.GetAttr(ctx, d.path)
	if err != nil {
		return err
	}
	resp.Attr.Mode = os.ModeDir | attr.Mode
	resp.Attr.Size = uint64(attr.Size)
	resp.Attr.Mtime = attr.Mtime
	resp.Attr.Uid = attr.Uid
	resp.Attr.Gid = attr.Gid
	return nil
}

// Getxattr gets an extended attribute
func (d *Dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	value, err := d.filesystem.GetXattr(ctx, d.path, req.Name)
	if err != nil {
		return err
	}
	resp.Xattr = value
	return nil
}

// Setxattr sets an extended attribute
func (d *Dir) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	return d.filesystem.SetXattr(ctx, d.path, req.Name, req.Xattr)
}

// Removexattr removes an extended attribute
func (d *Dir) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	return d.filesystem.RemoveXattr(ctx, d.path, req.Name)
}

// Listxattr lists extended attributes
func (d *Dir) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	names, err := d.filesystem.ListXattr(ctx, d.path)
	if err != nil {
		return err
	}
	var buf []byte
	for _, name := range names {
		buf = append(buf, []byte(name)...)
		buf = append(buf, 0)
	}
	resp.Xattr = buf
	return nil
}

// Mkdir creates a new directory
func (d *Dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += req.Name
	
	err := d.filesystem.Mkdir(ctx, childPath, req.Mode)
	if err != nil {
		return nil, err
	}
	
	return &Dir{
		filesystem: d.filesystem,
		path:       childPath,
	}, nil
}

// Create creates a new file in the directory
func (d *Dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += req.Name
	
	err := d.filesystem.Create(ctx, childPath, req.Mode)
	if err != nil {
		return nil, nil, err
	}
	
	file := &File{
		filesystem: d.filesystem,
		path:       childPath,
	}
	
	resp.Handle = fuse.HandleID(0) // Not used, but required
	return file, file, nil
}

// Remove removes a file or empty directory
func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += req.Name
	
	// Check if it's a directory
	attr, err := d.filesystem.GetAttr(ctx, childPath)
	if err != nil {
		return err
	}
	
	if attr.Mode.IsDir() {
		// Remove directory
		return d.filesystem.Rmdir(ctx, childPath)
	}
	
	// Remove file
	return d.filesystem.Remove(ctx, childPath)
}

// Symlink creates a symbolic link
func (d *Dir) Symlink(ctx context.Context, req *fuse.SymlinkRequest) (fs.Node, error) {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += req.NewName
	
	err := d.filesystem.Symlink(ctx, req.Target, childPath)
	if err != nil {
		return nil, err
	}
	
	// Return a file node for the symlink
	return &File{
		filesystem: d.filesystem,
		path:       childPath,
	}, nil
}

// Mknod creates a special file (not supported)
func (d *Dir) Mknod(ctx context.Context, req *fuse.MknodRequest) (fs.Node, error) {
	childPath := d.path
	if childPath != "/" {
		childPath += "/"
	}
	childPath += req.Name
	
	err := d.filesystem.Mknod(ctx, childPath, req.Mode, req.Rdev)
	if err != nil {
		return nil, err
	}
	
	return &File{
		filesystem: d.filesystem,
		path:       childPath,
	}, nil
}

// Access checks file access permissions
func (d *Dir) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return d.filesystem.Access(ctx, d.path, req.Mask)
}

// Opendir opens a directory handle - implemented as part of HandleReadDirAller
// No explicit opendir needed, handled by ReadDirAll

// File represents a file node
type File struct {
	filesystem *Filesystem
	path       string
}

var _ fs.Node = (*File)(nil)
var _ fs.NodeOpener = (*File)(nil)
var _ fs.HandleReader = (*File)(nil)
var _ fs.HandleWriter = (*File)(nil)
var _ fs.NodeSetattrer = (*File)(nil)
var _ fs.NodeGetxattrer = (*File)(nil)
var _ fs.NodeSetxattrer = (*File)(nil)
var _ fs.NodeRemovexattrer = (*File)(nil)
var _ fs.NodeListxattrer = (*File)(nil)
var _ fs.NodeReadlinker = (*File)(nil)
var _ fs.NodeLinker = (*File)(nil)
var _ fs.NodeAccesser = (*File)(nil)
var _ fs.NodeFsyncer = (*File)(nil)
var _ fs.HandleFlusher = (*File)(nil)
var _ fs.HandleReleaser = (*File)(nil)

// Attr returns file attributes
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	attr, err := f.filesystem.GetAttr(ctx, f.path)
	if err != nil {
		return err
	}
	a.Mode = attr.Mode
	a.Size = uint64(attr.Size)
	a.Mtime = attr.Mtime
	a.Uid = attr.Uid
	a.Gid = attr.Gid
	return nil
}

// Open opens a file
func (f *File) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	return f, nil
}

// Read reads file data
func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	data, err := f.filesystem.ReadFile(ctx, f.path, req.Offset, int64(req.Size))
	if err != nil {
		return err
	}
	resp.Data = data
	return nil
}

// Write writes file data
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	err := f.filesystem.WriteFile(ctx, f.path, req.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Size = len(req.Data)
	return nil
}

// Setattr sets file attributes
func (f *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.Mode() {
		err := f.filesystem.Chmod(ctx, f.path, req.Mode)
		if err != nil {
			return err
		}
	}
	if req.Valid.Uid() || req.Valid.Gid() {
		uid := req.Uid
		gid := req.Gid
		// Get current values if not set
		if !req.Valid.Uid() || !req.Valid.Gid() {
			attr, err := f.filesystem.GetAttr(ctx, f.path)
			if err == nil {
				if !req.Valid.Uid() {
					uid = attr.Uid
				}
				if !req.Valid.Gid() {
					gid = attr.Gid
				}
			}
		}
		err := f.filesystem.Chown(ctx, f.path, uid, gid)
		if err != nil {
			return err
		}
	}
	// Update response with new attributes
	attr, err := f.filesystem.GetAttr(ctx, f.path)
	if err != nil {
		return err
	}
	resp.Attr.Mode = attr.Mode
	resp.Attr.Size = uint64(attr.Size)
	resp.Attr.Mtime = attr.Mtime
	resp.Attr.Uid = attr.Uid
	resp.Attr.Gid = attr.Gid
	return nil
}

// Getxattr gets an extended attribute
func (f *File) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	value, err := f.filesystem.GetXattr(ctx, f.path, req.Name)
	if err != nil {
		return err
	}
	resp.Xattr = value
	return nil
}

// Setxattr sets an extended attribute
func (f *File) Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error {
	return f.filesystem.SetXattr(ctx, f.path, req.Name, req.Xattr)
}

// Removexattr removes an extended attribute
func (f *File) Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error {
	return f.filesystem.RemoveXattr(ctx, f.path, req.Name)
}

// Listxattr lists extended attributes
func (f *File) Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error {
	names, err := f.filesystem.ListXattr(ctx, f.path)
	if err != nil {
		return err
	}
	// Convert names to null-terminated strings
	var buf []byte
	for _, name := range names {
		buf = append(buf, []byte(name)...)
		buf = append(buf, 0)
	}
	resp.Xattr = buf
	return nil
}

// Readlink reads the target of a symbolic link
func (f *File) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	return f.filesystem.Readlink(ctx, f.path)
}

// Link creates a hard link (not supported)
func (f *File) Link(ctx context.Context, req *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
	oldFile, ok := old.(*File)
	if !ok {
		return nil, syscall.EINVAL
	}
	
	err := f.filesystem.Link(ctx, oldFile.path, f.path)
	if err != nil {
		return nil, err
	}
	
	return &File{
		filesystem: f.filesystem,
		path:       f.path,
	}, nil
}

// Access checks file access permissions
func (f *File) Access(ctx context.Context, req *fuse.AccessRequest) error {
	return f.filesystem.Access(ctx, f.path, req.Mask)
}

// Flush flushes file buffers
func (f *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return f.filesystem.Flush(ctx, f.path)
}

// Fsync syncs file data to storage
func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	// req.Flags: bit 1 is datasync (sync data only), 0 = fsync (sync data and metadata)
	datasync := req.Flags&1 != 0
	return f.filesystem.Fsync(ctx, f.path, datasync)
}

// Release releases a file handle
func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return f.filesystem.Release(ctx, f.path)
}

// MountOptions contains options for mounting the filesystem
type MountOptions struct {
	EnableFileLock bool // Enable file-level advisory locking (default: false)
}

// Mount mounts the filesystem at the given mountpoint
func Mount(mountpoint string, client S3ClientInterface) error {
	return MountWithOptions(mountpoint, client, MountOptions{})
}

// MountWithOptions mounts the filesystem at the given mountpoint with options
func MountWithOptions(mountpoint string, client S3ClientInterface, options MountOptions) error {
	filesystem := NewFilesystem(client)
	if options.EnableFileLock {
		filesystem.SetEnableFileLock(true)
	}
	fuseFS := &FuseFS{
		filesystem: filesystem,
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("s3fs"),
		fuse.Subtype("s3fs-go"),
	)
	if err != nil {
		return err
	}
	defer c.Close()

	log.Printf("Mounted filesystem at %s", mountpoint)

	err = fs.Serve(c, fuseFS)
	if err != nil {
		return err
	}

	return nil
}
