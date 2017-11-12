package main

import (
  "os"
  "./commands/dd"
  "./commands/dgst"
  "./flags"
  "./codec"
  "fmt"
  "io"
  "flag"
)

func main() {
  command := parseCommand()
  globalFlags := flags.SetupFlags(flag.CommandLine)
  command.SetupFlags(flag.CommandLine)
  flag.CommandLine.Usage = func () {
    usage := command.Usage()
    fmt.Fprintf(os.Stderr, "Usage: %s [<Options>] %s\n\nOptions:\n", os.Args[0], usage.CommandLine)
    flag.PrintDefaults()
    if len(codec.CodecList) > 0 {
      fmt.Fprintln(os.Stderr, "Codecs:")
      for _, c := range codec.CodecList {
        fmt.Fprintf(os.Stderr, "  %s\n\t%s\n", c.Name(), c.Description())
      }
    }
    if usage.Other != "" {
      fmt.Fprintf(os.Stderr, "\n%s\n", usage.Other)
    }
  }
  globalOptions := flags.ParseFlags(flag.CommandLine, globalFlags)
  err := command.ParseFlags()
  if err != nil {
    flag.CommandLine.Usage()
    os.Exit(2)
  }

  done := make(chan struct{})
  byteCounterIn := newByteCounter(globalOptions.FromByteIn, globalOptions.ToByteIn)
  byteCounterOut := newByteCounter(globalOptions.FromByteOut, globalOptions.ToByteOut)

  go func() {
    err := globalOptions.Encoders[len(globalOptions.Encoders) - 1].Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err in init encoder: %v", err)
      return
    }
    _, err = io.Copy(os.Stdout, globalOptions.Encoders[len(globalOptions.Encoders) - 1])
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err in reading encoder: %v", err)
      return
    }

    stdoutFileInfo, _ := os.Stdout.Stat()
    if (! (stdoutFileInfo.Mode() & os.ModeCharDevice == 0) && ! globalFlags.Chomp) {
      fmt.Println()
    }

    done <- struct{}{}
  }()

  var encoderReader codec.CodecEncoder
  encoderReader = globalOptions.Encoders[0]

  for _, encoder := range globalOptions.Encoders[1:] {
    go func(encoder codec.CodecEncoder, encoderReader codec.CodecEncoder) {
      err := encoder.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init encoder: %v", err)
        os.Exit(1)
      }

      _, err = io.Copy(encoder, encoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading reader: %v", err)
        os.Exit(1)
      }

      err = encoder.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(encoder, encoderReader)

    encoderReader = encoder
  }

  if globalOptions.FromByteOut != 0 || globalOptions.ToByteOut != 0 {
    go func() {
       _, err := io.Copy(byteCounterOut, command)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in command: %v", err)
        os.Exit(1)
      }

      err = byteCounterOut.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, byteCounterOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in byteCounterOut: %v", err)
        os.Exit(1)
      }

      err = encoderReader.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Encoders[0])
  } else {
    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, command)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in encoderReader: %v", err)
        os.Exit(1)
      }

      err = encoderReader.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Encoders[0])
  }

  var decoderReader codec.CodecDecoder
  decoderReader = globalOptions.Decoders[0]

  for _, decoder := range globalOptions.Decoders[1:] {
    go func(decoder codec.CodecDecoder, decoderReader codec.CodecDecoder) {
      err := decoderReader.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }

      _, err = io.Copy(decoder, decoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading decoder decoderReader: %v", err)
        os.Exit(1)
      }

      err = decoder.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(decoder, decoderReader)

    decoderReader = decoder
  }

  if globalOptions.FromByteIn != 0 || globalOptions.ToByteIn != 0 {
    go func() {
      err := decoderReader.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }
       _, err = io.Copy(byteCounterIn, decoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in decoder: %v", err)
        os.Exit(1)
      }

      err = byteCounterIn.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func() {
       _, err := io.Copy(command, byteCounterIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in byteCounterIn: %v", err)
        os.Exit(1)
      }

      err = command.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  } else {
    go func(decoder codec.CodecDecoder) {
      err := decoder.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }
      _, err = io.Copy(command, decoder)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading decoder decoderReader: %v", err)
        os.Exit(1)
      }

      err = command.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Decoders[len(globalOptions.Decoders) - 1])
  }

  go func() {
    stdinFileInfo, _ := os.Stdin.Stat()
    if (stdinFileInfo.Mode() & os.ModeCharDevice == 0) {
      _, err := io.Copy(globalOptions.Decoders[0], os.Stdin)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error in decoding stdin: %v", err)
        os.Exit(1)
      }
      err = globalOptions.Decoders[0].Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }
  }()

  <- done
}

var CommandList = []Command{
  dd.Command,
  dgst.Command,
}

func UsageCommand() {
  fmt.Fprintln(os.Stderr, "Commands:")
  for _, command := range CommandList {
    fmt.Fprintf(os.Stderr, "  %s:  %s\n", command.Name(), command.Description())
  }

  os.Exit(2)
}

func parseCommand() (CommandPipe) {
  if len(os.Args) == 1 {
    UsageCommand()
  }

  switch command := os.Args[1]; command {
    case dd.Command.Name():
      return dd.Command
    case dgst.Command.Name():
      return dgst.Command
    default:
      fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
      UsageCommand()
  }

  return nil
}
