package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

const maxFrameSize = 1 << 20 // 1 MiB

// WriteFrame 写入 4 字节大端长度 + payload。
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxFrameSize {
		return fmt.Errorf("frame: payload too large: %d", len(payload))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame 读取一帧；length 为 0 或超过上限时返回错误。
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > maxFrameSize {
		return nil, fmt.Errorf("frame: invalid length %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
