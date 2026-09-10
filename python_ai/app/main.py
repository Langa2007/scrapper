"""
FastAPI Microservice for AI Internet Scraper Core
Exposes internal REST endpoints used by the Go orchestrator.
"""
import logging
from typing import List, Optional, Dict, Any
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import uvicorn

from python_ai.app.config import settings
from python_ai.app.searcher import search_engine
from python_ai.app.extractor import content_extractor
from python_ai.app.ai_synthesizer import ai_synthesizer

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger(__name__)

app = FastAPI(title="Scraper AI Microservice", version="0.1.0")

# --- Request/Response Models ---
class SearchRequest(BaseModel):
    query: str
    max_results: int = 5
    type: str = "web"  # "web" or "news"
    providers: Optional[List[str]] = None

class CleanHtmlRequest(BaseModel):
    html: str
    url: Optional[str] = None

class DocumentItem(BaseModel):
    title: Optional[str] = ""
    url: Optional[str] = ""
    text: Optional[str] = ""
    snippet: Optional[str] = ""

class SynthesizeRequest(BaseModel):
    query: str
    documents: List[DocumentItem]

class NewsRequest(BaseModel):
    topic: str
    limit: int = 10
    timeframe: Optional[str] = "w"

# --- Endpoints ---
@app.get("/health")
def health_check():
    return {
        "status": "healthy",
        "service": "python_ai",
        "gemini_configured": bool(settings.gemini_api_key),
        "openai_configured": bool(settings.openai_api_key)
    }

@app.post("/api/ai/search")
def search_internet(req: SearchRequest):
    if req.type == "news":
        results = search_engine.search_news(req.query, max_results=req.max_results, providers=req.providers)
    else:
        results = search_engine.search_web(req.query, max_results=req.max_results, providers=req.providers)
    return {"query": req.query, "type": req.type, "count": len(results), "results": results, "providers": req.providers or search_engine.default_providers}

@app.post("/api/ai/clean-html")
def clean_html(req: CleanHtmlRequest):
    extracted = content_extractor.extract_from_html(req.html, url=req.url)
    return extracted

@app.post("/api/ai/synthesize")
def synthesize_answer(req: SynthesizeRequest):
    doc_dicts = [doc.model_dump() for doc in req.documents]
    result = ai_synthesizer.synthesize_answer(req.query, doc_dicts)
    return result

@app.post("/api/ai/news")
def get_news(req: NewsRequest):
    raw_news = search_engine.search_news(req.topic, max_results=req.limit, timelimit=req.timeframe)
    formatted = ai_synthesizer.format_news_feed(raw_news)
    return {
        "topic": req.topic,
        "count": len(formatted),
        "articles": formatted
    }

if __name__ == "__main__":
    uvicorn.run("python_ai.app.main:app", host=settings.host, port=settings.port, reload=True)
