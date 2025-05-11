package utils

import (
	"cli/internal/common"
	"github.com/otiai10/copy"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

func CreateRecursiveLinkTree(root string, targetRoot string) error {
	defer common.ExecutionTimer().Log()
	err := filepath.WalkDir(root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("failed walkdir", "root", root, "path", path, "err", err)
			return err
		}

		if root == path {
			// is the root of the entire tree
			return nil
		}

		info, err := de.Info()
		if err != nil {
			slog.Error("failed getting info", "entry", de)
			return err
		}

		ft := de.Type()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			slog.Error("failed getting rel path", "root", root, "path", path)
			return err
		}

		target := filepath.Join(targetRoot, rel)

		if info.Mode()&os.ModeSymlink != 0 {
			slog.Warn("found symlink - copyin as is", "path", path, "target", target)

			opts := copy.Options{
				PreserveTimes: true,
				PreserveOwner: true,
				OnSymlink: func(src string) copy.SymlinkAction {
					return copy.Shallow
				}}

			if err := copy.Copy(path, target, opts); err != nil {
				slog.Error("failed copying rel path", "target", target, "path", path)
				return err
			}

			return nil
		}

		if ft.IsDir() {
			if err := os.Mkdir(target, os.ModePerm); err != nil {
				slog.Error("failed making dir in target", "target", target)
				return err
			}

			return nil
		}

		if ft.IsRegular() {
			// file - link it
			if err := os.Symlink(path, target); err != nil {
				slog.Error("failed making symlink to file in target", "path", path, "target", target)
				return err
			}

			return nil
		}

		slog.Warn("unsupported dir entry type", "entry", de, "file-mode", ft)

		return nil
	})

	return err
}
