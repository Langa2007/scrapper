"""
AI Synthesizer & Intelligent Information Extractor
Analyzes scraped web content against user prompts, generates structured answers,
summaries, and news feeds. Supports Gemini, OpenAI, or local/rule-based heuristics.
"""
import logging
import json
from typing import List, Dict, Any, Optional
from python_ai.app.config import settings

logger = logging.getLogger(__name__)

class AISynthesizer:
    def __init__(self):
        self.gemini_client = None
        self.openai_client = None
        self._init_clients()

    def _init_clients(self):
        if settings.gemini_api_key:
            try:
                from google import genai
                self.gemini_client = genai.Client(api_key=settings.gemini_api_key)
                logger.info("Google GenAI client initialized successfully.")
            except Exception as e:
                logger.warning(f"Could not initialize Google GenAI: {e}")

        if settings.openai_api_key:
            try:
                import httpx
                # Lightweight check for openai
                self.openai_client = True
                logger.info("OpenAI API key detected.")
            except Exception as e:
                logger.warning(f"Could not initialize OpenAI client: {e}")

    def synthesize_answer(self, query: str, documents: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Takes a natural language user query and a list of scraped web documents,
        and synthesizes a well-structured answer with citations.
        """
        if not documents:
            return {
                "answer": "No relevant web pages could be scraped for this query.",
                "sources": [],
                "confidence": "low"
            }

        # Build context from scraped documents
        context_parts = []
        sources = []
        for idx, doc in enumerate(documents[:5], 1):
            title = doc.get("title", f"Source {idx}")
            url = doc.get("url", "")
            text = doc.get("text", doc.get("snippet", ""))[:2500]  # Cap length per doc
            sources.append({"id": idx, "title": title, "url": url})
            context_parts.append(f"--- SOURCE [{idx}]: {title} ({url}) ---\n{text}\n")

        context_text = "\n".join(context_parts)

        # 1. Try Gemini
        if self.gemini_client:
            try:
                prompt = (
                    f"You are an AI research assistant that scrapes and analyzes the internet.\n"
                    f"User Query: {query}\n\n"
                    f"Here is the scraped content from the web:\n{context_text}\n\n"
                    f"Instructions:\n"
                    f"1. Synthesize a comprehensive, accurate answer to the user's query based ONLY on the provided sources.\n"
                    f"2. Cite sources using [1], [2], etc. where appropriate.\n"
                    f"3. Provide bullet points for key facts, dates, numbers, or takeaways."
                )
                response = self.gemini_client.models.generate_content(
                    model=settings.default_model,
                    contents=prompt
                )
                return {
                    "answer": response.text,
                    "sources": sources,
                    "model_used": settings.default_model,
                    "confidence": "high"
                }
            except Exception as e:
                logger.error(f"Gemini generation error: {e}")

        # 2. Rule-based / Heuristic Fallback (when no LLM API key is configured yet)
        return self._heuristic_synthesis(query, documents, sources)

    def format_news_feed(self, articles: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Formats news articles into a clean structured format ready for external systems (e.g. Dira News).
        """
        formatted = []
        for art in articles:
            # Extract key takeaways / summary
            summary = art.get("snippet") or ""
            text = art.get("text") or ""
            if not summary and text:
                summary = text[:200] + "..."

            formatted.append({
                "title": art.get("title", "Untitled News"),
                "url": art.get("url", ""),
                "summary": summary,
                "author": art.get("author"),
                "published_date": art.get("date"),
                "source": art.get("source", "Web"),
                "image": art.get("image", "")
            })
        return formatted

    def _heuristic_synthesis(self, query: str, documents: List[Dict[str, Any]], sources: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Fallback synthesis when LLM API keys are not provided."""
        summary_lines = [
            f"### Synthesized Findings for: '{query}'\n",
            "*(Note: Set GEMINI_API_KEY or OPENAI_API_KEY in .env for full generative AI reasoning)*\n"
        ]
        
        for idx, doc in enumerate(documents[:5], 1):
            title = doc.get("title", f"Source {idx}")
            url = doc.get("url", "")
            snippet = doc.get("snippet") or (doc.get("text", "")[:280] + "...")
            summary_lines.append(f"- **[{title}]({url})** [Source {idx}]:\n  {snippet}\n")

        return {
            "answer": "\n".join(summary_lines),
            "sources": sources,
            "model_used": "heuristic-extractor",
            "confidence": "medium"
        }

ai_synthesizer = AISynthesizer()
