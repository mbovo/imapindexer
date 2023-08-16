# IMAP Indexer

A simple cli tool to index IMAP mailboxes into a ZincSearch / Elasticsearch.

[![asciicast](https://asciinema.org/a/T09JMxz2qxDqMU6oh73QGBx0c.svg)](https://asciinema.org/a/T09JMxz2qxDqMU6oh73QGBx0c)

## Usage

```
$ imapindexer --help

imapindexer  Copyright (C) 2023  Manuel Bovo
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
See <https://www.gnu.org/licenses/>.

Index all emails messages of your IMAP mailboxes and store
them into a ZincSearch/Elasticsearch to made them fully searchable.

Usage:
  imapindexer [flags]

Flags:
  -c, --config string          config file (default is $HOME/.imapindexer.yaml)
  -d, --debug                  Enable debug mode
  -h, --help                   help for imapindexer
      --imap.address string    IMAP server address
      --imap.mailbox string    IMAP mailbox pattern (default "INBOX")
      --imap.password string   IMAP password
      --imap.username string   IMAP username
      --indexer.batch int      Number of message to send to ZincSearch in a single batch (default 100)
      --indexer.buffer int     Size of buffer for messages channel (default 100)
      --indexer.workers int    Number of imap workers to use (default 1)
      --zinc.address string    ZincSearch server address
      --zinc.index string      ZincSearch index name (default "mail_index")
      --zinc.password string   ZincSearch password
      --zinc.username string   ZincSearch username
```

## Configuration

The configuration file is a YAML file with the following structure:
Default configuration file is `$HOME/.imapindexer.yaml` or you can specify a custom one with the `--config` flag.

```yaml
imap:
  address: imap.server.tld:993
  username: user@server.tld
  password: imap_password
  mailbox: INBOX    # IMAP mailbox pattern (eg: INBOX, INBOX/Spam, INBOX/Spam/* etc)
zinc:
  password: Complexpass#123
  username: admin
  address: http://localhost:4080
  index: mail_index   # ZincSearch index name
indexer:
  batch: 50     # number of message to send to ZincSearch in a single batch
  buffer: 100   # size of buffer for messages channel
  workers: 10   # number of imap threads to use
```

## Build

It uses [Task](https://taskfile.dev) and [GoReleaser](https://goreleaser.com) to build the binary.

```bash
$ task build
```

## License

This project is licensed under the terms of the GPLv3 license.

## Development

You will need a ZincSearch instance running on your machine to run the tests.
You can setup local development using `task setup`

### Dependencies

- [Task](https://taskfile.dev)
- [GoReleaser](https://goreleaser.com)
- [Go](https://golang.org)
- [pre-commit](https://pre-commit.com)
