You can install TwinHunter from source or grab a pre-built binary.

## Installing

#### From source

```sh
go install github.com/mrinjamul/twinhunter@latest
```

#### Pre-built binaries

Download the latest release for your platform:

[Download](https://github.com/mrinjamul/twinhunter/releases)

## Usage

Here are a few common ways to use TwinHunter:

```sh
# Scan the current directory
twinhunter

# Scan a folder recursively
twinhunter find /path/to/files -r

# Find and delete duplicates, keeping the oldest copy
twinhunter find /path/to/files -r -d -k oldest -y
```
