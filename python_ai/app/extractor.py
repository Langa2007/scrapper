import logging
from typing import Dict, Any, Optional
import trafilatura
from bs4 import BeautifulSoup

logger = logging.getLogger(__name__)

class ContentExtractor:
    def extract_from_html(self, html_content: str, url: Optional[str] = None) -> Dict[str, Any]:
        if not html_content or not html_content.strip():
            return {
                "title": "",
                "text": "",
                "author": None,
                "date": None,
                "description": "",
                "success": False
            }

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

        if not extracted_text or len(extracted_text.strip()) < 50:
            soup = BeautifulSoup(html_content, "html.parser")

            for elem in soup(["script", "style", "nav", "footer", "aside", "header", "form"]):
                elem.decompose()

            if not title and soup.title and soup.title.string:
                title = soup.title.string.strip()

            meta_desc = soup.find("meta", attrs={"name": "description"}) or soup.find("meta", attrs={"property": "og:description"})
            if not description and meta_desc:
                content = meta_desc.get("content")
                if isinstance(content, str):
                    description = content.strip()

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
