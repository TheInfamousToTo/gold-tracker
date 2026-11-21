package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/TheInfamousToTo/gold-tracker/backend/internal/model"
)

// discordLimit leaves headroom under Discord's 2000-character cap for
// the code-fence markers added around each table chunk.
const discordLimit = 1900

func money(v *float64, signed bool) string {
	if v == nil {
		return "—"
	}
	if signed {
		return fmt.Sprintf("%+.2f", *v)
	}
	return fmt.Sprintf("%.2f", *v)
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}

// Table renders the per-item table as fixed-width text for a code block.
// Prices are BHD per gram; gains are BHD.
func Table(d model.ReportData) string {
	var b strings.Builder
	row := func(cols ...string) {
		fmt.Fprintf(&b, "%-14s %7s %8s %8s %9s %9s\n", cols[0], cols[1], cols[2], cols[3], cols[4], cols[5])
	}
	row("Item", "Grams", "Buy/g", "Now/g", "Gain", "Month")
	b.WriteString(strings.Repeat("-", 60) + "\n")
	for _, it := range d.Items {
		buy := it.PaidPerGram
		row(clip(it.Name+" "+it.Purity, 14), fmt.Sprintf("%.2f", it.WeightGrams), money(&buy, false),
			money(it.NowPerGram, false), money(it.Gain, true), money(it.MonthGain, true))
	}
	b.WriteString(strings.Repeat("-", 60) + "\n")
	t := d.Totals
	row("TOTAL", "", "", "", money(&t.Gain, true), money(&t.MonthGain, true))
	return b.String()
}

// Messages builds the Discord posts: a headline, the table, and the
// analysis, each split to fit one message.
func Messages(rep model.MonthlyReport) []string {
	d := rep.Data
	t := d.Totals
	head := fmt.Sprintf("**Monthly report — %s**\nInvested %.2f BHD · Value %.2f BHD · Gain %+.2f BHD (%+.2f%%) · This month %+.2f BHD",
		d.Month, t.Paid, t.Value, t.Gain, t.GainPct, t.MonthGain)
	for _, m := range d.Metals {
		head += fmt.Sprintf("\n%s: %.3f → %.3f BHD/g (%+.2f%%), today %.3f", strings.ToUpper(m.Metal[:1])+m.Metal[1:], m.Open, m.Close, m.ChangePct, m.Latest)
	}

	msgs := []string{head}
	for _, chunk := range split(Table(d), discordLimit) {
		msgs = append(msgs, "```\n"+chunk+"```")
	}
	switch {
	case rep.Analysis != nil && *rep.Analysis != "":
		msgs = append(msgs, split(*rep.Analysis, discordLimit)...)
	case rep.Error != nil:
		msgs = append(msgs, "_AI analysis unavailable: "+clip(*rep.Error, 300)+"_")
	}
	return msgs
}

// split breaks text into pieces of at most n bytes, on line boundaries
// where possible.
func split(text string, n int) []string {
	var out []string
	var cur strings.Builder
	for _, line := range strings.SplitAfter(text, "\n") {
		for len(line) > n { // a single overlong line
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			cut := n
			for cut > 0 && !utf8.RuneStart(line[cut]) {
				cut--
			}
			out = append(out, line[:cut])
			line = line[cut:]
		}
		if cur.Len()+len(line) > n {
			out = append(out, cur.String())
			cur.Reset()
		}
		cur.WriteString(line)
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

// Send posts each message to a Discord webhook in order.
func Send(ctx context.Context, client *http.Client, webhook string, msgs []string) error {
	for _, m := range msgs {
		body, _ := json.Marshal(map[string]any{
			"content":          m,
			"allowed_mentions": map[string]any{"parse": []string{}},
			"flags":            4, // SUPPRESS_EMBEDS: no link previews of model-written URLs
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
		if err != nil {
			return errors.New("DISCORD_WEBHOOK_URL is not a valid URL")
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			// *url.Error embeds the webhook URL, which carries its token.
			var ue *url.Error
			if errors.As(err, &ue) {
				err = ue.Err
			}
			return fmt.Errorf("webhook request failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("discord webhook returned %s", resp.Status)
		}
		time.Sleep(500 * time.Millisecond) // stay well under the webhook rate limit
	}
	return nil
}
