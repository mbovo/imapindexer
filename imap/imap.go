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
	"fmt"
	"io"

	"github.com/rs/zerolog/log"

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

func GetMails(messages chan *types.Message, config ImapConfig) {

	c, err := imapclient.DialTLS(config.Address, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to IMAP server")
	}
	defer c.Close()

	// Login
	logincmd := c.Login(config.Username, config.Password)
	if e := logincmd.Wait(); e != nil {
		log.Fatal().Err(e).Msg("cannot login to IMAP server")
	}
	defer c.Logout()

	listCmd := c.List("", config.MailBoxPattern, &imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
		}})
	defer listCmd.Close()

	// Fetch the headers of the 10 most recent messages

	for {

		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		log.Info().Str("mailbox", mbox.Mailbox).
			Uint32("messages", *mbox.Status.NumMessages).
			Uint32("unseen", *mbox.Status.NumUnseen).
			Msg("Parsing Mailbox")
		seqSet := &imap.SeqSet{}
		seqSet.AddRange(1, *mbox.Status.NumMessages)
		fetchOptions := &imap.FetchOptions{
			UID:         true,
			Envelope:    true,
			BodySection: []*imap.FetchItemBodySection{{}},
		}
		c.Select(mbox.Mailbox, nil)
		fetchCmd := c.Fetch(*seqSet, fetchOptions)
		defer fetchCmd.Close()
		for {
			msg := fetchCmd.Next()
			if msg == nil {
				break
			}
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
						log.Error().Err(err).Msg("failed to read body section")
						continue
					}
					message.Body = string(b)
				//	log.Printf("Body %v", string(b))
				case imapclient.FetchItemDataUID:
					message.UID = item.UID
				//	log.Printf("UID %v", item.UID)
				case imapclient.FetchItemDataEnvelope:
					message.Envelope = item.Envelope
					//	log.Printf("%v : %v", item.Envelope.Date, item.Envelope.Subject)
				}
			}
			bb := bytes.Buffer{}
			bb.WriteString(mbox.Mailbox)
			bb.WriteString(fmt.Sprintf("%d", message.UID))
			bb.WriteString(message.Envelope.Subject)
			h := sha256.Sum256(bb.Bytes())
			message.Hash = fmt.Sprintf("%x", h)
			message.MailBox = mbox.Mailbox
			messages <- message
		}
	}
}
