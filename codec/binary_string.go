package codec

import (
  "io"
  "github.com/pkg/errors"
)

type BinaryString struct {}

type BinaryStringDecoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

type BinaryStringEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func NewCodecBinaryString() (Codec) {
  return &BinaryString{}
}

func (codec BinaryString) Decoder() (CodecDecoder) {
  dec := &BinaryStringDecoder{}
  dec.pipeReader, dec.pipeWriter = io.Pipe()
  return dec
}

func (codec BinaryString) Encoder() (CodecEncoder) {
  enc := &BinaryStringEncoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  return enc
}

func (dec BinaryStringDecoder) Init() (error) {
  return nil
}

func (dec *BinaryStringDecoder) Read(p []byte) (int, error) {
  return dec.pipeReader.Read(p)
}

var (
  ErrBinaryStringBadLen = errors.New("Binary string length must be a multiple of 8")
  ErrBinaryStringBadChar = errors.New("Binary string must be composed of '0' and/or '1' only")
)

func (dec *BinaryStringDecoder) Write(data []byte) (int, error) {
  if len(data) % 8 != 0 {
    return 0, ErrBinaryStringBadLen
  }

  buff := make([]byte, len(data) / 8 )

  k := 0
  for i := 0; i < len(buff); i++ {
    for j := 0; j < 8; j++ {
      if data[k] != 0x30 && data[k] != 0x31 {
        return 0, ErrBinaryStringBadChar
      }

      if data[k] == 0x30 {
        buff[i] = buff[i] << 1 | 0
      } else {
        buff[i] = buff[i] << 1 | 1
      }

      k++
    }
  }

  i, err := dec.pipeWriter.Write(buff)
  return i * 8, err
}

func (dec *BinaryStringDecoder) Close() (error) {
  return dec.pipeWriter.Close()
}

func (enc *BinaryStringEncoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc BinaryStringEncoder) Init() (error) {
  return nil
}

func (enc *BinaryStringEncoder) Write(data []byte) (int, error) {
  for i, b := range data {
    buff := make([]byte, 8)

    k := 0
    for j := 7; j > -1; j-- {
      if b >> uint8(j) & 1 == 1 {
        buff[k] = 0x31
      } else {
        buff[k] = 0x30
      }

      k++
    }

    _, err := enc.pipeWriter.Write(buff)
   if err != nil {
     return i, err
   }
  }

  return len(data), nil
}

func (enc *BinaryStringEncoder) Close() (error) {
  return enc.pipeWriter.Close()
}
