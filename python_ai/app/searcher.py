"""
Internet Search Provider
Provides search capabilities across general web and news using DuckDuckGo.
"""
import logging
from typing import List, Dict, Any, Optional
try:
    from ddgs import DDGS
except ImportError:
    from duckduckgo_search import DDGS

logger = logging.getLogger(__name__)

class SearchEngine:
    def __init__(self):
        pass

    def search_web(self, query: str, max_results: int = 5) -> List[Dict[str, Any]]:
        """
        Searches the open internet for general queries.
        Returns a list of dicts with title, url, snippet.
        """
        results = []
        try:
            with DDGS() as ddgs:
                ddgs_results = ddgs.text(query, max_results=max_results)
                for item in ddgs_results:
                    results.append({
                        "title": item.get("title", ""),
                        "url": item.get("href", ""),
                        "snippet": item.get("body", ""),
                        "source": "web"
                    })
        except Exception as e:
            logger.error(f"Error performing web search for '{query}': {e}")
        return results

    def search_news(self, query: str, max_results: int = 10, timelimit: Optional[str] = "w") -> List[Dict[str, Any]]:
        """
        Searches news specifically (ideal for Dira News integration).
        timelimit: 'd' (day), 'w' (week), 'm' (month)
        """
        results = []
        try:
            with DDGS() as ddgs:
                ddgs_news = ddgs.news(query, max_results=max_results, timelimit=timelimit)
                for item in ddgs_news:
                    results.append({
                        "title": item.get("title", ""),
                        "url": item.get("url", ""),
                        "snippet": item.get("body", ""),
                        "date": item.get("date", ""),
                        "source": item.get("source", "news"),
                        "image": item.get("image", "")
                    })
        except Exception as e:
            logger.error(f"Error performing news search for '{query}': {e}")
        return results

search_engine = SearchEngine()
