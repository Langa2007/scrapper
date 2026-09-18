import logging
import time
from collections.abc import Callable, Mapping
from typing import Any, Optional, TypeVar, cast

import httpx

from python_ai.app.config import settings

logger = logging.getLogger(__name__)

COINGECKO_BASE = "https://api.coingecko.com/api/v3"
_HEADERS = {"accept": "application/json"}
_CACHE_TTL = 60
_COIN_LIST_TTL = 3600

T = TypeVar("T")

Signal = dict[str, str | float]
JsonObject = Mapping[str, Any]

_cache: dict[str, tuple[object, float]] = {}
_coin_list_cache: list[JsonObject] | None = None
_coin_list_ts: float = 0.0


def _cached(key: str, fetch_fn: Callable[[], T]) -> T:
    now = time.monotonic()
    cached = _cache.get(key)
    if cached is not None:
        val, ts = cached
        if now - ts < _CACHE_TTL:
            return cast(T, val)
    val = fetch_fn()
    _cache[key] = (val, now)
    return val


def _client() -> httpx.Client:
    headers = dict(_HEADERS)
    if settings.coingecko_api_key:
        headers["x-cg-pro-api-key"] = settings.coingecko_api_key
    return httpx.Client(timeout=settings.coingecko_timeout_sec, headers=headers)


def _as_mapping(value: object) -> JsonObject:
    if isinstance(value, Mapping):
        return cast(JsonObject, value)
    return {}


def _as_json_list(value: object) -> list[JsonObject]:
    if not isinstance(value, list):
        return []
    items = cast(list[object], value)
    return [_as_mapping(item) for item in items]


def _json_object(response: httpx.Response) -> JsonObject:
    return _as_mapping(response.json())


def _json_list(response: httpx.Response) -> list[JsonObject]:
    return _as_json_list(response.json())


def _string(value: object, default: str = "") -> str:
    return value if isinstance(value, str) else default


def _number(value: object, default: float = 0.0) -> float:
    if isinstance(value, bool):
        return default
    if isinstance(value, (int, float)):
        return float(value)
    return default


def _optional_number(value: object) -> float | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    return None


def _optional_int(value: object) -> int | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    return None


def _signal(change_24h: float, change_7d: Optional[float]) -> Signal:
    score = change_24h
    if change_7d is not None:
        score = change_24h * 0.6 + change_7d * 0.4

    if score >= 5:
        label, color = "STRONG BUY", "green"
    elif score >= 2:
        label, color = "BUY", "lightgreen"
    elif score <= -5:
        label, color = "STRONG SELL", "red"
    elif score <= -2:
        label, color = "SELL", "orange"
    else:
        label, color = "HOLD", "gray"

    return {"signal": label, "color": color, "momentum_score": round(score, 2)}


def _format_coin(coin: JsonObject) -> dict[str, Any]:
    market = _as_mapping(coin.get("market_data"))
    current_usd = _optional_number(_as_mapping(market.get("current_price")).get("usd"))
    change_24h = _number(market.get("price_change_percentage_24h"))
    change_7d = _optional_number(market.get("price_change_percentage_7d"))
    change_1h = _optional_number(_as_mapping(market.get("price_change_percentage_1h_in_currency")).get("usd"))
    market_cap = _optional_number(_as_mapping(market.get("market_cap")).get("usd"))
    volume_24h = _optional_number(_as_mapping(market.get("total_volume")).get("usd"))
    high_24h = _optional_number(_as_mapping(market.get("high_24h")).get("usd"))
    low_24h = _optional_number(_as_mapping(market.get("low_24h")).get("usd"))
    ath = _optional_number(_as_mapping(market.get("ath")).get("usd"))
    atl = _optional_number(_as_mapping(market.get("atl")).get("usd"))
    image = _as_mapping(coin.get("image"))

    sig = _signal(change_24h, change_7d)

    return {
        "id": _string(coin.get("id")),
        "name": _string(coin.get("name")),
        "symbol": _string(coin.get("symbol")).upper(),
        "image": _string(image.get("small")),
        "price_usd": current_usd,
        "change_1h_pct": round(change_1h, 2) if change_1h is not None else None,
        "change_24h_pct": round(change_24h, 2),
        "change_7d_pct": round(change_7d, 2) if change_7d is not None else None,
        "high_24h_usd": high_24h,
        "low_24h_usd": low_24h,
        "market_cap_usd": market_cap,
        "volume_24h_usd": volume_24h,
        "ath_usd": ath,
        "atl_usd": atl,
        "signal": sig["signal"],
        "signal_color": sig["color"],
        "momentum_score": sig["momentum_score"],
        "last_updated": _string(coin.get("last_updated")),
        "disclaimer": "Signal based on momentum calculation. Not financial advice.",
    }


def _attach_futures_setup(data: dict[str, Any]) -> dict[str, Any]:
    symbol = str(data.get("symbol") or "").upper()
    if not symbol:
        return data

    matching_ticker = None
    for ticker in get_binance_futures_tickers():
        ticker_symbol = str(ticker.get("symbol") or "").upper()
        ticker_base = str(ticker.get("base") or "").upper()
        if ticker_symbol == f"{symbol}USDT" or ticker_base == symbol:
            matching_ticker = ticker
            break

    if matching_ticker is None:
        return data

    price = float(matching_ticker.get("price") or 0.0)
    high_24h = float(matching_ticker.get("high_24h") or 0.0)
    low_24h = float(matching_ticker.get("low_24h") or 0.0)
    change_pct = float(matching_ticker.get("change_pct") or 0.0)
    direction = "short" if change_pct >= 0 else "long"
    setup = _calculate_futures_setup(price, high_24h, low_24h, change_pct, direction)

    data["futures_setup"] = {
        "symbol": matching_ticker["symbol"],
        "direction": direction,
        "entry": setup["entry"],
        "sl": setup["sl"],
        "tp1": setup["tp1"],
        "tp2": setup["tp2"],
        "tp3": setup["tp3"],
        "leverage": setup["leverage"],
        "risk_reward": setup["rr"],
        "risk_pct": setup["risk_pct"],
        "rationale": (
            "Momentum is extended into resistance; wait for a small retracement before entering."
            if direction == "short"
            else "Dip is showing support; wait for a bounce into the lower entry zone before entering."
        ),
    }
    return data


def get_coin(query: str) -> Optional[dict[str, Any]]:
    coin_id = _resolve_id(query)
    if not coin_id:
        return None

    def fetch() -> dict[str, Any]:
        with _client() as c:
            r = c.get(
                f"{COINGECKO_BASE}/coins/{coin_id}",
                params={
                    "localization": "false",
                    "tickers": "false",
                    "market_data": "true",
                    "community_data": "false",
                    "developer_data": "false",
                },
            )
            r.raise_for_status()
            return _attach_futures_setup(_format_coin(_json_object(r)))

    try:
        return _cached(f"coin:{coin_id}", fetch)
    except Exception as e:
        logger.error("CoinGecko get_coin error: %s", e)
        return None


def get_trending() -> list[dict[str, Any]]:
    def fetch() -> list[dict[str, Any]]:
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/search/trending")
            r.raise_for_status()
            items = _as_json_list(_json_object(r).get("coins"))
            results: list[dict[str, Any]] = []
            for entry in items[:7]:
                item = _as_mapping(entry.get("item"))
                results.append({
                    "id": _string(item.get("id")),
                    "name": _string(item.get("name")),
                    "symbol": _string(item.get("symbol")).upper(),
                    "image": _string(item.get("small")),
                    "market_cap_rank": _optional_int(item.get("market_cap_rank")),
                    "price_btc": _optional_number(item.get("price_btc")),
                })
            return results

    try:
        return _cached("trending", fetch)
    except Exception as e:
        logger.error("CoinGecko get_trending error: %s", e)
        return []


def get_market_overview(coins: Optional[list[str]] = None, limit: int = 10) -> list[dict[str, Any]]:
    params: dict[str, str | int] = {
        "vs_currency": "usd",
        "order": "market_cap_desc",
        "per_page": min(limit, 25),
        "page": 1,
        "sparkline": "false",
        "price_change_percentage": "1h,24h,7d",
    }
    if coins:
        ids = _resolve_ids_bulk(coins)
        if ids:
            params["ids"] = ",".join(ids)

    def fetch() -> list[dict[str, Any]]:
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/coins/markets", params=params)
            r.raise_for_status()
            results: list[dict[str, Any]] = []
            for item in _json_list(r):
                change_24h = _number(item.get("price_change_percentage_24h"))
                change_7d = _optional_number(item.get("price_change_percentage_7d_in_currency"))
                sig = _signal(change_24h, change_7d)
                results.append({
                    "id": _string(item.get("id")),
                    "name": _string(item.get("name")),
                    "symbol": _string(item.get("symbol")).upper(),
                    "image": _string(item.get("image")),
                    "price_usd": _optional_number(item.get("current_price")),
                    "change_1h_pct": _optional_number(item.get("price_change_percentage_1h_in_currency")),
                    "change_24h_pct": round(change_24h, 2),
                    "change_7d_pct": round(change_7d, 2) if change_7d is not None else None,
                    "market_cap_usd": _optional_number(item.get("market_cap")),
                    "volume_24h_usd": _optional_number(item.get("total_volume")),
                    "high_24h_usd": _optional_number(item.get("high_24h")),
                    "low_24h_usd": _optional_number(item.get("low_24h")),
                    "signal": sig["signal"],
                    "signal_color": sig["color"],
                    "momentum_score": sig["momentum_score"],
                })
            return results

    try:
        cache_key = f"market:{'|'.join(sorted(coins or []))}:{limit}"
        return _cached(cache_key, fetch)
    except Exception as e:
        logger.error("CoinGecko get_market_overview error: %s", e)
        return []


def _load_coin_list() -> list[JsonObject]:
    global _coin_list_cache, _coin_list_ts
    now = time.monotonic()
    if _coin_list_cache is not None and now - _coin_list_ts < _COIN_LIST_TTL:
        return _coin_list_cache
    try:
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/coins/list")
            r.raise_for_status()
            _coin_list_cache = _json_list(r)
            _coin_list_ts = now
    except Exception as e:
        logger.warning("Could not load CoinGecko coin list: %s", e)
        if _coin_list_cache is None:
            _coin_list_cache = []
    return _coin_list_cache


def _resolve_id(query: str) -> Optional[str]:
    q = query.strip().lower()
    coin_list = _load_coin_list()

    for coin in coin_list:
        coin_id = _string(coin.get("id"))
        if coin_id.lower() == q:
            return coin_id

    matches = [c for c in coin_list if _string(c.get("symbol")).lower() == q]
    if len(matches) == 1:
        return _string(matches[0].get("id")) or None

    prefer = {
        "btc": "bitcoin", "eth": "ethereum", "bnb": "binancecoin",
        "sol": "solana", "ada": "cardano", "xrp": "ripple",
        "doge": "dogecoin", "dot": "polkadot", "avax": "avalanche-2",
        "matic": "matic-network", "link": "chainlink", "ltc": "litecoin",
        "usdt": "tether", "usdc": "usd-coin", "shib": "shiba-inu",
        "trx": "tron", "uni": "uniswap", "atom": "cosmos",
    }
    if q in prefer:
        return prefer[q]
    if matches:
        return _string(matches[0].get("id")) or None

    for coin in coin_list:
        if _string(coin.get("name")).lower() == q:
            return _string(coin.get("id")) or None

    for coin in coin_list:
        if q in _string(coin.get("name")).lower():
            return _string(coin.get("id")) or None
    return None


def _resolve_ids_bulk(queries: list[str]) -> list[str]:
    return [rid for q in queries if (rid := _resolve_id(q))]


# ---------------------------------------------------------------------------
# Binance USDT-M Futures Screener
# ---------------------------------------------------------------------------

BINANCE_FUTURES_BASE = "https://fapi.binance.com/fapi/v1"
_FUTURES_CACHE_TTL = 60  # seconds

_futures_cache: tuple[list[dict[str, Any]], float] | None = None


def get_binance_futures_tickers() -> list[dict[str, Any]]:
    """Fetch all 24-h ticker stats from Binance USDT-M futures."""
    global _futures_cache  # noqa: PLW0603
    now = time.monotonic()
    if _futures_cache is not None:
        data, ts = _futures_cache
        if now - ts < _FUTURES_CACHE_TTL:
            return data

    try:
        with httpx.Client(timeout=15) as client:
            resp = client.get(f"{BINANCE_FUTURES_BASE}/ticker/24hr")
            resp.raise_for_status()
            raw = resp.json()
    except Exception as exc:  # noqa: BLE001
        logger.warning("Binance futures ticker fetch failed: %s", exc)
        return []

    tickers: list[dict[str, Any]] = []
    raw_list = cast(list[object], raw) if isinstance(raw, list) else []
    for item in raw_list:
        obj = _as_mapping(item)
        symbol = _string(obj.get("symbol"))
        if not symbol.endswith("USDT"):
            continue
        try:
            tickers.append(
                {
                    "symbol": symbol,
                    "base": symbol.replace("USDT", ""),
                    "price": float(_string(obj.get("lastPrice")) or "0"),
                    "change_pct": float(_string(obj.get("priceChangePercent")) or "0"),
                    "high_24h": float(_string(obj.get("highPrice")) or "0"),
                    "low_24h": float(_string(obj.get("lowPrice")) or "0"),
                    "volume_usdt": float(_string(obj.get("quoteVolume")) or "0"),
                }
            )
        except ValueError:
            continue

    _futures_cache = (tickers, now)
    return tickers


def _calculate_futures_setup(
    price: float,
    high_24h: float,
    low_24h: float,
    change_pct: float,
    direction: str,
) -> dict[str, Any]:
    """Compute entry, TP1-3, stop-loss, R:R ratio, and leverage tier."""

    # Tier-based leverage: majors (BTC/ETH) → 8-10x; alts → 3-5x
    leverage = 10 if change_pct < 20 and price > 1_000 else (8 if price > 100 else 5)

    if direction == "short":
        # Entry slightly below current price (wait for micro-pullback)
        entry = round(price * 0.998, 6)
        sl = round(high_24h * 1.015, 6)          # 1.5% above 24h high
        tp1 = round(entry * (1 - 0.035), 6)       # −3.5%
        tp2 = round(entry * (1 - 0.07), 6)        # −7%
        tp3 = round(entry * (1 - 0.12), 6)        # −12%
    else:  # long
        entry = round(price * 1.002, 6)
        sl = round(low_24h * 0.985, 6)            # 1.5% below 24h low
        tp1 = round(entry * (1 + 0.035), 6)
        tp2 = round(entry * (1 + 0.07), 6)
        tp3 = round(entry * (1 + 0.12), 6)

    # Risk = |entry - sl| / entry, Reward = |tp1 - entry| / entry
    risk = abs(entry - sl) / entry if entry else 0.0
    reward = abs(tp1 - entry) / entry if entry else 0.0
    rr = round(reward / risk, 2) if risk > 0 else 0.0

    return {
        "entry": entry,
        "sl": sl,
        "tp1": tp1,
        "tp2": tp2,
        "tp3": tp3,
        "leverage": leverage,
        "rr": rr,
        "risk_pct": round(risk * 100, 2),
    }


def _entry_is_still_live(price: float, entry: float, direction: str) -> bool:
    if not price or not entry:
        return False
    if direction == "short":
        # entry zone was a small pullback/retest; once price moves beyond ~0.8% above entry, it is stale
        return price <= entry * 1.008
    if direction == "long":
        # long entry zone was near support; once price drops more than ~0.8% below entry, it is stale
        return price >= entry * 0.992
    return True


def get_futures_signals(
    strategy: str = "short",
    min_volume: float = 15_000_000.0,
    limit: int = 10,
) -> list[dict[str, Any]]:
    """
    Screen Binance USDT-M futures for actionable setups.

    strategy:
      "short"    – coins that surged ≥8% in 24h and are near the 24h high
      "long"     – coins that dropped ≥8% in 24h and are near the 24h low
      "trending" – top gainers by absolute |change_pct|, mixed direction
    """
    tickers = get_binance_futures_tickers()
    results: list[dict[str, Any]] = []
    now_ts = time.time()

    for t in tickers:
        price: float = t["price"]
        high_24h: float = t["high_24h"]
        low_24h: float = t["low_24h"]
        change_pct: float = t["change_pct"]
        volume_usdt: float = t["volume_usdt"]

        if volume_usdt < min_volume:
            continue
        if price <= 0 or high_24h <= 0 or low_24h <= 0:
            continue

        range_24h = high_24h - low_24h
        position_in_range = (price - low_24h) / range_24h if range_24h > 0 else 0.5
        # 0 = near low, 1 = near high

        if strategy == "short":
            # Require: surged ≥8%, price in top 20% of 24h range
            if change_pct < 8.0 or position_in_range < 0.80:
                continue
            direction = "short"
            score = change_pct * position_in_range  # higher = better short candidate
        elif strategy == "long":
            # Require: dropped ≥8%, price in bottom 20% of 24h range
            if change_pct > -8.0 or position_in_range > 0.20:
                continue
            direction = "long"
            score = abs(change_pct) * (1 - position_in_range)
        else:  # trending
            if abs(change_pct) < 5.0:
                continue
            direction = "short" if change_pct > 0 else "long"
            score = abs(change_pct)

        setup = _calculate_futures_setup(price, high_24h, low_24h, change_pct, direction)
        if not _entry_is_still_live(price, setup["entry"], direction):
            continue

        results.append(
            {
                "symbol": t["symbol"],
                "base": t["base"],
                "direction": direction,
                "price": price,
                "change_pct": round(change_pct, 2),
                "volume_usdt": volume_usdt,
                "high_24h": high_24h,
                "low_24h": low_24h,
                "position_in_range": round(position_in_range * 100, 1),
                "score": round(score, 2),
                "refreshed_at": now_ts,
                "expires_at": now_ts + 180,
                "is_stale": False,
                **setup,
            }
        )

    results.sort(key=lambda x: x["score"], reverse=True)
    return results[:limit]
