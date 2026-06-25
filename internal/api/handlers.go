package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/fipso/blazed-explorer/internal/models"
	"github.com/fipso/blazed-explorer/internal/wire"
)

const maxPageLimit = 100

var (
	reTxHash  = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	reAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	reBlock   = regexp.MustCompile(`^[0-9]+$`)
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func parseLimit(r *http.Request, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return def
	}
	return min(n, maxPageLimit)
}

func parseBefore(r *http.Request) time.Time {
	ms, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	if err != nil || ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// GET /api/txs?status=pending&limit=50&before=<unix-ms>
func (s *Server) handleTxs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = models.StatusPending
	}
	limit := parseLimit(r, 50)
	txs, err := s.st.RecentTxs(status, limit, parseBefore(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lites := make([]wire.TxLite, len(txs))
	for i := range txs {
		lites[i] = wire.TxLiteFromModel(&txs[i])
	}
	var nextCursor *int64
	if len(txs) == limit {
		c := txs[len(txs)-1].FirstSeen.UnixMilli()
		nextCursor = &c
	}
	writeJSON(w, http.StatusOK, map[string]any{"txs": lites, "nextCursor": nextCursor})
}

// GET /api/txs/{hash}
func (s *Server) handleTxDetail(w http.ResponseWriter, r *http.Request) {
	hash := strings.ToLower(r.PathValue("hash"))
	if !reTxHash.MatchString(hash) {
		writeErr(w, http.StatusBadRequest, "invalid tx hash")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	tx, err := s.st.TxByHash(hash)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Not something we ingested — ask the node so any hash works.
		tx, _, err = s.feed.FetchTx(ctx, hash)
		if err != nil || tx == nil {
			writeErr(w, http.StatusNotFound, "transaction not found")
			return
		}
	}

	// Calldata isn't stored; fetch it live alongside the receipt.
	var input string
	var receipt json.RawMessage
	if _, liveInput, err := s.feed.FetchTx(ctx, hash); err == nil {
		input = liveInput
	}
	if tx.Status == models.StatusMined {
		receipt, _ = s.feed.FetchReceipt(ctx, hash)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tx": tx, "input": input, "receipt": receipt})
}

// GET /api/address/{addr}?limit&before
func (s *Server) handleAddress(w http.ResponseWriter, r *http.Request) {
	addr := strings.ToLower(r.PathValue("addr"))
	if !reAddress.MatchString(addr) {
		writeErr(w, http.StatusBadRequest, "invalid address")
		return
	}
	limit := parseLimit(r, 50)
	txs, err := s.st.TxsByAddress(addr, limit, parseBefore(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lites := make([]wire.TxLite, len(txs))
	for i := range txs {
		lites[i] = wire.TxLiteFromModel(&txs[i])
	}
	var nextCursor *int64
	if len(txs) == limit {
		c := txs[len(txs)-1].FirstSeen.UnixMilli()
		nextCursor = &c
	}
	writeJSON(w, http.StatusOK, map[string]any{"address": addr, "txs": lites, "nextCursor": nextCursor})
}

// GET /api/address/{addr}/code
// Detects whether addr is a contract and returns its runtime bytecode (via
// eth_getCode). Creation info is filled in only when we ingested the deploy.
func (s *Server) handleAddressCode(w http.ResponseWriter, r *http.Request) {
	addr := strings.ToLower(r.PathValue("addr"))
	if !reAddress.MatchString(addr) {
		writeErr(w, http.StatusBadRequest, "invalid address")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	code, err := s.feed.FetchCode(ctx, addr)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	isContract := len(code) > 2 // more than just "0x"

	// EIP-7702 (Pectra): a delegated EOA reports code as the designator
	// 0xef0100 || <20-byte address>. It's not a deployed contract, so flag it
	// separately and point at the contract it delegates execution to.
	var delegatedTo any
	if strings.HasPrefix(code, "0xef0100") && len(code) == len("0xef0100")+40 {
		delegatedTo = "0x" + code[len("0xef0100"):]
		isContract = false
	}

	var creation any
	if isContract {
		if tx, err := s.st.TxByContractAddress(addr); err == nil {
			creation = map[string]any{
				"txHash":      tx.Hash,
				"deployer":    tx.Sender,
				"blockNumber": tx.BlockNumber,
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"address":     addr,
		"isContract":  isContract,
		"delegatedTo": delegatedTo,
		"bytecode":    code,
		"codeSize":    (len(code) - 2) / 2,
		"creation":    creation,
	})
}

// GET /api/blocks?limit&before=<number>
func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 25)
	var before uint64
	if v := r.URL.Query().Get("before"); v != "" {
		before, _ = strconv.ParseUint(v, 10, 64)
	}
	blocks, err := s.st.Blocks(limit, before)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	msgs := make([]wire.BlockMsg, len(blocks))
	for i := range blocks {
		msgs[i] = wire.BlockMsgFromModel(&blocks[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": msgs})
}

// GET /api/blocks/{number}
func (s *Server) handleBlockDetail(w http.ResponseWriter, r *http.Request) {
	num, err := strconv.ParseUint(r.PathValue("number"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid block number")
		return
	}
	block, err := s.st.BlockByNumber(num)
	if err != nil {
		writeErr(w, http.StatusNotFound, "block not found")
		return
	}
	txs, err := s.st.BlockTxs(num)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	lites := make([]wire.TxLite, len(txs))
	for i := range txs {
		lites[i] = wire.TxLiteFromModel(&txs[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"block": wire.BlockMsgFromModel(block), "txs": lites})
}

// GET /api/gas
func (s *Server) handleGas(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.feed.GasNow())
}

// GET /api/stats/history?window=1h|6h|24h|7d
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	window := time.Hour
	switch r.URL.Query().Get("window") {
	case "", "1h":
	case "6h":
		window = 6 * time.Hour
	case "24h":
		window = 24 * time.Hour
	case "7d":
		window = 7 * 24 * time.Hour
	default:
		writeErr(w, http.StatusBadRequest, "invalid window")
		return
	}
	const maxPoints = 500
	// Downsampling happens in SQL; this query returns ~maxPoints rows already.
	snaps, err := s.st.Snapshots(time.Now().Add(-window), maxPoints)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Safety net: the SQL downsample already bounds this, so stride is ~1.
	stride := 1
	if len(snaps) > maxPoints {
		stride = (len(snaps) + maxPoints - 1) / maxPoints
	}
	points := make([]wire.HistoryPoint, 0, maxPoints)
	for i := 0; i < len(snaps); i += stride {
		points = append(points, wire.HistoryPointFromSnapshot(&snaps[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}

// GET /api/search?q=
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	result := func(typ, ref string) {
		writeJSON(w, http.StatusOK, map[string]string{"type": typ, "ref": ref})
	}
	switch {
	case reTxHash.MatchString(q):
		result("tx", strings.ToLower(q))
	case reAddress.MatchString(q):
		result("address", strings.ToLower(q))
	case reBlock.MatchString(q):
		num, err := strconv.ParseUint(q, 10, 64)
		if err != nil || num > s.feed.Head() {
			result("none", "")
			return
		}
		result("block", q)
	default:
		result("none", "")
	}
}

// GET /api/status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	count, _ := s.st.TxCount()
	writeJSON(w, http.StatusOK, map[string]any{
		"chainId":   s.feed.ChainID(),
		"head":      s.feed.Head(),
		"dbTxCount": count,
		"uptimeSec": int(time.Since(s.startedAt).Seconds()),
	})
}
