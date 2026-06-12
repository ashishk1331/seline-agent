from ..tools.registry import register_tool
from urllib.parse import urlencode
import requests as R
from ..constants import TINYFISH_HEADERS
from agent.utils import top_three
from ..config import CONFIG
from ..logger import log


@register_tool(max_chars=-1, status_message='Searching for "$query"')
def web_search(query: str):
    """Search the web. Use this to find information or discover URLs.

    Args:
        query (str): The search query.
    """
    query = urlencode({"query": query})
    resp = R.get(f"{CONFIG.TINYFISH_SEARCH_URL}?{query}", headers=TINYFISH_HEADERS)

    if resp.status_code != 200:
        log.error(f"web_search failed [{resp.status_code}]: {resp.text}")
        return f"Failed to search for {query}"

    data = resp.json()
    sites = ", ".join(
        top_three([r["site_name"].lstrip("www.") for r in data["results"]])
    )
    log.info(f"web_search [bold cyan]{query}[/] → {sites}")

    return "\n".join(
        f"[{r['title']}]({r['url']}) - {r['snippet']}" for r in data["results"]
    )


@register_tool(max_chars=8_000, status_message="Fetching $url")
def web_fetch(url):
    """Fetch the full content of a URL as markdown. Use this when you already have a URL.

    Args:
        url (str): The URL to fetch.
    """
    resp = R.post(
        CONFIG.TINYFISH_FETCH_URL,
        headers=TINYFISH_HEADERS,
        json={"urls": [url], "format": "markdown"},
    )

    if resp.status_code != 200:
        log.error(f"web_fetch failed [{resp.status_code}]: {resp.text}")
        return f"Failed to fetch {url}"

    data = resp.json()
    result = data["results"][0]
    desc = result["description"] if result["description"] else result["title"]
    log.info(f"web_fetch [bold cyan]{url}[/] → {desc}")

    return result["text"]
