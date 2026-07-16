package api

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"time"
)

// The sitemap spec caps a single file at 50k URLs; the retention window holds
// ~14.4k blocks, so one file is enough with headroom.
const sitemapMaxURLs = 45000

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// GET /sitemap-blocks.xml — rolling sitemap of the retained block window.
// Blocks are immutable, so lastmod is the block timestamp and there is no
// changefreq/priority.
func (s *Server) handleBlocksSitemap(w http.ResponseWriter, r *http.Request) {
	entries, err := s.st.BlockSitemapEntries(sitemapMaxURLs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	set := urlset{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	set.URLs = make([]sitemapURL, len(entries))
	for i, e := range entries {
		set.URLs[i] = sitemapURL{
			Loc:     s.siteURL + "/block/" + strconv.FormatUint(e.Number, 10),
			LastMod: e.Timestamp.UTC().Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = io.WriteString(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(&set)
}
