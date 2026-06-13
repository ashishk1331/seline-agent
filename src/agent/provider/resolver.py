from ..config import CONFIG
from ..tools import TOOLS
from ..logger import log
import httpx


class LLMResolver:
    def __init__(self) -> None:
        self._provider = CONFIG.AI_PROVIDER
        self._provider_llm_url = CONFIG.AI_PROVIDER_LLM_URL
        self._provider_api_key = CONFIG.AI_PROVIDER_API_KEY
        self._http_client = httpx.AsyncClient(headers=self._get_headers())

    def _get_headers(self):
        return {
            "Authorization": f"Bearer {self._provider_api_key}",
            "Content-Type": "application/json",
        }

    def _get_payload(self, messages):
        return {
            "model": CONFIG.MODEL_NAME,
            "max_tokens": CONFIG.MAX_TOKENS,
            "temperature": CONFIG.TEMPERATURE,
            "messages": messages,
            "tools": TOOLS,
        }

    async def resolve(self, messages):
        resp = await self._http_client.post(
            url=self._provider_llm_url,
            json=self._get_payload(messages),
        )

        if resp.status_code != 200:
            log.error(
                f"API request failed with status code {resp.status_code}: {resp.text}"
            )
            return None

        return resp.json()

    async def close(self):
        await self._http_client.aclose()


LLMRESOLVER = LLMResolver()
