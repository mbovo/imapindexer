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

	"github.com/mbovo/imapindexer/types"
	"github.com/rs/zerolog/log"
	client "github.com/zinclabs/sdk-go-zincsearch"
)

type Zinc struct {
	client *client.APIClient
	buffer chan *types.Message
	config ZincConfig
}

type ZincConfig struct {
	Address  string
	Username string
	Password string
	Index    string
}

func NewZinc(ctx context.Context, buffer chan *types.Message, cfg ZincConfig) (*Zinc, context.Context) {
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
		buffer: buffer,
		config: cfg,
	}

	return z, newCtx
}

func (z *Zinc) IndexMails(ctx context.Context) {
	document := map[string]interface{}{}

	for msg := range z.buffer {
		m, _ := msg.JSON()

		err := json.Unmarshal(m, &document)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal")
			continue
		}

		// mq := client.NewMetaTermQuery()
		// mq.SetValue(msg.Hash)
		// q := client.NewMetaQuery()
		// q.SetTerm(map[string]client.MetaTermQuery{
		// 	"hash": *mq,
		// })

		// _, hresp, _ := z.client.Search.Search(ctx, z.config.Index).Query(client.MetaZincQuery{
		// 	Fields: []string{"hash"},
		// 	Query:  q,
		// }).Execute()

		// if hresp.StatusCode == http.StatusOK {
		// 	log.Info().Str("hash", msg.Hash).Msg("document already present, skipping")
		// 	continue
		// }

		resp, _, err := z.client.Document.IndexWithID(ctx, z.config.Index, msg.Hash).Document(document).Execute()
		if err != nil {
			log.Error().Err(err).Msg("failed to index document")
			continue
		}
		log.Info().Str("mailbox", msg.MailBox).Str("subject", msg.Envelope.Subject).Str("_id", resp.GetId()).Uint32("uid", msg.UID).Msg("Added")
	}
}

// func Setup_index(ctx context.Context, zincClient *client.APIClient, index string, shards, replicas int32, mappings map[string]any) (*string, error) {

// 	_, hresp, err := zincClient.Index.Exists(ctx, index).Execute()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if hresp.StatusCode == 200 {
// 		// index exist, do nothing
// 		return nil, nil
// 	}
// 	// index does not exist, create it

// 	md := client.MetaIndexSimple{
// 		Name:     &index,
// 		Mappings: mappings,
// 		Settings: &client.MetaIndexSettings{
// 			NumberOfShards:   &shards,
// 			NumberOfReplicas: &replicas,
// 		},
// 	}

// 	resp, hresp, err := zincClient.Index.Create(ctx).Data(md).Execute()
// 	if err != nil {
// 		return nil, err
// 	}
// 	i, ok := resp.GetIndexOk()
// 	if !ok || hresp.StatusCode != 200 {
// 		return nil, fmt.Errorf("failed to create index %s", hresp.Status)
// 	}
// 	return i, nil
// }
