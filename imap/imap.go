/*
			Copyright © 2023 Manuel Bovo <manuel.bovo@gmail.com>

	    This program is free software: you can redistribute it and/or modify
	    it under the terms of the GNU General Public License as published by
	    the Free Software Foundation, either version 3 of the License, or
	    (at your option) any later version.

	    This program is distributed in the hope that it will be useful,
	    but WITHOUT ANY WARRANTY; without even the implied warranty of
	    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	    GNU General Public License for more details.

	    You should have received a copy of the GNU General Public License
	    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package imap

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/mbovo/imapindexer/types"
)

type ImapConfig struct {
	Address        string
	Username       string
	Password       string
	MailBoxPattern string
}

func newClient(config ImapConfig) (*imapclient.Client, error) {
	c, err := imapclient.DialTLS(config.Address, nil)
	if err != nil {
		return nil, err
	}

	// Login
	logincmd := c.Login(config.Username, config.Password)
	if e := logincmd.Wait(); e != nil {
		return nil, errors.Join(e, errors.New("cannot login to IMAP server"))
	}
	return c, err
}

func GetMails(messages chan *types.Message, wg *sync.WaitGroup, config ImapConfig) {
	defer wg.Done()

	c, err := newClient(config)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create IMAP clinet")
	}
	defer c.Logout()
	defer c.Close()

	// list mailboxes with given pattern and get messages count
	listCmd := c.List("", config.MailBoxPattern, &imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
			UIDNext:     true,
		}})
	defer listCmd.Close()

	mboxes, err := listCmd.Collect()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot get mailboxes")
	}

	log.Info().Int("mailboxes", len(mboxes)).Msg("Mailboxes found")
	workerCount := viper.GetInt("indexer.workers")
	for {
		lwg := &sync.WaitGroup{}
		// ensure we don't start more workers than remaining mailboxes
		if len(mboxes) < workerCount {
			workerCount = len(mboxes)
			log.Info().Int("workers", workerCount).Int("mailboxes", len(mboxes)).Msg("Reducing workers to match mailboxes")
		}
		// start workers
		for i := 0; i < workerCount && i != len(mboxes); i++ {
			lwg.Add(1)
			log.Debug().Int("i", i).Int("mboxes", len(mboxes)).Msg("Starting worker")
			go imapWorker(config, mboxes[i], messages, lwg)
		}
		// resize mboxes slice
		mboxes = mboxes[workerCount:]
		if len(mboxes) == 0 {
			break
		}
		lwg.Wait()
	}
	close(messages)
}

func imapWorker(config ImapConfig, mbox *imap.ListData, messages chan *types.Message, wg *sync.WaitGroup) {
	defer wg.Done()

	c, err := newClient(config)
	if err != nil {
		log.Error().Err(err).Msg("cannot create IMAP client")
		return
	}
	defer c.Logout()
	defer c.Close()
	log.Info().Str("mailbox", mbox.Mailbox).Msg("Parsing Mailbox")

	seqSet := &imap.SeqSet{}
	seqSet.AddRange(1, *mbox.Status.NumMessages)

	fetchOptions := &imap.FetchOptions{
		UID:         true,
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{{}}, //TODO: do we want to filter the headers and body type?
	}

	c.Select(mbox.Mailbox, nil)
	fetchCmd := c.Fetch(*seqSet, fetchOptions)
	defer fetchCmd.Close()
	count := 0
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		count += 1
		message := &types.Message{}
		for {
			iteam := msg.Next()
			if iteam == nil {
				break
			}
			switch item := iteam.(type) {
			case imapclient.FetchItemDataBodySection:
				b, err := io.ReadAll(item.Literal)
				if err != nil {
					log.Error().Err(err).Str("mailbox", mbox.Mailbox).Uint32("uid", msg.SeqNum).Msg("failed to read body section")
					continue
				}
				message.Body = string(b)
			case imapclient.FetchItemDataUID:
				message.UID = item.UID
			case imapclient.FetchItemDataEnvelope:
				message.Envelope = item.Envelope
			}
		}
		if viper.GetBool("imap.useHash") {
			message.Hash = hash(message.Envelope.Subject, message.Body)
		}
		message.MailBox = mbox.Mailbox
		log.Debug().Str("mailbox", mbox.Mailbox).Uint32("uid", message.UID).Str("subject", message.Envelope.Subject).Msg("Got Message")
		messages <- message
	}
	log.Info().Str("mailbox", mbox.Mailbox).Int("tot", int(*mbox.Status.NumMessages)).Int("parsed", count).Msg("Done")
}

func hash(s ...string) string {
	bb := bytes.Buffer{}
	for _, v := range s {
		bb.WriteString(v)
	}
	h := sha256.Sum256(bb.Bytes())
	return fmt.Sprintf("%x", h)
}
