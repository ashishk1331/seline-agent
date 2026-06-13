FROM python:3.12-alpine

COPY --from=ghcr.io/astral-sh/uv:0.4.0 /uv /uvx /bin/

WORKDIR /app

COPY pyproject.toml uv.lock ./

RUN uv sync --frozen --no-install-project

COPY src ./src/

RUN uv sync --frozen

CMD ["uv", "run", "agent"]