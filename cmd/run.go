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
	"fmt"
	"os"
	"sync"

	"github.com/gosuri/uiprogress"
	"github.com/mbovo/imapindexer/imap"
	"github.com/mbovo/imapindexer/indexer"
	"github.com/mbovo/imapindexer/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func run() {

	log.Info().Msg("Starting imapindexer")

	if viper.GetBool("debug") {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger().Level(zerolog.DebugLevel)
	}

	l, err := zerolog.ParseLevel(viper.GetString("loglevel"))
	if err == nil {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Logger().Level(l)
	}

	if viper.GetBool("progress") {
		log.Info().Msg("Using Progress bars, disabling logging")
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Caller().Logger().Level(zerolog.Disabled)
	}

	log.Debug().Msg("Debug mode enabled")
	log.Info().Fields(viper.AllSettings()["indexer"]).Msg("Indexer settings")

	messages := make(chan *types.Message, viper.GetInt("imap.buffer"))
	barChan := make(chan int, 1)

	zinc, ctx := indexer.NewZinc(context.Background(), indexer.ZincConfig{
		Address:   viper.GetString("zinc.address"),
		Username:  viper.GetString("zinc.username"),
		Password:  viper.GetString("zinc.password"),
		Index:     viper.GetString("zinc.index"),
		BatchSize: int32(viper.GetInt("indexer.batch")),
	})
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go zinc.IndexMails(ctx, messages, wg, barChan) // will wait on messages channel for new data and index when ready

	go indexBar(ctx, barChan)

	go imap.GetMails(messages, wg, imap.ImapConfig{
		Address:        viper.GetString("imap.address"),
		Username:       viper.GetString("imap.username"),
		Password:       viper.GetString("imap.password"),
		MailBoxPattern: viper.GetString("imap.mailbox"),
	}, barChan)

	wg.Wait()
}

func indexBar(ctx context.Context, barChan chan int) {
	bar := uiprogress.AddBar(1).PrependElapsed().AppendCompleted()
	//	p := uiprogress.Progress{}
	//	bar := p.AddBar(1).PrependElapsed().AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Indexing %d/%d", b.Current(), b.Total)
	})
	tot := 1
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-barChan:
			if s > 1 {
				tot += s
				old := bar.Current()
				bar.Total = tot
				bar.Set(old)
			}
			bar.Incr()
		}
	}
}
