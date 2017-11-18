package inout

import (
  "io"
  "net/url"
  "github.com/pkg/errors"
)

var (
  InoutList []IO = []IO{
    DefaultFile,
    DefaultPipe,
  }
)

type IO interface {
  In(*url.URL) (Input)
  Out(*url.URL) (Output)
  Description() (string)
  Name() (string)
}

type Input interface {
  io.ReadCloser
  Init() (error)
  Name() (string)
}

type Output interface {
  io.WriteCloser
  Init(chomp bool) (error)
  Name() (string)
}

func ParseOutput(out string) (Output, error) {
  uri, output, err := parse(out)
  if err != nil {
    return nil, err
  }

  return output.Out(uri), nil
}

func ParseInput(in string) (Input, error) {
  uri, input, err := parse(in)
  if err != nil {
    return nil, err
  }

  return input.In(uri), nil
}

func parse(inout string) (*url.URL, IO, error) {
  if inout == "" {
    return nil, DefaultStd, nil
  }

  uri, err := url.Parse(inout)
  if err != nil {
    return nil, nil, errors.Wrapf(err, "Err in decoding uri")
  }

  switch uri.Scheme {
    case "file":
      return uri, DefaultFile, nil
    case "":
      return uri, DefaultFile, nil
    case "pipe":
      return uri, DefaultPipe, nil
    default:
      return nil, nil, errors.Errorf("Error unknown type %s\n", uri.Scheme)
  }

  return nil, nil, errors.New("Unhandled error 101\n")
}