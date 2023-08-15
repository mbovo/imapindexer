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
package cmd

import (
	"context"

	"github.com/mbovo/imapindexer/imap"
	"github.com/mbovo/imapindexer/indexer"
	"github.com/mbovo/imapindexer/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func run() {

	log.Info().Msg("Starting imapindexer")
	log.Info().Fields(viper.AllSettings()).Msg("Configuration")

	messages := make(chan *types.Message, viper.GetInt("imap.queue"))

	zinc, ctx := indexer.NewZinc(context.Background(), messages, indexer.ZincConfig{
		Address:  viper.GetString("zinc.address"),
		Username: viper.GetString("zinc.username"),
		Password: viper.GetString("zinc.password"),
		Index:    viper.GetString("zinc.index"),
	})

	go zinc.IndexMails(ctx) // will wait on messages channel for new data and index when ready

	imap.GetMails(messages, imap.ImapConfig{
		Address:        viper.GetString("imap.address"),
		Username:       viper.GetString("imap.username"),
		Password:       viper.GetString("imap.password"),
		MailBoxPattern: viper.GetString("imap.mailbox"),
	})
}
