import asyncio
import logging
import os
import time
from typing import Optional, Dict, Any
from dataclasses import dataclass

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type,
)

from .base import BaseProvider, ProviderConfig, ProviderResponse

logger = logging.getLogger(__name__)

try:
    import httpx

    HTTPX_AVAILABLE = True
except ImportError:
    HTTPX_AVAILABLE = False


class OpenRouterProvider(BaseProvider):
    """OpenRouter - unified API for ALL major models.

    Single API key, access to every provider. Best for:
    - Trying different models without multiple accounts
    - Fallback routing when one provider is down
    - Cost optimization (routes to cheapest available)

    Models available (January 2026) - examples:
    - anthropic/claude-4.5-sonnet, anthropic/claude-4.5-opus
    - openai/gpt-5.2, openai/gpt-5-turbo, openai/o3-mini
    - google/gemini-3.0-pro, google/gemini-3.0-flash
    - meta-llama/llama-4-maverick
    - mistralai/mistral-large-2501
    - deepseek/deepseek-v3
    - And 100+ more at https://openrouter.ai/models
    """

    DEFAULT_MODEL = "anthropic/claude-4.5-sonnet"
    BASE_URL = "https://openrouter.ai/api/v1"

    def __init__(
        self,
        api_key: str,
        model: str = None,
        site_url: Optional[str] = None,
        site_name: Optional[str] = None,
    ):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)
        self.site_url = site_url or os.environ.get("OPENROUTER_SITE_URL", "")
        self.site_name = site_name or os.environ.get(
            "OPENROUTER_SITE_NAME", "TranslationWorker"
        )

    def is_available(self) -> bool:
        if not HTTPX_AVAILABLE:
            raise ImportError("httpx package required")
        if not self.config.api_key:
            raise ValueError("API key required for OpenRouterProvider")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        start = time.time()
        max_tokens = max_tokens or self.config.max_tokens

        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": self.site_url,
            "X-Title": self.site_name,
        }

        body = {
            "model": self.config.model,
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": max_tokens,
            "temperature": temperature,
        }

        async with httpx.AsyncClient(timeout=self.config.timeout) as client:
            response = await client.post(
                f"{self.BASE_URL}/chat/completions",
                headers=headers,
                json=body,
            )
        response.raise_for_status()
        data = response.json()

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=data["choices"][0]["message"]["content"],
            model=data.get("model", self.config.model),
            usage={
                "prompt_tokens": data.get("usage", {}).get("prompt_tokens", 0),
                "completion_tokens": data.get("usage", {}).get("completion_tokens", 0),
                "total_tokens": data.get("usage", {}).get("total_tokens", 0),
            },
            latency_ms=latency,
        )

    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        return asyncio.run(self.generate_async(prompt, max_tokens, temperature))


class GitHubModelsProvider(BaseProvider):
    """GitHub Models - free tier access to GPT, Claude, Llama, and more.

    Free with GitHub account (generous limits). Great for development.

    Models available (January 2026):
    - OpenAI: gpt-5.2, gpt-4.1, gpt-4.1-mini, o3-mini
    - Anthropic: claude-4.5-sonnet, claude-4.5-haiku
    - Meta: Llama-4-Maverick, Llama-4-Scout
    - Microsoft: Phi-4
    - Mistral: mistral-large-2501

    Requires GitHub token with models:read scope.
    """

    DEFAULT_MODEL = "gpt-4.1"
    BASE_URL = "https://models.inference.ai.azure.com"

    def __init__(self, api_key: str, model: str = None):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=api_key, model=model)
        super().__init__(config)

    def is_available(self) -> bool:
        if not HTTPX_AVAILABLE:
            raise ImportError("httpx package required")
        if not self.config.api_key:
            raise ValueError("GitHub token required")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        start = time.time()
        max_tokens = max_tokens or self.config.max_tokens

        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
        }

        body = {
            "model": self.config.model,
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": max_tokens,
            "temperature": temperature,
        }

        async with httpx.AsyncClient(timeout=self.config.timeout) as client:
            response = await client.post(
                f"{self.BASE_URL}/chat/completions",
                headers=headers,
                json=body,
            )
        response.raise_for_status()
        data = response.json()

        latency = int((time.time() - start) * 1000)

        return ProviderResponse(
            text=data["choices"][0]["message"]["content"],
            model=data.get("model", self.config.model),
            usage={
                "prompt_tokens": data.get("usage", {}).get("prompt_tokens", 0),
                "completion_tokens": data.get("usage", {}).get("completion_tokens", 0),
                "total_tokens": data.get("usage", {}).get("total_tokens", 0),
            },
            latency_ms=latency,
        )

    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        return asyncio.run(self.generate_async(prompt, max_tokens, temperature))


class AWSBedrockProvider(BaseProvider):
    """AWS Bedrock - access Claude, GPT, Llama via AWS infrastructure.

    Bedrock is a managed service that provides access to foundation models
    from Anthropic, OpenAI, Meta, and others through AWS. Use this when
    you have AWS credits or need AWS-region compliance.

    Models available (January 2026):
    - anthropic.claude-4-5-sonnet-20250929-v1:0 (Claude 4.5 Sonnet)
    - anthropic.claude-4-5-opus-20251101-v1:0 (Claude 4.5 Opus)
    - openai.gpt-5-v1:0 (GPT-5 via Bedrock)
    - meta.llama4-maverick-instruct-v1:0 (Llama 4 Maverick)

    Requires AWS credentials (access key + secret) or IAM role.
    """

    DEFAULT_MODEL = "anthropic.claude-4-5-sonnet-20250929-v1:0"
    DEFAULT_REGION = "us-east-1"

    def __init__(
        self,
        access_key: str,
        secret_key: str,
        model: str = None,
        region: str = None,
        session_token: Optional[str] = None,
    ):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key=access_key, model=model)
        super().__init__(config)
        self.secret_key = secret_key
        self.region = region or os.environ.get("AWS_REGION", self.DEFAULT_REGION)
        self.session_token = session_token
        self._client = None

    def _get_client(self):
        if self._client is None:
            try:
                import boto3
            except ImportError:
                raise ImportError("boto3 package required. Install: pip install boto3")

            session_kwargs = {
                "aws_access_key_id": self.config.api_key,
                "aws_secret_access_key": self.secret_key,
                "region_name": self.region,
            }
            if self.session_token:
                session_kwargs["aws_session_token"] = self.session_token

            session = boto3.Session(**session_kwargs)
            self._client = session.client("bedrock-runtime")
        return self._client

    def is_available(self) -> bool:
        if not self.config.api_key or not self.secret_key:
            raise ValueError("AWS credentials required")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        import json

        start = time.time()
        client = self._get_client()
        max_tokens = max_tokens or self.config.max_tokens

        if "anthropic" in self.config.model:
            body = {
                "anthropic_version": "bedrock-2023-05-31",
                "max_tokens": max_tokens,
                "temperature": temperature,
                "messages": [{"role": "user", "content": prompt}],
            }
        elif "titan" in self.config.model:
            body = {
                "inputText": prompt,
                "textGenerationConfig": {
                    "maxTokenCount": max_tokens,
                    "temperature": temperature,
                },
            }
        else:
            body = {
                "prompt": prompt,
                "max_gen_len": max_tokens,
                "temperature": temperature,
            }

        response = client.invoke_model(
            modelId=self.config.model,
            body=json.dumps(body),
            contentType="application/json",
            accept="application/json",
        )

        response_body = json.loads(response["body"].read())
        latency = int((time.time() - start) * 1000)

        if "anthropic" in self.config.model:
            text = response_body["content"][0]["text"]
            usage = {
                "prompt_tokens": response_body.get("usage", {}).get("input_tokens", 0),
                "completion_tokens": response_body.get("usage", {}).get(
                    "output_tokens", 0
                ),
            }
        elif "titan" in self.config.model:
            text = response_body["results"][0]["outputText"]
            usage = {"prompt_tokens": 0, "completion_tokens": 0}
        else:
            text = response_body.get("generation", "")
            usage = {"prompt_tokens": 0, "completion_tokens": 0}

        usage["total_tokens"] = usage["prompt_tokens"] + usage["completion_tokens"]

        return ProviderResponse(
            text=text, model=self.config.model, usage=usage, latency_ms=latency
        )

    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            None, lambda: self.generate(prompt, max_tokens, temperature)
        )


class VertexAIProvider(BaseProvider):
    """Google Vertex AI - Gemini, Claude, Llama via GCP.

    Use GCP credits, regional compliance, or enterprise agreements.

    Models available (January 2026):
    - Gemini: gemini-3.0-pro, gemini-3.0-flash, gemini-3.0-ultra
    - Claude (Model Garden): claude-4-5-sonnet@20250929, claude-4-5-opus@20251101
    - Llama: llama-4-maverick-instruct-maas
    - Mistral: mistral-large-maas

    Requires Google Cloud credentials (ADC or service account).
    """

    DEFAULT_MODEL = "gemini-3.0-flash"
    DEFAULT_LOCATION = "us-central1"

    def __init__(
        self,
        project_id: str,
        model: str = None,
        location: str = None,
        credentials_path: Optional[str] = None,
    ):
        model = model or self.DEFAULT_MODEL
        config = ProviderConfig(api_key="vertex", model=model)
        super().__init__(config)
        self.project_id = project_id
        self.location = location or os.environ.get(
            "VERTEX_LOCATION", self.DEFAULT_LOCATION
        )
        self.credentials_path = credentials_path
        self._client = None

    def _get_client(self):
        if self._client is None:
            try:
                import google.auth
                from google.auth.transport.requests import Request
                import google.auth.credentials
            except ImportError:
                raise ImportError(
                    "google-auth package required. Install: pip install google-auth"
                )

            if self.credentials_path:
                os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = self.credentials_path

            credentials, _ = google.auth.default()
            credentials.refresh(Request())
            self._credentials = credentials
            self._client = True
        return self._client

    def is_available(self) -> bool:
        if not self.project_id:
            raise ValueError("project_id required for VertexAIProvider")
        return True

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=1, max=60),
        retry=retry_if_exception_type((Exception,)),
    )
    async def generate_async(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        from google.auth.transport.requests import Request

        start = time.time()
        self._get_client()
        max_tokens = max_tokens or self.config.max_tokens

        self._credentials.refresh(Request())
        access_token = self._credentials.token

        endpoint = (
            f"https://{self.location}-aiplatform.googleapis.com/v1/"
            f"projects/{self.project_id}/locations/{self.location}/"
            f"publishers/google/models/{self.config.model}:generateContent"
        )

        headers = {
            "Authorization": f"Bearer {access_token}",
            "Content-Type": "application/json",
        }

        body = {
            "contents": [{"role": "user", "parts": [{"text": prompt}]}],
            "generationConfig": {
                "maxOutputTokens": max_tokens,
                "temperature": temperature,
            },
        }

        async with httpx.AsyncClient(timeout=self.config.timeout) as client:
            response = await client.post(endpoint, headers=headers, json=body)
        response.raise_for_status()
        data = response.json()

        latency = int((time.time() - start) * 1000)

        text = data["candidates"][0]["content"]["parts"][0]["text"]
        usage_metadata = data.get("usageMetadata", {})

        return ProviderResponse(
            text=text,
            model=self.config.model,
            usage={
                "prompt_tokens": usage_metadata.get("promptTokenCount", 0),
                "completion_tokens": usage_metadata.get("candidatesTokenCount", 0),
                "total_tokens": usage_metadata.get("totalTokenCount", 0),
            },
            latency_ms=latency,
        )

    def generate(
        self, prompt: str, max_tokens: Optional[int] = None, temperature: float = 0.0
    ) -> ProviderResponse:
        return asyncio.run(self.generate_async(prompt, max_tokens, temperature))


def get_openrouter_provider(
    api_key: str, model: str = None, **kwargs
) -> OpenRouterProvider:
    return OpenRouterProvider(api_key=api_key, model=model, **kwargs)


def get_github_models_provider(api_key: str, model: str = None) -> GitHubModelsProvider:
    return GitHubModelsProvider(api_key=api_key, model=model)


def get_bedrock_provider(
    access_key: str, secret_key: str, model: str = None, region: str = None, **kwargs
) -> AWSBedrockProvider:
    return AWSBedrockProvider(
        access_key=access_key,
        secret_key=secret_key,
        model=model,
        region=region,
        **kwargs,
    )


def get_vertex_provider(
    project_id: str, model: str = None, location: str = None, **kwargs
) -> VertexAIProvider:
    return VertexAIProvider(
        project_id=project_id, model=model, location=location, **kwargs
    )
