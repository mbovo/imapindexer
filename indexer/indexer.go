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
package indexer

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/mbovo/imapindexer/types"
	"github.com/rs/zerolog/log"
	client "github.com/zinclabs/sdk-go-zincsearch"
)

type Zinc struct {
	client *client.APIClient
	config ZincConfig
}

type ZincConfig struct {
	Address   string
	Username  string
	Password  string
	Index     string
	BatchSize int32
}

func NewZinc(ctx context.Context, cfg ZincConfig) (*Zinc, context.Context) {
	newCtx := context.WithValue(ctx, client.ContextBasicAuth, client.BasicAuth{
		UserName: cfg.Username,
		Password: cfg.Password,
	})
	config := client.NewConfiguration()
	config.Servers = client.ServerConfigurations{
		client.ServerConfiguration{
			URL: cfg.Address,
		},
	}

	z := &Zinc{
		client: client.NewAPIClient(config),
		config: cfg,
	}

	return z, newCtx
}

func msgToMap(msg *types.Message) map[string]interface{} {
	document := map[string]interface{}{}

	// convert msg to map[string]interface{}
	m, _ := msg.JSON()
	err := json.Unmarshal(m, &document)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal")
		return nil
	}

	// Force id and timestamp for ZincSearch index
	document["_id"] = msg.Envelope.MessageID
	document["@timestamp"] = msg.Envelope.Date

	return document
}

func (z *Zinc) IndexMails(ctx context.Context, messages chan *types.Message, wg *sync.WaitGroup, bar chan int) {
	var document map[string]interface{}
	batch := []map[string]interface{}{}
	log.Debug().Msg("Waiting for messages to index")
	time.Sleep(time.Second)

	for msg := range messages {
		bar <- 1
		document = msgToMap(msg)
		if document == nil {
			log.Error().Msg("failed to convert message to map")
			continue
		}

		batch = append(batch, document)

		if len(batch) < int(z.config.BatchSize) {
			log.Debug().
				Str("mailbox", msg.MailBox).
				Str("Subject", msg.Envelope.Subject).
				Uint32("uid", msg.UID).
				Int("batch", len(batch)).
				Msg("Enqueue")
			continue
		}

		a := client.NewMetaJSONIngest()
		a.SetIndex(z.config.Index)
		a.SetRecords(batch)
		resp, _, err := z.client.Document.Bulkv2(ctx).Query(*a).Execute()
		// resp, _, err := z.client.Document.IndexWithID(ctx, z.config.Index, msg.Hash).Document(document).Execute()
		if err != nil {
			log.Error().Err(err).Msg("failed to index document")
			continue
		}
		log.Info().
			Int32("count", resp.GetRecordCount()).
			Int("batch", len(batch)).
			Msg("Indexed")
		// zeroing batch
		batch = []map[string]interface{}{}
	}

	if len(batch) > 0 {
		// pushing last remaining items
		a := client.NewMetaJSONIngest()
		a.SetIndex(z.config.Index)
		a.SetRecords(batch)
		resp, _, err := z.client.Document.Bulkv2(ctx).Query(*a).Execute()
		// resp, _, err := z.client.Document.IndexWithID(ctx, z.config.Index, msg.Hash).Document(document).Execute()
		if err != nil {
			log.Error().Err(err).Msg("failed to index document")
		}
		log.Info().
			Int32("count", resp.GetRecordCount()).
			Int("batch", len(batch)).
			Msg("Indexed")
	}
	log.Info().Msg("No more messages")
	wg.Done()
}
