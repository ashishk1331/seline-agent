from ..tools.registry import register_tool
from urllib.parse import urlencode
import requests as R
from ..constants import TINYFISH_HEADERS
from agent.utils import top_three
from ..config import CONFIG


@register_tool(max_chars=-1)
def web_search(query: str):
    """Search the web. Use this to find information or discover URLs.

    Args:
        query (str): The search query.
    """
    query = urlencode({"query": query})
    resp = R.get(f"{CONFIG.TINYFISH_SEARCH_URL}?{query}", headers=TINYFISH_HEADERS)

    if resp.status_code != 200:
        print(
            f"[TOOL] [ERROR] Search failed with status code {resp.status_code}: {resp.text}"
        )
        return f"Failed to search for {query}"

    data = resp.json()
    print(
        f'[TOOL] [WEB_SEARCH_RESULT] "{query}" = {", ".join(top_three([result["site_name"].lstrip("www.") for result in data["results"]]))}'
    )

    results = data["results"]
    return '\n'.join(
        f'[{result["title"]}]({result["url"]}) - {result["snippet"]}' for result in results
    )


@register_tool(max_chars=8_000)
def web_fetch(url):
    """Fetch the full content of a URL as markdown. Use this when you already have a URL.

    Args:
        url (str): The URL to fetch.
    """
    resp = R.post(
        CONFIG.TINYFISH_FETCH_URL,
        headers=TINYFISH_HEADERS,
        json={
            "urls": [url],
            "format": "markdown",
        },
    )

    if resp.status_code != 200:
        print(
            f"[TOOL] [ERROR] Fetch failed with status code {resp.status_code}: {resp.text}"
        )
        return f"Failed to fetch {url}"

    data = resp.json()
    print(f"[TOOL] [WEB_FETCH_RESULT] {url} = {data['results'][0]['description']}")
    return data["results"][0]['text']
