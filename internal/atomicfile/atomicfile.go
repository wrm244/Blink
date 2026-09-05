// Package atomicfile 提供原子性的小文件写入：先写同目录临时文件，
// 再 rename 到目标路径。rename 在同一文件系统上是原子的，因此进程
// 在写入中途崩溃或断电时，目标文件要么保持旧内容、要么已是完整的
// 新内容，绝不会留下截断损坏的半份文件。
//
// 这对 settings.json / stats.json 至关重要：损坏的设置文件会让用户的
// 全部配置静默回退默认值，损坏的统计文件则丢掉全部历史记录。
package atomicfile

import (
	"os"
	"path/filepath"
)

// WriteFile 将 data 原子性地写入 path，文件权限为 perm。
// 临时文件建在 path 所在目录（保证与目标在同一文件系统上，rename
// 才是原子的），失败时会被清理。
func WriteFile(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	// 兜底清理：成功路径上 rename 之后 Remove 找不到文件，无害；
	// 显式 Close 之后这里的二次 Close 返回错误，同样被丢弃。
	defer func() {
		f.Close()
		os.Remove(tmpName)
	}()

	if _, err := f.Write(data); err != nil {
		return err
	}
	// CreateTemp 固定 0600，写完改为调用方要求的权限。
	if err := f.Chmod(perm); err != nil {
		return err
	}
	// rename 前必须关闭：Windows 上无法重命名处于打开状态的文件。
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
