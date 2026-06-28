/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/goplus/llgo/internal/meta"
)

const (
	cacheMetaMemberName = "index.meta"
	arGlobalHeader      = "!<arch>\n"
	arHeaderSize        = 60
)

func (c *context) writeMetaMember(archivePath string, pm *meta.PackageMeta) error {
	tmpDir, err := os.MkdirTemp(filepath.Dir(archivePath), "llgo-meta-member-*")
	if err != nil {
		return fmt.Errorf("create meta member temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	metaPath := filepath.Join(tmpDir, cacheMetaMemberName)
	if err := os.WriteFile(metaPath, pm.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write meta member: %w", err)
	}

	arCmd := c.archiver()
	args := []string{"r", archivePath, metaPath}
	if c.shouldPrintCommands(false) {
		fmt.Fprintf(os.Stderr, "%s %s\n", filepath.Base(arCmd), strings.Join(args, " "))
	}
	if output, err := exec.Command(arCmd, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("append meta member to archive %s: %w\n%s", archivePath, err, output)
	}
	if pm, err := readMetaFromArchive(archivePath); err != nil {
		return fmt.Errorf("verify meta member in archive %s: %w", archivePath, err)
	} else {
		_ = pm.Close()
	}
	return nil
}

func readMetaFromArchive(archivePath string) (*meta.PackageMeta, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(fi.Size())
	if size < len(arGlobalHeader)+arHeaderSize {
		return nil, fmt.Errorf("archive too small: %s", archivePath)
	}

	raw, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap archive %s: %w", archivePath, err)
	}

	member, err := arMember(raw, cacheMetaMemberName)
	if err != nil {
		_ = syscall.Munmap(raw)
		return nil, err
	}

	pm, err := meta.MmapView(member, raw)
	if err != nil {
		_ = syscall.Munmap(raw)
		return nil, err
	}
	return pm, nil
}

func arMember(raw []byte, name string) ([]byte, error) {
	if len(raw) < len(arGlobalHeader) || string(raw[:len(arGlobalHeader)]) != arGlobalHeader {
		return nil, fmt.Errorf("archive: bad global header")
	}

	for off := len(arGlobalHeader); off+arHeaderSize <= len(raw); {
		hdr := raw[off : off+arHeaderSize]
		if string(hdr[58:60]) != "`\n" {
			return nil, fmt.Errorf("archive: bad member header at offset %d", off)
		}

		size, err := strconv.ParseInt(strings.TrimSpace(string(hdr[48:58])), 10, 64)
		if err != nil || size < 0 {
			return nil, fmt.Errorf("archive: bad member size at offset %d", off)
		}

		payloadOff := off + arHeaderSize
		payloadEnd := payloadOff + int(size)
		if payloadEnd < payloadOff || payloadEnd > len(raw) {
			return nil, fmt.Errorf("archive: truncated member at offset %d", off)
		}

		memberName := strings.TrimRight(string(hdr[:16]), " ")
		dataOff := payloadOff
		dataEnd := payloadEnd
		if strings.HasPrefix(memberName, "#1/") {
			n, err := strconv.Atoi(strings.TrimPrefix(memberName, "#1/"))
			if err != nil || n < 0 || payloadOff+n > payloadEnd {
				return nil, fmt.Errorf("archive: bad extended name at offset %d", off)
			}
			memberName = normalizeArMemberName(string(raw[payloadOff : payloadOff+n]))
			dataOff = payloadOff + n
		} else {
			memberName = normalizeArMemberName(memberName)
		}

		if memberName == name {
			return raw[dataOff:dataEnd], nil
		}

		off = payloadEnd
		if size%2 != 0 {
			off++
		}
	}

	return nil, fmt.Errorf("archive: member %q not found", name)
}

func normalizeArMemberName(name string) string {
	name = strings.TrimRight(name, "\x00")
	if name != "/" && name != "//" && strings.HasSuffix(name, "/") {
		return strings.TrimSuffix(name, "/")
	}
	return name
}
