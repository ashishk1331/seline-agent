import httpx
from .logger import log


async def fetch(url, headers, payload):
    async with httpx.AsyncClient() as client:
        resp = await client.post(url, headers=headers, json=payload)

        if resp.status_code != 200:
            log.error(
                f"API request failed with status code {resp.status_code}: {resp.text}"
            )
            return None

        return resp.json()
