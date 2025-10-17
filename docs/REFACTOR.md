# CQAR → CQC (new contracts) Refactor

## 0) Bump contracts & regenerate

* **go.mod**

  * Update dependency: `require github.com/Combine-Capital/cqc/gen/go vX.Y.Z`
* **Makefile / codegen scripts**

  * Ensure `protoc`/`buf` steps pull the updated packages
* **Run codegen** and fix imports throughout: `github.com/Combine-Capital/cqc/gen/go/cqc/...`

---

## 1) Database migrations (replace Symbols with Instruments/Markets)

Create new migrations under **/migrations**:

1. **Add** core tables

   * `instruments` (base)
   * `spot_instruments` (FK instrument_id)
   * `perp_contracts`
   * `future_contracts`
   * `option_series`
   * `lending_deposits`
   * `lending_borrows`
   * `markets` (venue listing; FK instrument_id, venue_id)
   * `identifiers` (unified; entity_type + one-of {asset_id|instrument_id|market_id})
2. **Keep/Trim**:

   * Keep `venue_assets` (lean ops flags as discussed)
   * (Optional) Add `venue_asset_networks` for per-chain CEX flags
3. **Drop/Replace**:

   * Drop `symbols`, `venue_symbols`, and any `symbol_identifiers` tables
4. **Indexes**

   * `markets(venue_id, venue_symbol)` unique
   * `identifiers(source, external_id)` unique
   * `identifiers(entity_type, asset_id|instrument_id|market_id)` unique where applicable

> Keep decimal columns as `NUMERIC` and ensure constraints (tick/lot/mins > 0). Align columns to the schemas we finalized earlier.

---

## 2) Bootstrap data updates (**/bootstrap_data**)

* Replace:

  * `symbols.json` → `instruments_spot.json`, `instruments_deriv.json` (perp/future/option), and `markets.json`
* Update:

  * `symbol_identifiers.json` → **`identifiers.json`** with `entity_type` = `ASSET|INSTRUMENT|MARKET`
* Keep:

  * `venue_assets.json` (same file; fields trimmed to the lean set if needed)

---

## 3) Repositories (Postgres layer) — **/internal/repo/**

Create/modify packages (names may differ; keep your project style):

* **/internal/repo/instrument_repo.go**

  * `GetInstrument(id string) (*cqc.markets.v1.Instrument, error)`
  * `GetInstrumentSubtype(id string) (one of SpotInstrument/PerpContract/FutureContract/OptionSeries/Lending*, error)`
  * `CreateInstrument(...)`, `Upsert...` as needed
* **/internal/repo/market_repo.go**

  * `GetMarket(id string) (*cqc.markets.v1.Market, error)`
  * `ResolveMarket(venueID, venueSymbol string) (*Market, error)`  ← **critical**
  * `ListMarketsByInstrument(instrumentID string) ([]Market, error)`
* **/internal/repo/identifier_repo.go**

  * `ResolveByExternalID(entityType, source, externalID string) (assetID|instrumentID|marketID, error)`
  * `ListIdentifiers(entityType, entityID string) ([]Identifier, error)`
* **/internal/repo/venue_asset_repo.go** (keep)

  * CRUD + lookups for `(venue_id, asset_id)`; optionally `(venue_id, asset_id, chain_id)` if you add networks

Remove old symbol repos:

* **DELETE** `/internal/repo/symbol_repo.go`
* **DELETE** `/internal/repo/venue_symbol_repo.go`
* **DELETE** `/internal/repo/symbol_identifier_repo.go`

---

## 4) Caches (Redis) — **/internal/cache/**

* **Keys**

  * `market:{venue_id}:{venue_symbol} -> market_id`
  * `instrument:{id} -> Instrument + subtype snapshot`
  * `market:{id} -> Market snapshot`
  * `identifier:{source}:{external_id} -> {entity_type,id}`
* Remove:

  * Any `symbol:*` keys

---

## 5) Services/Handlers (gRPC) — **/internal/handlers/grpc/** or **/internal/services/**

Implement the new CQC service surface (names may differ in your tree; align to **cqc/services/v1**):

* **Reference Data**

  * **Add**:

    * `GetInstrument`
    * `GetSpotInstrument`
    * `GetPerpContract`
    * `GetFutureContract`
    * `GetOptionSeries`
    * `GetLendingDeposit`
    * `GetLendingBorrow`
    * `GetMarket`
    * `ResolveMarket` (request: `venue_id`, `venue_symbol`; response: `{market_id, instrument_id}`)
  * **Remove**:

    * `GetSymbol*`, `ListSymbols*`, `ResolveSymbol*`
* **Market Data (if CQAR exposes any read-through)**

  * **All requests** targeting listings should pass **`market_id`** (not `symbol_id`)
  * Provide a resolver path `{venue_id, venue_symbol} -> market_id` using Reference Data above
* **Trading (if proxied)**

  * Target **`market_id`** for orders; include `instrument_id` in responses when helpful

---

## 6) Business logic — **/internal/domain/** (or equivalent)

* **Replace symbol-centric flows** with:

  * **Instrument-oriented** lineage (underlying asset, contract traits)
  * **Market-oriented** venue specifics (tick/lot/min_notional/maker&taker/funding_interval)
* **Validation**

  * For OPTIONS ensure `option_type`/`exercise_style` strings validated against allowed-set (string set, not enums)
  * For PERP/FUTURE handle `is_inverse`, `is_quanto`, `contract_multiplier`
* **Derivatives linkage**

  * Ensure `instrument.underlying_asset_id` is enforced for *Perp/Future/Option/Lending* subtypes

---

## 7) HTTP & Admin endpoints — **/internal/http/** (if present)

* Swap any `symbol` params to `instrument_id` or `market_id`
* Add `/markets/resolve?venue_id=...&venue_symbol=...`

---

## 8) Events — **/internal/events/**

* Publish/consume with the new nouns:

  * `InstrumentCreated/Updated`
  * `MarketListed/Updated/Delisted`
  * (If needed) `IdentifierLinked/Unlinked`
* Remove any `SymbolCreated/Updated` topics

---

## 9) Tests — **/test/** (and **/internal/***_test.go)

* Update fixtures:

  * Instruments + subtypes + markets + identifiers
* Replace symbol tests with:

  * `ResolveMarket(binance, BTCUSDT)` → returns `{market_id, instrument_id}`
  * Spot: ETH/USDC; Perp: ETH-PERP; Lending: WETH deposit / USDC borrow
* Keep venue_asset tests (deposit/withdraw flags)

---

## 10) Docs — **/docs/**

* **README.md**

  * Replace “Symbol / VenueSymbol” with “Instrument / Market”
  * Update Architecture diagram blocks accordingly (Instrument Manager / Market Manager, Identifier)
  * Update Quick Start examples to show `ResolveMarket` and `GetInstrument`
* **SPEC.md / BRIEF.md / ROADMAP.md (if present)**

  * Mirror the contract changes & new DB model

---

## 11) Remove dead code & configs

* Rip out any feature flags, env vars, or config sections that reference `symbol` or `venue_symbol` repositories or RPCs.

---

## 12) Smoke checklist (end-to-end)

1. **Migration up**: tables created; no `symbols`/`venue_symbols` remain
2. **Bootstrap load**: assets → deployments → instruments → markets → identifiers → venue_assets
3. **Resolve**: `ResolveMarket(binance, BTCUSDT)` returns `{market_id, instrument_id}`
4. **Fetch**: `GetInstrument(instrument_id)` + `GetSpotInstrument(instrument_id)`
5. **Fetch**: `GetMarket(market_id)` (tick/lot/min_notional/fees)
6. **Identifier**: `ResolveByExternalID(entity=MARKET, source=tradingview, external_id="BINANCE:BTCUSDT") -> market_id`
7. **VenueAsset**: `GetVenueAsset(binance, BTC)` shows deposit/withdraw flags
