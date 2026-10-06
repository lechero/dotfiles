package scan

import "syscall"

// Volume is the disk a path lives on. On APFS all volumes of a container share
// its space, so Used is the whole container's use — more than the "Used" df
// prints for one volume, and the number that decides when the disk is full.
type Volume struct {
	Total, Free int64
}

// Used is everything but the free space.
func (v Volume) Used() int64 { return max(0, v.Total-v.Free) }

// VolumeOf reads the volume that holds path.
func VolumeOf(path string) (Volume, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return Volume{}, err
	}
	bs := int64(st.Bsize)
	return Volume{Total: int64(st.Blocks) * bs, Free: int64(st.Bavail) * bs}, nil
}
