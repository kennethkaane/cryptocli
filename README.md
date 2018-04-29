# cryptocli
A modern tool to replace dd and openssl cli using a pipeline type data flow to move bytes around.

You can read from many sources, write to many sources, decode, encode, stop a byte X, read from byte X, redirect data to many sources from different point on the pipeline and perform some command.

It'll be your next swiss army knife for sure!

Read the Usage for more explanations.

## Motivation
I use decoding/encoding tools, dd and openssl all the time. It was getting a little bit annoying to have to use shell tricks to get what I wanted.

Pull requests are of course welcome.

## How to install
There are two ways you could install `cryptocli`:

  - By going to the [release page](https://github.com/tehmoon/cryptocli/releases)
  - By compiling it yourself:
    - Download and install [go](https://golang.org)
    - `git clone https://github.com/tehmoon/cryptocli`
    - Set `GOPATH`: `mkdir ~/work/go && export GOPATH=~/work/go`
    - `cd cryptocli`
    - Get dependencies `go get ./...`
    - Generate binary `go build`
    - Optional: Move the binary to `${GOPATH}/bin`: `go install`

## Commands:

```
Commands:
  dd:  Copy input to output like the dd tool.
  dgst:  Hash from input
  pbkdf2:  Derive a key from input using the PBKDF2 algorithm
  scrypt:  Derive a key from input using the scrypt algorithm
  pipe:  Execute a command and attach stdin and stdout to the pipeline
  get-certs:  Establish tls connection and get the certificates. Doesn't use any input.
  aes-gcm-encrypt:  Encrypt and authenticate 16KiB blocks of data using AES algorithm with GCM mode. Padding is not necessary so if EOF is reached, it will return less. Nonce are 8 random bytes followed byte 4 bytes which starts by 0 and are incremented. So it outputs the following <-salt-length> || nonce[:8] || (tag || encrypted data)... . To Decrypt you must read <-salt-length> if you need to derive the key, then reconstruct the nonce by taking the following 8 bytes and appending 0x00000000, then the following 16KiB can be decrypted with GCM. For each 16KiB until EOF -- no padding -- don't forget to reuse the first 8 bytes of the nonce and increment the last 4 bytes ONLY.
  aes-gcm-decrypt:  Decrypt and verify authentication of 16KiB block of data using AES algorithm with GCM mode. Nonce of 8 bytes is read after reading the salt to derived the key. Then we append 4 bytes number to the nonce every block starting at 0. Only the first 8 bytes of the nonce is reused. By default it uses scrypt to derive the key but if you want to use your own KDF, aes-gcm-decrypt  will read the salt up to -salt-length then set the environment variable SALT to the hex salt value so you can execute your KDF using the pipe: inout module. If you do that, the salt is expected to be found prepended to the key.
```

## Usage

```
cryptocli <command> [<options>] [<arguments>]
```

```
Usage: ./cryptocli [<Options>]

Options:
  -chomp
        Get rid of the last \n when not in pipe
  -decoders string
        Set a list of codecs separated by ',' to decode input that will be process in the order given (default "binary")
  -encoders string
        Set a list of codecs separated by ',' to encode output that will be process in the order given (default "binary")
  -filters-cmd-in string
        List of <filter> in URL format that filters data right after -decoders
  -filters-cmd-out string
        List of <filter> in URL format that filters data right before -encoders
  -filters-in string
        List of <filter> in URL format that filters data right before -decoders
  -filters-out string
        List of <filter> in URL format that filters data right after -encoders
  -in string
        Input <fileType> method
  -out string
        Output <fileType> method
  -tee-cmd-in string
        Copy output after -decoders and before <command> to <fileType>
  -tee-cmd-out string
        Copy output after <command> and before -encoders to <fileType>
  -tee-in string
        Copy output before -encoders to <fileType>
  -tee-out string
        Copy output after -encoders to <fileType>

Codecs:
  hex
        hex encode output and hex decode input
  binary
        Do nothing in input and nothing in output
  binary-string
        Take ascii string of 1 and 0 in input and decode it to binary. A byte is always 8 characters number. Does the opposite for output
  base64
        base64 decode input and base64 encode output
  gzip
        gzip compress output and gzip decompress input
  hexdump
        Encode output to hexdump -c. Doesn't support decoding
  byte-string
        Decode and encode in a byte string format

FileTypes:
  file://
        Read from a file or write to a file. Default when no <filetype> is specified. Truncate output file unless OUTFILENOTRUNC=1 in environment variable.
  pipe:
        Run a command in a sub shell. Either write to the command's stdin or read from its stdout.
  https://
        Get https url or post the output to https. Use INHTTPSNOVERIFY=1 and/or OUTHTTPSNOVERIFY=1 environment variables to disable certificate check. Max redirects count is 3. Will fail if scheme changes.
  http://
        Get http url or post the output to https. Max redirects count is 3. Will fail if scheme changes.
  env:
        Read and unset environment variable. Doesn't work for output
  readline:
        Read lines from stdin until WORD is reached.
  s3://
        Either upload or download from s3.
  null:
        Behaves like /dev/null on *nix system
  hex:
        Decode hex value and use it for input. Doesn't work for output
  ascii:
        Decode ascii value and use it for input. Doesn't work for output
  rand:
        Rand is a global, shared instance of a cryptographically strong pseudo-random generator. On Linux, Rand uses getrandom(2) if available, /dev/urandom otherwise. On OpenBSD, Rand uses getentropy(2). On other Unix-like systems, Rand reads from /dev/urandom. On Windows systems, Rand uses the CryptGenRandom API. Doesn't work with output.
  password:
        Reads a line of input from a terminal without local echo
  math:
        Evaluate an expression using robpike.io/ivy. Doesn't support output.

Filters:
  pem
        Filter PEM objects. Options: type=<PEM type> start-at=<number> stop-at=<number>. Type will filter only PEM objects with this type. Start-at will discard the first <number> PEM objects. Stop-at will stop at PEM object <number>.
  byte-counter
        Keep track of in and out bytes. Options: start-at=<number> stop-at=[+]<number>. Start-at option will discard the first <number> bytes. The Stop-at option will stop at byte <number>. Position <number> can be express in base16 with 0x/0X, base2 with 0b/0B or 0 for base8. If a + sign is found in the stop-at option; start-at <number> is added to stop-at <number>.
```

## Examples

### Passing data and transforming

Get the last 32 byte of a sha512 hash function from a hex string to base64 without last \n

```
echo -n 'DEADBEEF' | ./cryptocli dgst -decoders hex -encoders base64 -filters-cmd-out byte-counter:start-at=32 -chomp sha512
```

Transform stdin to binary string

```
echo -n toto | cryptocli dd -encoders binary-string
```

Gzip stdin then base64 it

```
echo -n toto | cryptocli dd -encoders gzip,base64
```

Get rid of the first 2 bytes

```
echo -n toto | cryptocli dd -filters-in byte-counter:start-at=2 2
```

Decode base64 from file to stdout in hex

```
cryptocli dd -decoders base64 -encoders hex -in ./toto.txt
```

Gzip input, write it to file and write its sha512 checksum in hex format to another file

```
echo toto | cryptocli dd -encoders gzip -tee-out pipe:"cryptocli dgst -encoders hex -out ./checksum.txt" -out ./file.gz
```

Execute nc -l 12344 which opens a tcp server and base64 the output

```
cryptocli pipe -encoders base64 nc -l 12344
```

Download an s3 object in a streaming fashion then gunzip it

```
cryptocli dd -in s3://bucket/path/to/key -decoders gzip -out key
```

Upload an s3 object, gzip it and write checksum

```
cryptocli dd -in file -encoders gzip -tee-out pipe:"cryptocli dgst -encoders hex -out file.checksum" -out s3://bucket/path/to/file
```

Filter only PEM objects of type certificate

```
cryptocli pipe -filters-cmd-out pem:type=certificate openssl s_client -connect google.com:443
```

Get first cert from tls connection

```
cryptocli get-certs google.com:443
```

Generate random 32 bytes strings using crypto/rand lib

```
cryptocli dd -in rand: -filters-in byte-counter:stop-at=32 -encoders hex
```

### Hashing/Key Derivation Function

Verify a checksum

```
echo toto | cryptocli dgst -checksum-in hex:9a266fc8b42966fb624d852bafa241d8fd05b47d36153ff6684ab344bd1ae57bba96a7de8fc12ec0bb016583735d7f5bca6dd7d9bc6482c2a3ac6bf6f9ec323f
```

Output the base64 hash of stdin to file

```
echo -n toto | cryptocli dgst -encoders base64 -out file://./toto.txt sha512
```

SHA512 an https web page then POST the result to http server:

```
cryptocli dgst -in https://www.google.com -encoders hex sha512 -out http://localhost:12345/
```

Generate 32 byte salt and derive a 32 bytes key from input to `derived-key.txt` file.

```
echo -n toto | cryptocli pbkdf2 -encoders base64 -out derived-key.txt
```

You should have the same result as in `derived-key.txt` file

```
echo -n toto | cryptocli pbkdf2 -salt-in pipe:"cryptocli dd -in derived-key.txt -decoders base64 -filters-out byte-counter:stop-at=32" -encoders base64
```

Read key from env then scrypt it

```
key=blah cryptocli scrypt -in env:key -encoders base64
```

Hash lines read from stdin

```
cryptocli dgst -in readline:WORD -encoders hex
```

Set salt in pbkdf2/scrypt from hex or from ascii

```
cryptocli pbkdf2 -salt-in hex:deadbeef -encoders hex
cryptocli scrypt -salt-in ascii:deadbeef -encoders hex
```

### Encryption/Decryption

Encrypt using AES in GCM mode

```
# Read a password from keyboard then encrypt a file using default KDF. Write the output to file
./cryptocli aes-gcm-encrypt -in ascii:blah -password-in password: -out enc

# Read a password from keyboard then encrypt a file using custom KDF. Write the output to file
# WARNING: USING OWN KDF IS FOR PEOPLE WHO KNOW WHAT THEY ARE DOING. MIGHT RESULT IN WEAK KEY
#        : THIS EXAMPLE USES A WEAKER SCRYPT THAN DEFAULT BUT STILL OK TO USE.
read -s password; password=${password} ./cryptocli aes-gcm-encrypt -in ascii:blah -derived-salt-key-in pipe:"cryptocli scrypt -rounds $((1<<16)) -in ascii:\${password} -key-length 32" -out enc

# Encrypt a file using provided key and without salt
# WARNING: USING YOUR OWN KEY SHOULD BE FOR PEOPLE WHO KNOW WHAT THEY ARE DOING. DONT USE EXAMPLE'S KEY. USE PROPER KDF FOR WEAKER PASSWORD. PASSWORD != KEY
./cryptocli aes-gcm-encrypt -in ascii:blah -derived-salt-key-in hex:0000000000000000000000000000000000000000000000000000000000000000 -salt-length 0 -out enc3
```

Decrypt using AES in GCM mode from examples above

```
# Read a password from keyboard then decrypt a file using default KDF. Read the input from file
./cryptocli aes-gcm-decrypt -in enc -password-in password:

# Read a password from keyboard then decrypt a file using custom KDF. Read the input from file
read -s password; password=${password} ./cryptocli aes-gcm-decrypt -in enc -derived-salt-key-in pipe:"cryptocli scrypt -rounds $((1<<16)) -in env:\${password} -salt-in env:SALT"

# Decrypt a file using provided key and without salt
./cryptocli aes-gcm-decrypt -in enc -derived-salt-key-in hex:0000000000000000000000000000000000000000000000000000000000000000 -salt-length 0
```

## Internal data flow

Input -> filters-in -> tee input -> decoders -> filters-cmd-in -> tee command input -> command -> filters-cmd-out -> tee command output -> encoders -> filters-out -> tee output -> output

## Futur

  - redo the README.md file
  - http/https/ssl-strip proxy
  - http/https/ws/wss servers
  - tcp/tls server
  - cleanup the code
  - code coverage
  - unit tests
  - go tool suite
  - options:
    - interval
    - timeout
    - loop
    - err `redirect error`
    - debug `redirect debug`
    - delimiter pipe `from input reset pipes everytime it hits the delimiter`
  - file types:
    - tls://\<addr>
    - tcp://\<addr> `read/write to tcp connection`
    - socket://\<path> `read/write to socket file`
    - ws://\<path> `read/write to http websocket`
    - wss://\<path> `read/write to htts websocket`
    - fifo://\<path> `read/write to fifo file on filesystem`
    - scp://\<path> `copy from/to sshv2 server`
    - kafka://\<host>/\<topic> `receive/send message to kafka`
  - commands
    - nacl
    - ec
    - hmac
  - codecs
    - pem
    - delete-chars:`characters`
    - base58
    - decimal
    - uint
    - octal
