"""
Web Content Extractor & Cleaner
Extracts clean main body text, metadata, and structured content from raw HTML.
Uses Trafilatura with BeautifulSoup4 fallback.
"""
import logging
from typing import Dict, Any, Optional
import trafilatura
from bs4 import BeautifulSoup

logger = logging.getLogger(__name__)

class ContentExtractor:
    def __init__(self):
        pass

    def extract_from_html(self, html_content: str, url: Optional[str] = None) -> Dict[str, Any]:
        """
        Extracts clean text, metadata, and core information from raw HTML content.
        """
        if not html_content or not html_content.strip():
            return {
                "title": "",
                "text": "",
                "author": None,
                "date": None,
                "description": "",
                "success": False
            }

        # 1. Primary extraction via trafilatura (industry standard for article/content extraction)
        extracted_text = trafilatura.extract(
            html_content,
            url=url,
            include_links=False,
            include_images=False,
            include_tables=True,
            output_format="txt"
        )

        metadata = trafilatura.extract_metadata(html_content, default_url=url)

        title = metadata.title if metadata and metadata.title else ""
        author = metadata.author if metadata and metadata.author else None
        date = metadata.date if metadata and metadata.date else None
        description = metadata.description if metadata and metadata.description else ""

        # 2. Fallback using BeautifulSoup if trafilatura extracted little/no text
        if not extracted_text or len(extracted_text.strip()) < 50:
            soup = BeautifulSoup(html_content, "html.parser")
            
            # Remove scripts, styles, navigations, footers
            for elem in soup(["script", "style", "nav", "footer", "aside", "header", "form"]):
                elem.decompose()

            if not title and soup.title and soup.title.string:
                title = soup.title.string.strip()

            # Meta description
            meta_desc = soup.find("meta", attrs={"name": "description"}) or soup.find("meta", attrs={"property": "og:description"})
            if not description and meta_desc and meta_desc.get("content"):
                description = meta_desc["content"].strip()

            # Text content
            lines = (line.strip() for line in soup.get_text().splitlines())
            chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
            extracted_text = "\n".join(chunk for chunk in chunks if chunk)

        return {
            "title": title or "Untitled",
            "text": extracted_text or "",
            "author": author,
            "date": date,
            "description": description,
            "success": bool(extracted_text and len(extracted_text.strip()) > 0)
        }

content_extractor = ContentExtractor()
