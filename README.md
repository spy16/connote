# 📝 connote

![Build](https://github.com/spy16/connote/actions/workflows/tag_created.yml/badge.svg)

connote is a simple console-based note taking tool.

## Features

* Simple markdown based notes.
* All notes are stored as files in `$HOME/.connote/<profile>`.
* Front-matter is used for tags and other metadata.
* Multiple profiles support for isolating notes.
* All commands support `json`, `yaml`, `pretty` outputs. 

## Install

* Download the binary for your operating system from [Releases](https://github.com/spy16/connote/releases)
* OR, Run `go install github.com/spy16/connote` to directly build and install.

## Usage

```shell
# write down anything specific to the day (e.g., work log)
$ connote edit

# edit yesterday's note
$ connote edit @yday

# create a custom note with tags 
$ connote edit kafka -t tldr

# show today's note
$ connote show

# show last week's note
$ connote show @-7

# show note on kafka
$ connote show kafka

# list all notes (table view)
$ connote ls

# list all notes (yaml format)
$ connote ls -o yaml

# list all tldr type notes
$ connote ls -i tldr
```

* *💡 Tip*: Alias `connote` as `cn` for easy access.
* *📌 Note*: Connote uses the editor command set through `EDITOR` environment variable (The editor must be blocking, like Vim).
