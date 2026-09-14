import logging
import time
from typing import Any, Dict, List, Optional

import httpx

from python_ai.app.config import settings

logger = logging.getLogger(__name__)

COINGECKO_BASE = "https://api.coingecko.com/api/v3"
_HEADERS = {"accept": "application/json"}

_cache: Dict[str, tuple] = {}
_CACHE_TTL = 60


def _cached(key: str, fetch_fn):
    now = time.monotonic()
    if key in _cache:
        val, ts = _cache[key]
        if now - ts < _CACHE_TTL:
            return val
    val = fetch_fn()
    _cache[key] = (val, now)
    return val


def _client() -> httpx.Client:
    headers = dict(_HEADERS)
    if settings.coingecko_api_key:
        headers["x-cg-pro-api-key"] = settings.coingecko_api_key
    return httpx.Client(timeout=settings.coingecko_timeout_sec, headers=headers)


def _signal(change_24h: float, change_7d: Optional[float]) -> Dict[str, str]:
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


def _format_coin(coin: Dict[str, Any]) -> Dict[str, Any]:
    market = coin.get("market_data", {})
    current_usd = (market.get("current_price") or {}).get("usd")
    change_24h = (market.get("price_change_percentage_24h") or 0)
    change_7d = market.get("price_change_percentage_7d")
    change_1h = (market.get("price_change_percentage_1h_in_currency") or {}).get("usd")
    market_cap = (market.get("market_cap") or {}).get("usd")
    volume_24h = (market.get("total_volume") or {}).get("usd")
    high_24h = (market.get("high_24h") or {}).get("usd")
    low_24h = (market.get("low_24h") or {}).get("usd")
    ath = (market.get("ath") or {}).get("usd")
    atl = (market.get("atl") or {}).get("usd")

    sig = _signal(change_24h, change_7d)

    return {
        "id": coin.get("id"),
        "name": coin.get("name"),
        "symbol": (coin.get("symbol") or "").upper(),
        "image": (coin.get("image") or {}).get("small", ""),
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
        "last_updated": coin.get("last_updated"),
        "disclaimer": "Signal based on momentum calculation. Not financial advice.",
    }


def get_coin(query: str) -> Optional[Dict[str, Any]]:
    coin_id = _resolve_id(query)
    if not coin_id:
        return None

    def fetch():
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
            return _format_coin(r.json())

    try:
        return _cached(f"coin:{coin_id}", fetch)
    except Exception as e:
        logger.error("CoinGecko get_coin error: %s", e)
        return None


def get_trending() -> List[Dict[str, Any]]:
    def fetch():
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/search/trending")
            r.raise_for_status()
            items = r.json().get("coins", [])
            results = []
            for entry in items[:7]:
                item = entry.get("item", {})
                results.append({
                    "id": item.get("id"),
                    "name": item.get("name"),
                    "symbol": (item.get("symbol") or "").upper(),
                    "image": item.get("small", ""),
                    "market_cap_rank": item.get("market_cap_rank"),
                    "price_btc": item.get("price_btc"),
                })
            return results

    try:
        return _cached("trending", fetch)
    except Exception as e:
        logger.error("CoinGecko get_trending error: %s", e)
        return []


def get_market_overview(coins: List[str] = None, limit: int = 10) -> List[Dict[str, Any]]:
    params = {
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

    def fetch():
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/coins/markets", params=params)
            r.raise_for_status()
            results = []
            for item in r.json():
                change_24h = item.get("price_change_percentage_24h") or 0
                change_7d = item.get("price_change_percentage_7d_in_currency")
                sig = _signal(change_24h, change_7d)
                results.append({
                    "id": item.get("id"),
                    "name": item.get("name"),
                    "symbol": (item.get("symbol") or "").upper(),
                    "image": item.get("image", ""),
                    "price_usd": item.get("current_price"),
                    "change_1h_pct": item.get("price_change_percentage_1h_in_currency"),
                    "change_24h_pct": round(change_24h, 2),
                    "change_7d_pct": round(change_7d, 2) if change_7d is not None else None,
                    "market_cap_usd": item.get("market_cap"),
                    "volume_24h_usd": item.get("total_volume"),
                    "high_24h_usd": item.get("high_24h"),
                    "low_24h_usd": item.get("low_24h"),
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


_COIN_LIST: Optional[List[Dict]] = None
_COIN_LIST_TS: float = 0
_COIN_LIST_TTL = 3600


def _load_coin_list() -> List[Dict]:
    global _COIN_LIST, _COIN_LIST_TS
    now = time.monotonic()
    if _COIN_LIST is not None and now - _COIN_LIST_TS < _COIN_LIST_TTL:
        return _COIN_LIST
    try:
        with _client() as c:
            r = c.get(f"{COINGECKO_BASE}/coins/list")
            r.raise_for_status()
            _COIN_LIST = r.json()
            _COIN_LIST_TS = now
    except Exception as e:
        logger.warning("Could not load CoinGecko coin list: %s", e)
        _COIN_LIST = _COIN_LIST or []
    return _COIN_LIST


def _resolve_id(query: str) -> Optional[str]:
    q = query.strip().lower()
    coin_list = _load_coin_list()

    for coin in coin_list:
        if coin.get("id", "").lower() == q:
            return coin["id"]

    matches = [c for c in coin_list if c.get("symbol", "").lower() == q]
    if len(matches) == 1:
        return matches[0]["id"]

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
        return matches[0]["id"]

    for coin in coin_list:
        if coin.get("name", "").lower() == q:
            return coin["id"]

    for coin in coin_list:
        if q in coin.get("name", "").lower():
            return coin["id"]
    return None


def _resolve_ids_bulk(queries: List[str]) -> List[str]:
    return [rid for q in queries if (rid := _resolve_id(q))]
