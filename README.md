# 📝 connote

connote is a simple console-based note taking tool.

## Install

* Download the binary for your operating system from [Releases](https://github.com/spy16/connote/releases)
* Run `go install github.com/spy16/connote` to directly build and install.

## Usage

```shell
# write down anything specific to the day (e.g., work log)
$ connote write

# edit yesterday's note
$ connote note @yday

# create a custom note with tags 
$ connote note kafka -t type:tldr

# show today's note
$ connote show

# show last week's note
$ connote show @-7

# show note on kafka
$ connote show kafka

# list all notes
$ connote list

# list all tldr type notes
$ connote list -t type:tldr
```

